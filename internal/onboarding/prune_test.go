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
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPruneRemovedNamespacesKeepsOrphanManagedNamespace(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:            "team-active",
				DefaultPolicies: minimalDefaultPolicies(),
			}},
		},
	}

	orphanLabels := ManagedLabels(po, onboardingv1beta1.NamespaceSpec{Name: "team-removed"})
	orphanNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "team-removed",
			Labels: orphanLabels,
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po, orphanNS).WithStatusSubresource(po).Build()
	ctx := context.Background()

	if err := PruneRemovedNamespaces(ctx, c, po); err != nil {
		t.Fatalf("prune removed namespaces: %v", err)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: "team-removed"}, ns); err != nil {
		t.Fatalf("orphan namespace should be kept, got err=%v", err)
	}
}

func TestPruneRemovedNamespacesKeepsFrozenNamespace(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	disabled := false
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:    "team-disabled",
				Enabled: &disabled,
			}},
		},
	}

	activeLabels := ManagedLabels(po, onboardingv1beta1.NamespaceSpec{Name: "team-disabled"})
	managedNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "team-disabled",
			Labels: activeLabels,
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po, managedNS).WithStatusSubresource(po).Build()
	ctx := context.Background()

	if err := PruneRemovedNamespaces(ctx, c, po); err != nil {
		t.Fatalf("prune removed namespaces: %v", err)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: "team-disabled"}, ns); err != nil {
		t.Fatalf("frozen namespace should be kept, got err=%v", err)
	}
}

func TestPruneRemovedNamespacesPrunesOffboardedNamespace(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:     "team-offboard",
				Offboard: boolPtr(true),
			}},
		},
	}

	activeLabels := ManagedLabels(po, onboardingv1beta1.NamespaceSpec{Name: "team-offboard"})
	managedNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "team-offboard",
			Labels: activeLabels,
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po, managedNS).WithStatusSubresource(po).Build()
	ctx := context.Background()

	if err := PruneRemovedNamespaces(ctx, c, po); err != nil {
		t.Fatalf("prune removed namespaces: %v", err)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: "team-offboard"}, ns); !apierrors.IsNotFound(err) {
		t.Fatalf("offboarded namespace should be pruned, got err=%v", err)
	}
}

func TestPruneNamespaceResourcesRemovesQuotaWhenDisabled(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:            "team-a",
				DefaultPolicies: minimalDefaultPolicies(),
			}},
		},
	}
	nsSpec := po.Spec.Namespaces[0]
	resolved := nsSpec.DeepCopy()
	resolved.ResourceQuotas = &onboardingv1beta1.ResourceQuotaSpec{Enabled: boolPtr(false)}

	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "team-a-quota",
			Namespace: "team-a",
			Labels:    ManagedLabels(po, nsSpec),
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po, quota).WithStatusSubresource(po).Build()
	ctx := context.Background()

	if err := PruneNamespaceResources(ctx, c, po, nsSpec, *resolved); err != nil {
		t.Fatalf("prune namespace resources: %v", err)
	}

	got := &corev1.ResourceQuota{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-a", Name: "team-a-quota"}, got); !apierrors.IsNotFound(err) {
		t.Fatalf("resource quota should be deleted, got err=%v", err)
	}
}

func newOnboardingTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{
		corev1.AddToScheme,
		networkingv1.AddToScheme,
		rbacv1.AddToScheme,
		onboardingv1beta1.AddToScheme,
	} {
		if err := add(scheme); err != nil {
			t.Fatalf("add scheme: %v", err)
		}
	}
	return scheme
}
