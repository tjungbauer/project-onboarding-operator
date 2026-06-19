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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsEnabled(enabled *bool) bool {
	if enabled == nil {
		return true
	}
	return *enabled
}

// IsOptInEnabled is for features disabled by default (e.g. egressIPs, offboard). Nil or false means off.
func IsOptInEnabled(enabled *bool) bool {
	return enabled != nil && *enabled
}

// IsOffboard reports whether a namespace entry should be torn down (managed resources + tenant namespace).
func IsOffboard(offboard *bool) bool {
	return IsOptInEnabled(offboard)
}

// ActiveTenantCount returns enabled, non-offboarded namespace entries in spec.
func ActiveTenantCount(namespaces []onboardingv1beta1.NamespaceSpec) int {
	count := 0
	for _, ns := range namespaces {
		if IsEnabled(ns.Enabled) && !IsOffboard(ns.Offboard) {
			count++
		}
	}
	return count
}

func IsDefaultTrue(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func NormalizeQuantity(value string) string {
	return strings.NewReplacer("gi", "Gi", "mi", "Mi", "GI", "Gi", "MI", "Mi").Replace(value)
}

func storageClassQuotasToMap(items []onboardingv1beta1.StorageClassQuota) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		if item.Key == "" {
			continue
		}
		out[item.Key] = item.Value
	}
	return out
}

func storageClassQuotasFromMap(values map[string]string) []onboardingv1beta1.StorageClassQuota {
	if len(values) == 0 {
		return nil
	}
	out := make([]onboardingv1beta1.StorageClassQuota, 0, len(values))
	for key, value := range values {
		out = append(out, onboardingv1beta1.StorageClassQuota{Key: key, Value: value})
	}
	return out
}

func ManagedLabels(po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) map[string]string {
	labels := map[string]string{
		onboardingv1beta1.ProjectOnboardingManagedByKey: onboardingv1beta1.ProjectOnboardingManagedByVal,
		onboardingv1beta1.ProjectOnboardingLabelKey:     po.Name,
		onboardingv1beta1.ProjectOnboardingNamespaceKey: nsSpec.Name,
	}
	return labels
}

func SanitizeName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

func GroupName(nsSpec onboardingv1beta1.NamespaceSpec) string {
	if nsSpec.LocalAdminGroup != nil && nsSpec.LocalAdminGroup.GroupName != "" {
		return SanitizeName(nsSpec.LocalAdminGroup.GroupName)
	}
	return SanitizeName(nsSpec.Name) + "-admins"
}

func RoleBindingName(nsSpec onboardingv1beta1.NamespaceSpec) string {
	if nsSpec.LocalAdminGroup != nil && nsSpec.LocalAdminGroup.GroupName != "" {
		return SanitizeName(nsSpec.LocalAdminGroup.GroupName)
	}
	return SanitizeName(nsSpec.Name) + "-rb"
}

func ClusterRoleName(nsSpec onboardingv1beta1.NamespaceSpec) string {
	if nsSpec.LocalAdminGroup != nil && nsSpec.LocalAdminGroup.ClusterRole != "" {
		return nsSpec.LocalAdminGroup.ClusterRole
	}
	return "admin"
}

func boolPtr(v bool) *bool { return &v }

const maxStatusMessageLen = 32000

// TruncateStatusMessage keeps condition/status messages within CRD validation limits.
func TruncateStatusMessage(message string) string {
	if len(message) <= maxStatusMessageLen {
		return message
	}
	return message[:maxStatusMessageLen-3] + "..."
}

func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i := range *conditions {
		if (*conditions)[i].Type == conditionType {
			if (*conditions)[i].Status == status && (*conditions)[i].Reason == reason && (*conditions)[i].Message == message {
				return
			}
			(*conditions)[i].Status = status
			(*conditions)[i].Reason = reason
			(*conditions)[i].Message = message
			(*conditions)[i].LastTransitionTime = now
			return
		}
	}
	*conditions = append(*conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}
