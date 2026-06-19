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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newOpenShiftTestClient(objs ...client.Object) (client.Client, *runtime.Scheme) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = onboardingv1beta1.AddToScheme(scheme)

	groupGV := schema.GroupVersion{Group: "user.openshift.io", Version: "v1"}
	egressGV := schema.GroupVersion{Group: "k8s.ovn.org", Version: "v1"}
	scheme.AddKnownTypeWithName(groupGV.WithKind("Group"), &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(groupGV.WithKind("GroupList"), &unstructured.UnstructuredList{})
	scheme.AddKnownTypeWithName(egressGV.WithKind("EgressIP"), &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(egressGV.WithKind("EgressIPList"), &unstructured.UnstructuredList{})

	builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...)
	for _, obj := range objs {
		if po, ok := obj.(*onboardingv1beta1.ProjectOnboarding); ok {
			builder = builder.WithStatusSubresource(po)
		}
	}
	return builder.Build(), scheme
}

func TestReconcileOpenShiftGroupAndEgressIP(t *testing.T) {
	t.Parallel()

	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-ocp"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name: "team-ocp-dev",
				LocalAdminGroup: &onboardingv1beta1.LocalAdminGroupSpec{
					Enabled: boolPtr(true),
					Users:   []string{"dev1"},
				},
				EgressIPs: &onboardingv1beta1.EgressIPSpec{
					Enabled: boolPtr(true),
					IPs:     []string{"203.0.113.10"},
				},
				DefaultPolicies: minimalDefaultPolicies(),
			}},
		},
	}

	c, scheme := newOpenShiftTestClient(po)
	ctx := context.Background()
	nsSpec := po.Spec.Namespaces[0]

	if err := ReconcileNamespace(ctx, c, scheme, po, nsSpec); err != nil {
		t.Fatalf("reconcile namespace: %v", err)
	}

	group := &unstructured.Unstructured{}
	group.SetGroupVersionKind(schema.GroupVersionKind{Group: "user.openshift.io", Version: "v1", Kind: "Group"})
	if err := c.Get(ctx, types.NamespacedName{Name: "team-ocp-dev-admins"}, group); err != nil {
		t.Fatalf("get group: %v", err)
	}
	users, _, _ := unstructured.NestedStringSlice(group.Object, "users")
	if len(users) != 1 || users[0] != "dev1" {
		t.Fatalf("unexpected group users: %v", users)
	}

	rb := &rbacv1.RoleBinding{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-ocp-dev", Name: "team-ocp-dev-rb"}, rb); err != nil {
		t.Fatalf("get role binding: %v", err)
	}

	egress := &unstructured.Unstructured{}
	egress.SetGroupVersionKind(schema.GroupVersionKind{Group: "k8s.ovn.org", Version: "v1", Kind: "EgressIP"})
	if err := c.Get(ctx, types.NamespacedName{Name: "team-ocp-dev"}, egress); err != nil {
		t.Fatalf("get egressip: %v", err)
	}
	ips, _, _ := unstructured.NestedStringSlice(egress.Object, "spec", "egressIPs")
	if len(ips) != 1 || ips[0] != "203.0.113.10" {
		t.Fatalf("unexpected egress IPs: %v", ips)
	}

	if err := CleanupNamespace(ctx, c, po, nsSpec); err != nil {
		t.Fatalf("cleanup namespace: %v", err)
	}

	if err := c.Get(ctx, types.NamespacedName{Name: "team-ocp-dev-admins"}, group); !apierrors.IsNotFound(err) {
		t.Fatalf("group should be deleted, got err=%v", err)
	}
	if err := c.Get(ctx, types.NamespacedName{Name: "team-ocp-dev"}, egress); !apierrors.IsNotFound(err) {
		t.Fatalf("egressip should be deleted, got err=%v", err)
	}
}

func TestReconcileAppliesTShirtSizeWithOverwrite(t *testing.T) {
	t.Parallel()

	cpu1 := "1"
	mem1 := testQuantity1Gi
	cpu2 := "2"

	tshirt := &onboardingv1beta1.TShirtSize{
		ObjectMeta: metav1.ObjectMeta{Name: "small"},
		Spec: onboardingv1beta1.TShirtSizeSpec{
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
				CPU:    &cpu1,
				Memory: &mem1,
			},
		},
	}

	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-size"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:            "team-gamma-dev",
				ProjectSize:     "small",
				OverwriteTshirt: boolPtr(true),
				ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
					Enabled: boolPtr(true),
					CPU:     &cpu2,
				},
				DefaultPolicies: minimalDefaultPolicies(),
			}},
		},
	}

	c, scheme := newOpenShiftTestClient(tshirt, po)
	ctx := context.Background()
	nsSpec := po.Spec.Namespaces[0]

	if err := ReconcileNamespace(ctx, c, scheme, po, nsSpec); err != nil {
		t.Fatalf("reconcile namespace: %v", err)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: "team-gamma-dev"}, ns); err != nil {
		t.Fatalf("get namespace: %v", err)
	}
	if ns.Labels["namespace-size"] != "small" {
		t.Fatalf("expected namespace-size=small, got %q", ns.Labels["namespace-size"])
	}

	quota := &corev1.ResourceQuota{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-gamma-dev", Name: "team-gamma-dev-quota"}, quota); err != nil {
		t.Fatalf("get resource quota: %v", err)
	}
	if q, ok := quota.Spec.Hard[corev1.ResourceCPU]; !ok || q.String() != "2" {
		t.Fatalf("expected cpu=2, got %v", quota.Spec.Hard[corev1.ResourceCPU])
	}
	if q, ok := quota.Spec.Hard[corev1.ResourceMemory]; !ok || q.String() != testQuantity1Gi {
		t.Fatalf("expected memory=1Gi from t-shirt, got %v", quota.Spec.Hard[corev1.ResourceMemory])
	}
}

func minimalDefaultPolicies() *onboardingv1beta1.DefaultPoliciesSpec {
	return &onboardingv1beta1.DefaultPoliciesSpec{
		AllowFromSameNamespace: boolPtr(true),
	}
}

func platformDefaultPolicies() *onboardingv1beta1.DefaultPoliciesSpec {
	return &onboardingv1beta1.DefaultPoliciesSpec{
		AllowFromMonitoring:    boolPtr(true),
		AllowKubeAPIServer:     boolPtr(true),
		AllowToDNS:             boolPtr(true),
		AllowFromSameNamespace: boolPtr(true),
	}
}
