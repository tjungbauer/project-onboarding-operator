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
	"strings"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

const PauseReconciliationAnnotation = "onboarding.stderr.at/pause-reconciliation"

// IsReconciliationPaused reports whether the ProjectOnboarding should skip reconcile work.
func IsReconciliationPaused(po *onboardingv1beta1.ProjectOnboarding) bool {
	if po == nil {
		return false
	}
	v, ok := po.Annotations[PauseReconciliationAnnotation]
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
