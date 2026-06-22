package validation

import (
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

func FuzzNamespaceSpecName(f *testing.F) {
	f.Add("team-payments-dev")
	f.Add("small")
	f.Add("")
	f.Fuzz(func(t *testing.T, name string) {
		spec := onboardingv1beta1.NamespaceSpec{Name: name}
		if spec.Name == "" {
			return
		}
		_ = spec.Name
	})
}

func FuzzTShirtSizeResourceQuotas(f *testing.F) {
	f.Add(int32(1))
	f.Fuzz(func(t *testing.T, pods int32) {
		if pods < 0 {
			return
		}
		spec := onboardingv1beta1.TShirtSizeSpec{
			ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
				Pods: &pods,
			},
		}
		if spec.ResourceQuotas.Pods != nil && *spec.ResourceQuotas.Pods < 0 {
			t.Fatalf("negative pods")
		}
	})
}
