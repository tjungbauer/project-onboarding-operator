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

package validation

import (
	"context"
	"strings"
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidateUniqueNamespaceNames(t *testing.T) {
	t.Parallel()

	err := validateUniqueNamespaceNames([]onboardingv1beta1.NamespaceSpec{
		{Name: "team-a"},
		{Name: "team-a"},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestValidateProjectOnboardingRequiresTShirtSize(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = onboardingv1beta1.AddToScheme(scheme)

	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:        "team-a",
				ProjectSize: "missing",
			}},
		},
	}

	v := NewValidator(fake.NewClientBuilder().WithScheme(scheme).WithObjects(po).Build())
	_, err := v.ValidateProjectOnboarding(context.Background(), po)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestTShirtSizeSpecHasSizing(t *testing.T) {
	t.Parallel()

	cpu := "1"
	if TShirtSizeSpecHasSizing(onboardingv1beta1.TShirtSizeSpec{
		ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{CPU: &cpu},
	}) {
		return
	}
	t.Fatal("expected sizing from resourceQuotas.cpu")
}

func TestValidateTShirtSizeDeleteBlockedWhenReferenced(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = onboardingv1beta1.AddToScheme(scheme)

	size := &onboardingv1beta1.TShirtSize{ObjectMeta: metav1.ObjectMeta{Name: "small"}}
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:        "team-a",
				ProjectSize: "small",
			}},
		},
	}

	v := NewValidator(fake.NewClientBuilder().WithScheme(scheme).WithObjects(size, po).Build())
	_, err := v.ValidateTShirtSizeDelete(context.Background(), size)
	if err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("expected referenced error, got %v", err)
	}
}
