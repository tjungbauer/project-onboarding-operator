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
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

func TestApplyUserMonitoringLabelExplicitFalse(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"openshift.io/user-monitoring": labelValueTrue}
	disabled := false
	ApplyUserMonitoringLabel(labels, &onboardingv1beta1.AdditionalSettingsSpec{
		EnableClusterMonitoring: &disabled,
	})
	if labels["openshift.io/user-monitoring"] != labelValueFalse {
		t.Fatalf("expected user-monitoring=false when disabled, got %q", labels["openshift.io/user-monitoring"])
	}
}

func TestApplyUserMonitoringLabelExplicitTrue(t *testing.T) {
	t.Parallel()

	labels := map[string]string{}
	enabled := true
	ApplyUserMonitoringLabel(labels, &onboardingv1beta1.AdditionalSettingsSpec{
		EnableClusterMonitoring: &enabled,
	})
	if labels["openshift.io/user-monitoring"] != labelValueTrue {
		t.Fatalf("expected user-monitoring=true when enabled, got %q", labels["openshift.io/user-monitoring"])
	}
}

func TestIsReconciliationPaused(t *testing.T) {
	t.Parallel()

	po := &onboardingv1beta1.ProjectOnboarding{}
	if IsReconciliationPaused(po) {
		t.Fatal("expected not paused")
	}
	po.Annotations = map[string]string{PauseReconciliationAnnotation: "true"}
	if !IsReconciliationPaused(po) {
		t.Fatal("expected paused")
	}
}

func TestShouldApplyTshirtResourceQuotasRequiresEnabled(t *testing.T) {
	t.Parallel()

	cpu := "10"
	ns := onboardingv1beta1.NamespaceSpec{
		ProjectSize: "small",
		ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
			Limits: &onboardingv1beta1.ResourceQuotaLimitSpec{CPU: &cpu},
		},
	}
	if shouldApplyTshirtResourceQuotas(ns) {
		t.Fatal("expected false without resourceQuotas.enabled")
	}
	enabled := true
	ns.ResourceQuotas.Enabled = &enabled
	if !shouldApplyTshirtResourceQuotas(ns) {
		t.Fatal("expected true with resourceQuotas.enabled")
	}
}

func TestShouldApplyTshirtLimitRangesRequiresExplicitEnabled(t *testing.T) {
	t.Parallel()

	limitEnabled := true
	ns := onboardingv1beta1.NamespaceSpec{
		ProjectSize: "small",
		LimitRanges: &onboardingv1beta1.LimitRangeSpec{Enabled: &limitEnabled},
	}
	if !shouldApplyTshirtLimitRanges(ns) {
		t.Fatal("expected true with limitRanges.enabled")
	}

	disabled := false
	ns.LimitRanges.Enabled = &disabled
	if shouldApplyTshirtLimitRanges(ns) {
		t.Fatal("expected false with limitRanges.enabled false")
	}
}
