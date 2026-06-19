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
	"errors"
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestReconcileErrorReason(t *testing.T) {
	t.Parallel()

	if got := ReconcileErrorReason(nil); got != "" {
		t.Fatalf("nil: got %q", got)
	}
	if got := ReconcileErrorReason(errors.New("boom")); got != "reconcile" {
		t.Fatalf("generic: got %q", got)
	}
	conflict := apierrors.NewConflict(schema.GroupResource{Group: "", Resource: "pods"}, "x", errors.New("conflict"))
	if got := ReconcileErrorReason(conflict); got != "transient" {
		t.Fatalf("conflict: got %q", got)
	}
}

func TestActiveTenantCount(t *testing.T) {
	t.Parallel()

	disabled := false
	offboard := true
	namespaces := []onboardingv1beta1.NamespaceSpec{
		{Name: "a"},
		{Name: "b", Enabled: &disabled},
		{Name: "c", Offboard: &offboard},
		{Name: "d"},
	}
	if got := ActiveTenantCount(namespaces); got != 2 {
		t.Fatalf("ActiveTenantCount() = %d, want 2", got)
	}
}

func TestSetActiveTenantCount(t *testing.T) {
	t.Parallel()

	SetActiveTenantCount("test-po", 1)
	DeleteTenantMetrics("test-po")
}

func TestRecordReconcileErrorEmptyReason(t *testing.T) {
	t.Parallel()

	RecordReconcileError("")
}
