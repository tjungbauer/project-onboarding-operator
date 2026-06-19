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

package webhook

import (
	"context"
	"strings"
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"github.com/tjungbauer/project-onboarding-operator/internal/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProjectOnboardingWebhookRejectsDuplicateNamespaces(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = onboardingv1beta1.AddToScheme(scheme)

	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{
				{Name: "team-a"},
				{Name: "team-a"},
			},
		},
	}

	v := &ProjectOnboardingCustomValidator{
		Validator: validation.NewValidator(fake.NewClientBuilder().WithScheme(scheme).Build()),
	}
	_, err := v.ValidateCreate(context.Background(), po)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate namespace error, got %v", err)
	}
}

func TestTShirtSizeWebhookRejectsEmptySizing(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = onboardingv1beta1.AddToScheme(scheme)

	size := &onboardingv1beta1.TShirtSize{
		ObjectMeta: metav1.ObjectMeta{Name: "small"},
		Spec:       onboardingv1beta1.TShirtSizeSpec{},
	}

	v := &TShirtSizeCustomValidator{
		Validator: validation.NewValidator(fake.NewClientBuilder().WithScheme(scheme).Build()),
	}
	_, err := v.ValidateCreate(context.Background(), size)
	if err == nil || !strings.Contains(err.Error(), "limit value") {
		t.Fatalf("expected sizing error, got %v", err)
	}
}

func TestTShirtSizeWebhookBlocksDeleteWhenReferenced(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = onboardingv1beta1.AddToScheme(scheme)

	size := &onboardingv1beta1.TShirtSize{
		ObjectMeta: metav1.ObjectMeta{Name: "small"},
		Spec: onboardingv1beta1.TShirtSizeSpec{
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
				Enabled: boolPtr(true),
				Pods:    int32Ptr(5),
			},
		},
	}
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:        "team-a",
				ProjectSize: "small",
			}},
		},
	}

	v := &TShirtSizeCustomValidator{
		Validator: validation.NewValidator(
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(size, po).Build(),
		),
	}
	_, err := v.ValidateDelete(context.Background(), size)
	if err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("expected referenced delete error, got %v", err)
	}
}

func boolPtr(v bool) *bool    { return &v }
func int32Ptr(v int32) *int32 { return &v }
