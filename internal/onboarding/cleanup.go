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
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CleanupNamespace deletes resources previously created for one namespace entry.
func CleanupNamespace(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) error {
	nsName := SanitizeName(nsSpec.Name)

	if err := deleteManagedNetworkPolicies(ctx, c, po, nsName); err != nil {
		return fmt.Errorf("network policies: %w", err)
	}

	if err := deleteManagedRoleBinding(ctx, c, po, nsSpec, nsName); err != nil {
		return fmt.Errorf("role binding: %w", err)
	}

	if err := deleteManagedLimitRange(ctx, c, po, nsName); err != nil {
		return fmt.Errorf("limit range: %w", err)
	}

	if err := deleteManagedResourceQuota(ctx, c, po, nsName); err != nil {
		return fmt.Errorf("resource quota: %w", err)
	}

	if nsSpec.EgressIPs != nil && IsOptInEnabled(nsSpec.EgressIPs.Enabled) {
		if err := deleteManagedEgressIP(ctx, c, po, nsName); err != nil {
			return fmt.Errorf("egress IP: %w", err)
		}
	}

	if nsSpec.LocalAdminGroup != nil && IsOptInEnabled(nsSpec.LocalAdminGroup.Enabled) {
		if err := deleteManagedGroup(ctx, c, po, nsSpec); err != nil {
			return fmt.Errorf("local admin group: %w", err)
		}
	}

	if err := deleteManagedAppProjects(ctx, c, po, nsSpec); err != nil {
		return fmt.Errorf("argo cd app project: %w", err)
	}

	if err := deleteManagedNamespace(ctx, c, po, nsName); err != nil {
		return fmt.Errorf("namespace: %w", err)
	}

	return nil
}

func managedListOptions(po *onboardingv1beta1.ProjectOnboarding) []client.ListOption {
	return []client.ListOption{
		client.MatchingLabels{
			onboardingv1beta1.ProjectOnboardingManagedByKey: onboardingv1beta1.ProjectOnboardingManagedByVal,
			onboardingv1beta1.ProjectOnboardingLabelKey:     po.Name,
		},
	}
}

func isManagedByProjectOnboarding(labels map[string]string, po *onboardingv1beta1.ProjectOnboarding) bool {
	if labels == nil {
		return false
	}
	return labels[onboardingv1beta1.ProjectOnboardingManagedByKey] == onboardingv1beta1.ProjectOnboardingManagedByVal &&
		labels[onboardingv1beta1.ProjectOnboardingLabelKey] == po.Name
}

func deleteIfManaged(ctx context.Context, c client.Client, obj client.Object, po *onboardingv1beta1.ProjectOnboarding) error {
	if !isManagedByProjectOnboarding(obj.GetLabels(), po) {
		return nil
	}
	if err := c.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func deleteManagedNetworkPolicies(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsName string) error {
	list := &networkingv1.NetworkPolicyList{}
	if err := c.List(ctx, list, append([]client.ListOption{client.InNamespace(nsName)}, managedListOptions(po)...)...); err != nil {
		return err
	}
	for i := range list.Items {
		if err := c.Delete(ctx, &list.Items[i]); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func deleteManagedRoleBinding(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec, nsName string) error {
	rb := &rbacv1.RoleBinding{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nsName, Name: RoleBindingName(nsSpec)}, rb)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return deleteIfManaged(ctx, c, rb, po)
}

func deleteManagedLimitRange(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsName string) error {
	lr := &corev1.LimitRange{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nsName, Name: nsName + "-limitrange"}, lr)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return deleteIfManaged(ctx, c, lr, po)
}

func deleteManagedResourceQuota(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsName string) error {
	quota := &corev1.ResourceQuota{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nsName, Name: nsName + "-quota"}, quota)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return deleteIfManaged(ctx, c, quota, po)
}

func deleteManagedEgressIP(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsName string) error {
	egress := &unstructured.Unstructured{}
	egress.SetGroupVersionKind(schema.GroupVersionKind{Group: "k8s.ovn.org", Version: "v1", Kind: "EgressIP"})

	err := c.Get(ctx, client.ObjectKey{Name: nsName}, egress)
	if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return deleteIfManaged(ctx, c, egress, po)
}

func deleteManagedGroup(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) error {
	group := &unstructured.Unstructured{}
	group.SetGroupVersionKind(schema.GroupVersionKind{Group: "user.openshift.io", Version: "v1", Kind: "Group"})

	err := c.Get(ctx, client.ObjectKey{Name: GroupName(nsSpec)}, group)
	if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return deleteIfManaged(ctx, c, group, po)
}

func deleteManagedNamespace(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsName string) error {
	ns := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{Name: nsName}, ns)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isManagedByProjectOnboarding(ns.Labels, po) {
		return nil
	}
	if err := c.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
