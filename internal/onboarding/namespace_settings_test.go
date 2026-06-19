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

func TestApplyAdditionalSettingsLabelsCustomAndPodSecurity(t *testing.T) {
	t.Parallel()

	labels := map[string]string{}
	ApplyAdditionalSettingsLabels(labels, &onboardingv1beta1.AdditionalSettingsSpec{
		AdditionalLabels: []onboardingv1beta1.NamespaceLabel{
			{Key: "team", Value: "payments"},
			{Key: podSecurityEnforceLabel, Value: "should-not-win"},
		},
		PodSecurityEnforce: "restricted",
		PodSecurityWarn:    "baseline",
		PodSecurityAudit:   "privileged",
	})

	if labels["team"] != "payments" {
		t.Fatalf("expected custom label team=payments, got %q", labels["team"])
	}
	if labels[podSecurityEnforceLabel] != "restricted" {
		t.Fatalf("expected enforce=restricted, got %q", labels[podSecurityEnforceLabel])
	}
	if labels[podSecurityWarnLabel] != "baseline" {
		t.Fatalf("expected warn=baseline, got %q", labels[podSecurityWarnLabel])
	}
	if labels[podSecurityAuditLabel] != "privileged" {
		t.Fatalf("expected audit=privileged, got %q", labels[podSecurityAuditLabel])
	}
	if labels["openshift.io/user-monitoring"] != "true" {
		t.Fatalf("expected user-monitoring default true when enableClusterMonitoring is omitted, got %q", labels["openshift.io/user-monitoring"])
	}
}

func TestApplyAdditionalSettingsLabelsNilUsesMonitoringDefault(t *testing.T) {
	t.Parallel()

	labels := map[string]string{}
	ApplyAdditionalSettingsLabels(labels, nil)
	if labels["openshift.io/user-monitoring"] != "true" {
		t.Fatalf("expected user-monitoring default true, got %v", labels["openshift.io/user-monitoring"])
	}
}
