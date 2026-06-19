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
	"github.com/prometheus/client_golang/prometheus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	projectOnboardingTenantsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "projectonboarding_tenants_total",
			Help: "Active tenant namespaces (enabled and not offboarded) per ProjectOnboarding.",
		},
		[]string{"project_onboarding"},
	)

	projectOnboardingReconcileErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "projectonboarding_reconcile_errors_total",
			Help: "Total ProjectOnboarding reconcile errors by reason.",
		},
		[]string{"reason"},
	)
)

func init() {
	metrics.Registry.MustRegister(projectOnboardingTenantsTotal, projectOnboardingReconcileErrorsTotal)
}

// ReconcileErrorReason classifies an error for metrics when no explicit reason is supplied.
func ReconcileErrorReason(err error) string {
	if err == nil {
		return ""
	}
	if IsTransientError(err) {
		return "transient"
	}
	return "reconcile"
}

// RecordReconcileError increments projectonboarding_reconcile_errors_total for reason.
func RecordReconcileError(reason string) {
	if reason == "" {
		reason = "unknown"
	}
	projectOnboardingReconcileErrorsTotal.WithLabelValues(reason).Inc()
}

// RecordAndReconcileError records a reconcile error metric and returns the standard result.
func RecordAndReconcileError(err error, reason string) (ctrl.Result, error) {
	if err != nil {
		if reason == "" {
			reason = ReconcileErrorReason(err)
		}
		RecordReconcileError(reason)
	}
	return ReconcileResultForError(err)
}

// SetActiveTenantCount updates projectonboarding_tenants_total for a ProjectOnboarding.
func SetActiveTenantCount(projectOnboarding string, count int) {
	projectOnboardingTenantsTotal.WithLabelValues(projectOnboarding).Set(float64(count))
}

// DeleteTenantMetrics removes per-ProjectOnboarding tenant metrics (e.g. after CR deletion).
func DeleteTenantMetrics(projectOnboarding string) {
	projectOnboardingTenantsTotal.DeleteLabelValues(projectOnboarding)
}
