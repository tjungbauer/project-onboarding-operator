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

const userMonitoringLabel = "openshift.io/user-monitoring"

const (
	labelValueTrue  = "true"
	labelValueFalse = "false"
)

// ApplyUserMonitoringLabel sets openshift.io/user-monitoring on the tenant namespace.
// User workload monitoring is enabled by default (opt-out): nil or true sets "true";
// explicit false sets "false" rather than removing the label, because an absent label
// would opt back in to the OpenShift default (true).
func ApplyUserMonitoringLabel(labels map[string]string, settings *onboardingv1beta1.AdditionalSettingsSpec) {
	if settings == nil || settings.EnableClusterMonitoring == nil || *settings.EnableClusterMonitoring {
		labels[userMonitoringLabel] = labelValueTrue
		return
	}
	labels[userMonitoringLabel] = labelValueFalse
}
