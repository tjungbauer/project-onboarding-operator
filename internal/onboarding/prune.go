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
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PruneRemovedNamespaces deletes managed resources only for namespace entries explicitly marked
// offboard. Removing an entry from spec or deleting the ProjectOnboarding CR does not destroy
// tenant namespaces unless offboard=true or the namespace was deleted manually.
func PruneRemovedNamespaces(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding) error {
	desired := desiredTenantNamespaceNames(po)

	managedNamespaces, err := listManagedNamespaces(ctx, c, po)
	if err != nil {
		return err
	}

	for _, ns := range managedNamespaces {
		if _, ok := desired[ns.Name]; ok {
			continue
		}
		nsSpec := namespaceSpecForManagedName(po, ns.Name)
		if !IsOffboard(nsSpec.Offboard) {
			continue
		}
		if err := CleanupNamespace(ctx, c, po, nsSpec); err != nil {
			return fmt.Errorf("cleanup offboarded namespace %q: %w", ns.Name, err)
		}
	}
	return nil
}

func desiredTenantNamespaceNames(po *onboardingv1beta1.ProjectOnboarding) map[string]struct{} {
	out := make(map[string]struct{}, len(po.Spec.Namespaces))
	for _, nsSpec := range po.Spec.Namespaces {
		if IsOffboard(nsSpec.Offboard) {
			continue
		}
		out[SanitizeName(nsSpec.Name)] = struct{}{}
	}
	return out
}

func namespaceSpecForManagedName(po *onboardingv1beta1.ProjectOnboarding, sanitizedName string) onboardingv1beta1.NamespaceSpec {
	for _, nsSpec := range po.Spec.Namespaces {
		if SanitizeName(nsSpec.Name) == sanitizedName {
			return nsSpec
		}
	}
	return onboardingv1beta1.NamespaceSpec{Name: sanitizedName}
}

// PruneNamespaceResources removes managed objects that are no longer desired for an active namespace entry.
func PruneNamespaceResources(
	ctx context.Context,
	c client.Client,
	po *onboardingv1beta1.ProjectOnboarding,
	nsSpec onboardingv1beta1.NamespaceSpec,
	resolved onboardingv1beta1.NamespaceSpec,
) error {
	nsName := SanitizeName(nsSpec.Name)
	labels := ManagedLabels(po, nsSpec)

	if resolved.ResourceQuotas == nil || !IsEnabled(resolved.ResourceQuotas.Enabled) {
		if err := deleteManagedResourceQuota(ctx, c, po, nsName); err != nil {
			return err
		}
	}

	if resolved.LimitRanges == nil || !IsEnabled(resolved.LimitRanges.Enabled) {
		if err := deleteManagedLimitRange(ctx, c, po, nsName); err != nil {
			return err
		}
	}

	keepPolicies := desiredNetworkPolicyNames(nsSpec, nsName, labels)
	if err := pruneManagedNetworkPolicies(ctx, c, po, nsName, keepPolicies); err != nil {
		return fmt.Errorf("network policies: %w", err)
	}

	if nsSpec.LocalAdminGroup == nil || !IsOptInEnabled(nsSpec.LocalAdminGroup.Enabled) {
		if err := deleteManagedRoleBinding(ctx, c, po, nsSpec, nsName); err != nil {
			return err
		}
		if err := deleteManagedGroup(ctx, c, po, nsSpec); err != nil {
			return err
		}
	}

	if nsSpec.EgressIPs == nil || !IsOptInEnabled(nsSpec.EgressIPs.Enabled) {
		if err := deleteManagedEgressIP(ctx, c, po, nsName); err != nil {
			return err
		}
	}

	if err := pruneAppProjectsForNamespace(ctx, c, po, nsSpec, nsName); err != nil {
		return fmt.Errorf("argo cd app projects: %w", err)
	}

	return nil
}

func desiredNetworkPolicyNames(nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) map[string]struct{} {
	names := make(map[string]struct{})
	for _, pol := range defaultNetworkPolicies(nsSpec, nsName, labels) {
		names[pol.Name] = struct{}{}
	}
	for _, polSpec := range nsSpec.NetworkPolicies {
		if IsEnabled(polSpec.Active) {
			names[SanitizeName(polSpec.Name)] = struct{}{}
		}
	}
	return names
}

func pruneManagedNetworkPolicies(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsName string, keep map[string]struct{}) error {
	list := &networkingv1.NetworkPolicyList{}
	if err := c.List(ctx, list, append([]client.ListOption{client.InNamespace(nsName)}, managedListOptions(po)...)...); err != nil {
		return err
	}
	for i := range list.Items {
		if _, ok := keep[list.Items[i].Name]; ok {
			continue
		}
		if err := c.Delete(ctx, &list.Items[i]); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
