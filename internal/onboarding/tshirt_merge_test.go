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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestShouldMergeTshirtOverridesWhenOverwriteTrue(t *testing.T) {
	t.Parallel()

	nsSpec := onboardingv1beta1.NamespaceSpec{
		ProjectSize:     "small",
		OverwriteTshirt: boolPtr(true),
	}
	if !shouldMergeTshirtOverrides(nsSpec) {
		t.Fatal("expected merge when overwriteTshirt is true")
	}
}

func TestShouldNotMergeWhenInlineQuotaWithoutOverwrite(t *testing.T) {
	t.Parallel()

	cpu := "2"
	nsSpec := onboardingv1beta1.NamespaceSpec{
		ProjectSize: "small",
		ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
			Limits: &onboardingv1beta1.ResourceQuotaLimitSpec{CPU: &cpu},
		},
	}
	if shouldMergeTshirtOverrides(nsSpec) {
		t.Fatal("expected no merge without overwriteTshirt")
	}
}

func TestShouldMergeTshirtOverridesWhenTshirtOnly(t *testing.T) {
	t.Parallel()

	nsSpec := onboardingv1beta1.NamespaceSpec{ProjectSize: "small"}
	if shouldMergeTshirtOverrides(nsSpec) {
		t.Fatal("expected no merge for projectSize only")
	}
}

func TestResolveNamespaceSpecTenant3LimitsOnlyNoQuota(t *testing.T) {
	t.Parallel()

	cpu4 := "4"
	tshirt := &onboardingv1beta1.TShirtSize{
		ObjectMeta: metav1.ObjectMeta{Name: "small"},
		Spec: onboardingv1beta1.TShirtSizeSpec{
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{CPU: &cpu4},
			LimitRanges:    &onboardingv1beta1.LimitRangeSpec{Enabled: boolPtr(true)},
		},
	}

	overrideCPU := "10"
	limitEnabled := true
	nsSpec := onboardingv1beta1.NamespaceSpec{
		ProjectSize: "small",
		ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
			Limits: &onboardingv1beta1.ResourceQuotaLimitSpec{CPU: &overrideCPU},
		},
		LimitRanges: &onboardingv1beta1.LimitRangeSpec{Enabled: &limitEnabled},
	}

	scheme := newOnboardingTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tshirt).Build()

	resolved, err := ResolveNamespaceSpec(context.Background(), c, nsSpec)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.ResourceQuotas != nil {
		t.Fatalf("tenant3 pattern should not produce quota, got %+v", resolved.ResourceQuotas)
	}
	if resolved.LimitRanges == nil || !IsEnabled(resolved.LimitRanges.Enabled) {
		t.Fatal("expected limit ranges from t-shirt")
	}
}

func TestResolveNamespaceSpecMergesWhenOverwriteTshirt(t *testing.T) {
	t.Parallel()

	baseCPU := "1"
	baseMemory := testQuantity1Gi
	overrideCPU := "2"

	tshirt := &onboardingv1beta1.TShirtSize{
		ObjectMeta: metav1.ObjectMeta{Name: "small"},
		Spec: onboardingv1beta1.TShirtSizeSpec{
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
				CPU:    &baseCPU,
				Memory: &baseMemory,
			},
		},
	}

	nsSpec := onboardingv1beta1.NamespaceSpec{
		ProjectSize:     "small",
		OverwriteTshirt: boolPtr(true),
		ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
			Enabled: boolPtr(true),
			CPU:     &overrideCPU,
		},
	}

	scheme := newOnboardingTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tshirt).Build()

	resolved, err := ResolveNamespaceSpec(context.Background(), c, nsSpec)
	if err != nil {
		t.Fatalf("resolve namespace spec: %v", err)
	}
	if resolved.ResourceQuotas == nil || resolved.ResourceQuotas.CPU == nil || *resolved.ResourceQuotas.CPU != "2" {
		t.Fatalf("cpu override: got %+v", resolved.ResourceQuotas)
	}
	if resolved.ResourceQuotas.Memory == nil || *resolved.ResourceQuotas.Memory != testQuantity1Gi {
		t.Fatalf("memory from t-shirt: got %+v", resolved.ResourceQuotas.Memory)
	}
}

