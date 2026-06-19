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

func TestValidateByteQuantityRejectsBareInteger(t *testing.T) {
	t.Parallel()

	err := validateByteQuantity("spec.resourceQuotas.memory", "4")
	if err == nil || !strings.Contains(err.Error(), "unit suffix") {
		t.Fatalf("expected unit suffix error, got %v", err)
	}
}

func TestValidateByteQuantityAcceptsGi(t *testing.T) {
	t.Parallel()

	if err := validateByteQuantity("spec.resourceQuotas.memory", "4Gi"); err != nil {
		t.Fatalf("expected 4Gi to be valid, got %v", err)
	}
	if err := validateByteQuantity("spec.resourceQuotas.memory", "4gi"); err != nil {
		t.Fatalf("expected normalized 4gi to be valid, got %v", err)
	}
}

func TestValidateCPUQuantityAcceptsBareInteger(t *testing.T) {
	t.Parallel()

	if err := validateCPUQuantity("spec.resourceQuotas.cpu", "4"); err != nil {
		t.Fatalf("expected cpu=4 to be valid, got %v", err)
	}
}

func TestValidateProjectOnboardingRejectsBareMemory(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = onboardingv1beta1.AddToScheme(scheme)

	mem := "4"
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name: "team-a",
				ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
					Memory: &mem,
				},
			}},
		},
	}

	v := NewValidator(fake.NewClientBuilder().WithScheme(scheme).WithObjects(po).Build())
	_, err := v.ValidateProjectOnboarding(context.Background(), po)
	if err == nil || !strings.Contains(err.Error(), "unit suffix") {
		t.Fatalf("expected unit suffix error, got %v", err)
	}
}

func TestValidateTShirtSizeRejectsBareStorage(t *testing.T) {
	t.Parallel()

	storage := "20"
	size := &onboardingv1beta1.TShirtSize{
		ObjectMeta: metav1.ObjectMeta{Name: "small"},
		Spec: onboardingv1beta1.TShirtSizeSpec{
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
				Requests: &onboardingv1beta1.ResourceQuotaRequestSpec{
					Storage: &storage,
				},
			},
		},
	}

	v := NewValidator(fake.NewClientBuilder().Build())
	_, err := v.ValidateTShirtSize(context.Background(), size)
	if err == nil || !strings.Contains(err.Error(), "unit suffix") {
		t.Fatalf("expected unit suffix error, got %v", err)
	}
}

func TestValidateStorageClassCountValue(t *testing.T) {
	t.Parallel()

	err := validateStorageClassQuotaValue(
		"spec.resourceQuotas.storageClasses[0].value",
		"bronze.storageclass.storage.k8s.io/persistentvolumeclaims",
		"10",
	)
	if err != nil {
		t.Fatalf("expected count 10 to be valid, got %v", err)
	}

	err = validateStorageClassQuotaValue(
		"spec.resourceQuotas.storageClasses[0].value",
		"bronze.storageclass.storage.k8s.io/requests.storage",
		"10",
	)
	if err == nil || !strings.Contains(err.Error(), "unit suffix") {
		t.Fatalf("expected storage value to require suffix, got %v", err)
	}
}
