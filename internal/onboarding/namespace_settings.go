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

import onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"

const (
	podSecurityAuditLabel   = "pod-security.kubernetes.io/audit"
	podSecurityWarnLabel    = "pod-security.kubernetes.io/warn"
	podSecurityEnforceLabel = "pod-security.kubernetes.io/enforce"
)

// ApplyAdditionalSettingsLabels merges additionalSettings onto desired namespace labels.
// Custom labels are applied first; pod security and monitoring labels override on conflict.
func ApplyAdditionalSettingsLabels(labels map[string]string, settings *onboardingv1beta1.AdditionalSettingsSpec) {
	if settings == nil {
		ApplyUserMonitoringLabel(labels, nil)
		return
	}

	for _, label := range settings.AdditionalLabels {
		if label.Key == "" {
			continue
		}
		labels[label.Key] = label.Value
	}

	if settings.PodSecurityAudit != "" {
		labels[podSecurityAuditLabel] = settings.PodSecurityAudit
	}
	if settings.PodSecurityWarn != "" {
		labels[podSecurityWarnLabel] = settings.PodSecurityWarn
	}
	if settings.PodSecurityEnforce != "" {
		labels[podSecurityEnforceLabel] = settings.PodSecurityEnforce
	}

	ApplyUserMonitoringLabel(labels, settings)
}
