package v1alpha1

import (
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

func TestProjectOnboardingConvertToFromHub(t *testing.T) {
	t.Parallel()

	spoke := &ProjectOnboarding{}
	spoke.Name = "tenant-a"
	spoke.Spec.Namespaces = []NamespaceSpec{{Name: "tenant-a-dev", Enabled: boolPtr(true)}}

	hub := &onboardingv1beta1.ProjectOnboarding{}
	if err := spoke.ConvertTo(hub); err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}
	if hub.Name != spoke.Name || len(hub.Spec.Namespaces) != 1 {
		t.Fatalf("unexpected hub after ConvertTo: %+v", hub)
	}

	roundTrip := &ProjectOnboarding{}
	if err := roundTrip.ConvertFrom(hub); err != nil {
		t.Fatalf("ConvertFrom: %v", err)
	}
	if roundTrip.APIVersion != GroupVersion.String() {
		t.Fatalf("apiVersion after ConvertFrom = %q, want %q", roundTrip.APIVersion, GroupVersion.String())
	}
	if roundTrip.Spec.Namespaces[0].Name != "tenant-a-dev" {
		t.Fatalf("unexpected spoke after ConvertFrom: %+v", roundTrip.Spec)
	}
}

func TestTShirtSizeConvertToFromHub(t *testing.T) {
	t.Parallel()

	spoke := &TShirtSize{}
	spoke.Name = "S"
	spoke.Spec.ResourceQuotas = &ResourceQuotaSpec{Enabled: boolPtr(true), Pods: int32Ptr(10)}

	hub := &onboardingv1beta1.TShirtSize{}
	if err := spoke.ConvertTo(hub); err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}

	roundTrip := &TShirtSize{}
	if err := roundTrip.ConvertFrom(hub); err != nil {
		t.Fatalf("ConvertFrom: %v", err)
	}
	if roundTrip.Spec.ResourceQuotas == nil || *roundTrip.Spec.ResourceQuotas.Pods != 10 {
		t.Fatalf("unexpected spoke after ConvertFrom: %+v", roundTrip.Spec)
	}
}

func boolPtr(v bool) *bool { return &v }

func int32Ptr(v int32) *int32 { return &v }
