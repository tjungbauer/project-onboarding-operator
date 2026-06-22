package onboarding

import (
	"context"
	"fmt"
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Exercises reconcile scheduling with many namespace entries (no live API server).
func TestReconcileManyNamespaceEntries(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("corev1 AddToScheme: %v", err)
	}
	if err := networkingv1.AddToScheme(scheme); err != nil {
		t.Fatalf("networkingv1 AddToScheme: %v", err)
	}
	if err := rbacv1.AddToScheme(scheme); err != nil {
		t.Fatalf("rbacv1 AddToScheme: %v", err)
	}
	if err := onboardingv1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}

	namespaces := make([]onboardingv1beta1.NamespaceSpec, 0, 32)
	for i := 0; i < 32; i++ {
		enabled := true
		pods := int32(5)
		namespaces = append(namespaces, onboardingv1beta1.NamespaceSpec{
			Name:    fmt.Sprintf("tenant-%02d", i),
			Enabled: &enabled,
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
				Pods:   &pods,
				CPU:    strPtr("500m"),
				Memory: strPtr("512Mi"),
			},
		})
	}

	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "load-test"},
		Spec:       onboardingv1beta1.ProjectOnboardingSpec{Namespaces: namespaces},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po).Build()
	ctx := context.Background()

	for _, nsSpec := range po.Spec.Namespaces {
		if err := ReconcileNamespace(ctx, cl, scheme, po, nsSpec); err != nil {
			t.Fatalf("ReconcileNamespace %s: %v", nsSpec.Name, err)
		}
	}
}

func strPtr(v string) *string { return &v }
