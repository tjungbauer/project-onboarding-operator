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
	"sort"
	"strings"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const deletionBlockedMessagePrefix = "Cannot delete ProjectOnboarding while managed tenant namespaces still exist. " +
	"Set offboard=true on each namespace entry or delete the tenant namespaces manually. Pending: "

// FinalizeProjectOnboardingDeletion tears down namespace entries that were explicitly offboarded
// or whose tenant namespace was already removed manually. It returns complete=false while any
// operator-managed tenant namespace still exists without offboard=true.
func FinalizeProjectOnboardingDeletion(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding) (complete bool, pendingMessage string, err error) {
	pending := make([]string, 0)

	for _, nsSpec := range po.Spec.Namespaces {
		nsPending, err := finalizeNamespaceEntry(ctx, c, po, nsSpec)
		if err != nil {
			return false, "", err
		}
		if nsPending != "" {
			pending = append(pending, nsPending)
		}
	}

	specNames := namespaceNamesInSpec(po)
	managedNamespaces, err := listManagedNamespaces(ctx, c, po)
	if err != nil {
		return false, "", err
	}
	for _, ns := range managedNamespaces {
		if _, inSpec := specNames[ns.Name]; inSpec {
			continue
		}
		if !ns.DeletionTimestamp.IsZero() {
			continue
		}
		pending = append(pending, ns.Name)
	}

	if err := cleanupClusterScopedForReleasedTenants(ctx, c, po); err != nil {
		return false, "", err
	}

	if len(pending) > 0 {
		sort.Strings(pending)
		return false, deletionBlockedMessagePrefix + strings.Join(pending, ", "), nil
	}
	return true, "", nil
}

func finalizeNamespaceEntry(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) (pendingName string, err error) {
	nsName := SanitizeName(nsSpec.Name)
	exists, err := managedNamespaceExists(ctx, c, po, nsName)
	if err != nil {
		return "", err
	}

	if IsOffboard(nsSpec.Offboard) {
		if err := CleanupNamespace(ctx, c, po, nsSpec); err != nil {
			return "", err
		}
		return "", nil
	}

	if exists {
		return nsName, nil
	}

	if err := CleanupNamespaceClusterScoped(ctx, c, po, nsSpec); err != nil {
		return "", err
	}
	return "", nil
}

func managedNamespaceExists(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsName string) (bool, error) {
	ns := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{Name: nsName}, ns)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !isManagedByProjectOnboarding(ns.Labels, po) {
		return false, nil
	}
	if !ns.DeletionTimestamp.IsZero() {
		return false, nil
	}
	return true, nil
}

func listManagedNamespaces(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding) ([]corev1.Namespace, error) {
	list := &corev1.NamespaceList{}
	if err := c.List(ctx, list, client.MatchingLabels{
		onboardingv1beta1.ProjectOnboardingManagedByKey: onboardingv1beta1.ProjectOnboardingManagedByVal,
		onboardingv1beta1.ProjectOnboardingLabelKey:     po.Name,
	}); err != nil {
		return nil, fmt.Errorf("list managed namespaces: %w", err)
	}
	return list.Items, nil
}

func namespaceNamesInSpec(po *onboardingv1beta1.ProjectOnboarding) map[string]struct{} {
	out := make(map[string]struct{}, len(po.Spec.Namespaces))
	for _, nsSpec := range po.Spec.Namespaces {
		out[SanitizeName(nsSpec.Name)] = struct{}{}
	}
	return out
}

// CleanupNamespaceClusterScoped removes cluster-scoped operator resources for a tenant
// namespace that was deleted manually (namespace-scoped objects are already gone).
func CleanupNamespaceClusterScoped(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) error {
	nsName := SanitizeName(nsSpec.Name)

	if err := deleteManagedEgressIP(ctx, c, po, nsName); err != nil {
		return fmt.Errorf("egress IP: %w", err)
	}
	if err := deleteManagedGroup(ctx, c, po, nsSpec); err != nil {
		return fmt.Errorf("local admin group: %w", err)
	}
	if err := deleteManagedAppProjects(ctx, c, po, nsSpec); err != nil {
		return fmt.Errorf("argo cd app project: %w", err)
	}
	return nil
}

func cleanupClusterScopedForReleasedTenants(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding) error {
	tenantNames := make(map[string]struct{})

	for _, nsSpec := range po.Spec.Namespaces {
		tenantNames[SanitizeName(nsSpec.Name)] = struct{}{}
	}
	managedNamespaces, err := listManagedNamespaces(ctx, c, po)
	if err != nil {
		return err
	}
	for _, ns := range managedNamespaces {
		tenantNames[ns.Name] = struct{}{}
		if label := ns.Labels[onboardingv1beta1.ProjectOnboardingNamespaceKey]; label != "" {
			tenantNames[SanitizeName(label)] = struct{}{}
		}
	}

	for tenantName := range tenantNames {
		exists, err := namespaceExists(ctx, c, tenantName)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := CleanupNamespaceClusterScoped(ctx, c, po, onboardingv1beta1.NamespaceSpec{Name: tenantName}); err != nil {
			return err
		}
	}
	return nil
}

func namespaceExists(ctx context.Context, c client.Client, name string) (bool, error) {
	ns := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{Name: name}, ns)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
