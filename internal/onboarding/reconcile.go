/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package onboarding

import (
	"context"
	"fmt"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ReconcileNamespace ensures all onboarding resources exist for one namespace entry.
func ReconcileNamespace(ctx context.Context, c client.Client, scheme *runtime.Scheme, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) error {
	resolved, err := ResolveNamespaceSpec(ctx, c, nsSpec)
	if err != nil {
		return err
	}

	nsName := SanitizeName(nsSpec.Name)
	labels := ManagedLabels(po, nsSpec)

	tenantNS, err := reconcileNamespaceObject(ctx, c, scheme, po, nsSpec, nsName, labels)
	if err != nil {
		return fmt.Errorf("namespace: %w", err)
	}

	if resolved.ResourceQuotas != nil && IsEnabled(resolved.ResourceQuotas.Enabled) {
		if err := reconcileResourceQuota(ctx, c, scheme, tenantNS, resolved, nsName, labels); err != nil {
			return fmt.Errorf("resource quota: %w", err)
		}
	}

	if resolved.LimitRanges != nil && IsEnabled(resolved.LimitRanges.Enabled) {
		if err := reconcileLimitRange(ctx, c, scheme, tenantNS, resolved, nsName, labels); err != nil {
			return fmt.Errorf("limit range: %w", err)
		}
	}

	if err := reconcileDefaultNetworkPolicies(ctx, c, scheme, tenantNS, nsSpec, nsName, labels); err != nil {
		return fmt.Errorf("default network policies: %w", err)
	}

	if err := reconcileCustomNetworkPolicies(ctx, c, scheme, tenantNS, nsSpec, nsName, labels); err != nil {
		return fmt.Errorf("custom network policies: %w", err)
	}

	if nsSpec.LocalAdminGroup != nil && IsOptInEnabled(nsSpec.LocalAdminGroup.Enabled) {
		if err := reconcileLocalAdminGroup(ctx, c, scheme, po, nsSpec, labels); err != nil {
			return fmt.Errorf("local admin group: %w", err)
		}
		if err := reconcileLocalAdminRoleBinding(ctx, c, scheme, tenantNS, nsSpec, nsName, labels); err != nil {
			return fmt.Errorf("local admin role binding: %w", err)
		}
	}

	if nsSpec.EgressIPs != nil && IsOptInEnabled(nsSpec.EgressIPs.Enabled) {
		if err := reconcileEgressIP(ctx, c, scheme, po, nsSpec, nsName, labels); err != nil {
			return fmt.Errorf("egress IP: %w", err)
		}
	}

	if err := reconcileArgoCDProjects(ctx, c, scheme, po, nsSpec, nsName, labels); err != nil {
		return fmt.Errorf("argo cd app project: %w", err)
	}

	if err := PruneNamespaceResources(ctx, c, po, nsSpec, resolved); err != nil {
		return fmt.Errorf("prune: %w", err)
	}

	return nil
}

func reconcileNamespaceObject(ctx context.Context, c client.Client, scheme *runtime.Scheme, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{Name: nsName}, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	desiredLabels := map[string]string{}
	for k, v := range labels {
		desiredLabels[k] = v
	}
	for k, v := range nsSpec.Labels {
		desiredLabels[k] = v
	}

	ApplyAdditionalSettingsLabels(desiredLabels, nsSpec.AdditionalSettings)

	if nsSpec.ProjectSize != "" {
		desiredLabels["namespace-size"] = nsSpec.ProjectSize
	}

	if nsSpec.EgressIPs != nil && IsOptInEnabled(nsSpec.EgressIPs.Enabled) {
		desiredLabels["env"] = nsName
	}

	if managedBy := gitOpsManagedByLabel(po, nsSpec); managedBy != "" {
		desiredLabels[onboardingv1beta1.ArgoCDManagedByLabel] = managedBy
	}

	desiredAnnotations := map[string]string{}
	for k, v := range nsSpec.Annotations {
		desiredAnnotations[k] = v
	}

	if apierrors.IsNotFound(err) {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        nsName,
				Labels:      desiredLabels,
				Annotations: desiredAnnotations,
			},
		}
		if err := ensureProjectOnboardingControllerRef(scheme, po, ns); err != nil {
			return nil, err
		}
		if err := c.Create(ctx, ns); err != nil {
			return nil, err
		}
		return ns, nil
	}

	patch := client.MergeFrom(ns.DeepCopy())
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	for k, v := range desiredLabels {
		ns.Labels[k] = v
	}
	if ns.Annotations == nil {
		ns.Annotations = map[string]string{}
	}
	for k, v := range desiredAnnotations {
		ns.Annotations[k] = v
	}
	if err := ensureProjectOnboardingControllerRef(scheme, po, ns); err != nil {
		return nil, err
	}
	if err := c.Patch(ctx, ns, patch); err != nil {
		return nil, err
	}
	return ns, nil
}

// EnsureFinalizer adds the project onboarding finalizer when missing.
func EnsureFinalizer(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding) error {
	if controllerutil.ContainsFinalizer(po, onboardingv1beta1.ProjectOnboardingFinalizer) {
		return nil
	}
	controllerutil.AddFinalizer(po, onboardingv1beta1.ProjectOnboardingFinalizer)
	return c.Update(ctx, po)
}

// RemoveFinalizer drops the project onboarding finalizer.
func RemoveFinalizer(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding) error {
	if !controllerutil.ContainsFinalizer(po, onboardingv1beta1.ProjectOnboardingFinalizer) {
		return nil
	}
	controllerutil.RemoveFinalizer(po, onboardingv1beta1.ProjectOnboardingFinalizer)
	return c.Update(ctx, po)
}
