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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileAndCleanupNamespace(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := networkingv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add networking scheme: %v", err)
	}
	if err := rbacv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add rbac scheme: %v", err)
	}
	if err := onboardingv1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("add onboarding scheme: %v", err)
	}

	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name: "team-test",
				ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
					Enabled: boolPtr(true),
					Pods:    int32Ptr(5),
				},
				DefaultPolicies: &onboardingv1beta1.DefaultPoliciesSpec{
					AllowFromIngress:       boolPtr(false),
					AllowFromMonitoring:    boolPtr(false),
					AllowKubeAPIServer:     boolPtr(false),
					AllowToDNS:             boolPtr(false),
					AllowFromSameNamespace: boolPtr(true),
					DenyAllEgress:          boolPtr(false),
					DenyAllIngress:         boolPtr(false),
				},
			}},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po).WithStatusSubresource(po).Build()
	ctx := context.Background()
	nsSpec := po.Spec.Namespaces[0]

	if err := ReconcileNamespace(ctx, c, scheme, po, nsSpec); err != nil {
		t.Fatalf("reconcile namespace: %v", err)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: "team-test"}, ns); err != nil {
		t.Fatalf("get namespace: %v", err)
	}
	if !isManagedByProjectOnboarding(ns.Labels, po) {
		t.Fatal("namespace should have managed labels")
	}
	if len(ns.OwnerReferences) == 0 || ns.OwnerReferences[0].Name != po.Name {
		t.Fatalf("namespace should be owned by ProjectOnboarding %q", po.Name)
	}

	quota := &corev1.ResourceQuota{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-test", Name: "team-test-quota"}, quota); err != nil {
		t.Fatalf("get resource quota: %v", err)
	}
	if len(quota.OwnerReferences) == 0 || quota.OwnerReferences[0].Name != ns.Name {
		t.Fatalf("resource quota should be owned by tenant namespace %q", ns.Name)
	}

	npList := &networkingv1.NetworkPolicyList{}
	if err := c.List(ctx, npList, client.InNamespace("team-test")); err != nil {
		t.Fatalf("list network policies: %v", err)
	}
	if len(npList.Items) != 1 {
		t.Fatalf("expected 1 network policy, got %d", len(npList.Items))
	}

	if err := CleanupNamespace(ctx, c, po, nsSpec); err != nil {
		t.Fatalf("cleanup namespace: %v", err)
	}

	if err := c.Get(ctx, types.NamespacedName{Name: "team-test"}, ns); !apierrors.IsNotFound(err) {
		t.Fatalf("namespace should be deleted, got err=%v", err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-test", Name: "team-test-quota"}, quota); !apierrors.IsNotFound(err) {
		t.Fatalf("resource quota should be deleted, got err=%v", err)
	}
	npList = &networkingv1.NetworkPolicyList{}
	if err := c.List(ctx, npList, client.InNamespace("team-test")); err != nil {
		t.Fatalf("list network policies after cleanup: %v", err)
	}
	if len(npList.Items) != 0 {
		t.Fatalf("expected 0 network policies after cleanup, got %d", len(npList.Items))
	}
}

func int32Ptr(v int32) *int32 { return &v }