func TestResolveNamespaceSpecOverwriteWithoutEnabledProducesNothing(t *testing.T) {
	t.Parallel()

	cpu := "1"
	overrideCPU := "10"
	tshirt := &onboardingv1beta1.TShirtSize{
		ObjectMeta: metav1.ObjectMeta{Name: "small"},
		Spec: onboardingv1beta1.TShirtSizeSpec{
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
				Limits: &onboardingv1beta1.ResourceQuotaLimitSpec{CPU: &cpu},
			},
		},
	}

	nsSpec := onboardingv1beta1.NamespaceSpec{
		ProjectSize:     "small",
		OverwriteTshirt: boolPtr(true),
		ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
			Limits: &onboardingv1beta1.ResourceQuotaLimitSpec{CPU: &overrideCPU},
		},
	}

	scheme := newOnboardingTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tshirt).Build()

	resolved, err := ResolveNamespaceSpec(context.Background(), c, nsSpec)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.ResourceQuotas != nil {
		t.Fatalf("overwrite without enabled must not merge quota, got %+v", resolved.ResourceQuotas)
	}
}

func TestResolveNamespaceSpecIgnoresInlineQuotaWithoutOverwriteTshirt(t *testing.T) {
	t.Parallel()

	baseCPU := "1"
	overrideCPU := "2"

	tshirt := &onboardingv1beta1.TShirtSize{
		ObjectMeta: metav1.ObjectMeta{Name: "small"},
		Spec: onboardingv1beta1.TShirtSizeSpec{
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
				CPU: &baseCPU,
			},
		},
	}

	nsSpec := onboardingv1beta1.NamespaceSpec{
		ProjectSize: "small",
		ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
			Enabled: boolPtr(true),
			CPU:     &overrideCPU,
		},
	}

	scheme := newOnboardingTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tshirt).Build()

	resolved, err := ResolveNamespaceSpec(context.Background(), c, nsSpec)
	if err != nil {
		t.Fatalf("resolve namespace spec: %v", err)
	}
	if resolved.ResourceQuotas == nil || resolved.ResourceQuotas.CPU == nil || *resolved.ResourceQuotas.CPU != "1" {
		t.Fatalf("expected t-shirt cpu 1, inline ignored without overwriteTshirt: got %+v", resolved.ResourceQuotas)
	}
}

func TestMergeResourceQuotaSpecOverwritePartial(t *testing.T) {
	t.Parallel()

	baseCPU := "1"
	baseMemory := testQuantity1Gi
	overrideCPU := "2"

	base := &onboardingv1beta1.ResourceQuotaSpec{
		CPU:    &baseCPU,
		Memory: &baseMemory,
	}
	override := &onboardingv1beta1.ResourceQuotaSpec{
		CPU: &overrideCPU,
	}

	merged := mergeResourceQuotaSpec(base, override)
	if merged == nil {
		t.Fatal("expected merged quota")
	}
	if merged.CPU == nil || *merged.CPU != "2" {
		t.Fatalf("cpu: want 2, got %v", merged.CPU)
	}
	if merged.Memory == nil || *merged.Memory != testQuantity1Gi {
		t.Fatalf("memory: want 1Gi from t-shirt, got %v", merged.Memory)
	}
}

func TestMergeResourceQuotaSpecBaseOnly(t *testing.T) {
	t.Parallel()

	cpu := "1"
	base := &onboardingv1beta1.ResourceQuotaSpec{CPU: &cpu}

	merged := mergeResourceQuotaSpec(base, nil)
	if merged == nil || merged.CPU == nil || *merged.CPU != "1" {
		t.Fatalf("expected base cpu 1, got %+v", merged)
	}
	if merged.Enabled == nil || !*merged.Enabled {
		t.Fatal("expected enabled true")
	}
}
