package onboarding

import (
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func TestBuildResourceQuotaHardRejectsLimitsStorage(t *testing.T) {
	t.Parallel()

	storage50 := "50Gi"
	hard := buildResourceQuotaHard(&onboardingv1beta1.ResourceQuotaSpec{
		Requests: &onboardingv1beta1.ResourceQuotaRequestSpec{
			Storage: &storage50,
		},
	})
	if _, ok := hard[corev1.ResourceName("limits.storage")]; ok {
		t.Fatal("limits.storage must not appear in ResourceQuota hard limits")
	}
	if got := hard[corev1.ResourceRequestsStorage]; got.String() != "50Gi" {
		t.Fatalf("requests.storage: got %q", got.String())
	}
}

func TestBuildResourceQuotaHardLimitsCPUAndMemory(t *testing.T) {
	t.Parallel()

	cpu := "10"
	memory := "4Gi"
	hard := buildResourceQuotaHard(&onboardingv1beta1.ResourceQuotaSpec{
		Limits: &onboardingv1beta1.ResourceQuotaLimitSpec{
			CPU:    &cpu,
			Memory: &memory,
		},
	})
	if got := hard[corev1.ResourceLimitsCPU]; got.String() != "10" {
		t.Fatalf("limits.cpu: got %q", got.String())
	}
	if got := hard[corev1.ResourceLimitsMemory]; got.String() != "4Gi" {
		t.Fatalf("limits.memory: got %q", got.String())
	}
}
