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
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testQuantity1Gi = "1Gi"

func TestIsOptInEnabled(t *testing.T) {
	t.Parallel()

	if IsOptInEnabled(nil) {
		t.Fatal("nil should default to disabled for opt-in features")
	}
	if IsOptInEnabled(boolPtr(false)) {
		t.Fatal("false should be disabled")
	}
	if !IsOptInEnabled(boolPtr(true)) {
		t.Fatal("true should be enabled")
	}
}
func TestIsEnabled(t *testing.T) {
	t.Parallel()

	if !IsEnabled(nil) {
		t.Fatal("nil enabled should default to true")
	}
	if IsEnabled(boolPtr(false)) {
		t.Fatal("false enabled should be disabled")
	}
	if !IsEnabled(boolPtr(true)) {
		t.Fatal("true enabled should be enabled")
	}
}

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	if got := SanitizeName("team_alpha"); got != "team-alpha" {
		t.Fatalf("expected team-alpha, got %q", got)
	}
}

func TestNormalizeQuantity(t *testing.T) {
	t.Parallel()

	if got := NormalizeQuantity("8gi"); got != "8Gi" {
		t.Fatalf("expected 8Gi, got %q", got)
	}
}

func TestManagedLabels(t *testing.T) {
	t.Parallel()

	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
	}
	nsSpec := onboardingv1beta1.NamespaceSpec{Name: "team-dev"}

	labels := ManagedLabels(po, nsSpec)
	if labels[onboardingv1beta1.ProjectOnboardingLabelKey] != "tenant-a" {
		t.Fatalf("unexpected project label: %v", labels)
	}
	if labels[onboardingv1beta1.ProjectOnboardingNamespaceKey] != "team-dev" {
		t.Fatalf("unexpected namespace label: %v", labels)
	}
}

func TestTruncateStatusMessage(t *testing.T) {
	t.Parallel()

	short := "ok"
	if got := TruncateStatusMessage(short); got != short {
		t.Fatalf("expected unchanged short message, got len=%d", len(got))
	}

	long := strings.Repeat("x", maxStatusMessageLen+10)
	got := TruncateStatusMessage(long)
	if len(got) > maxStatusMessageLen {
		t.Fatalf("truncated message too long: %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatal("expected ellipsis suffix")
	}
}

func TestGroupAndRoleBindingNames(t *testing.T) {
	t.Parallel()

	nsSpec := onboardingv1beta1.NamespaceSpec{Name: "team_dev"}
	if GroupName(nsSpec) != "team-dev-admins" {
		t.Fatalf("unexpected group name: %s", GroupName(nsSpec))
	}
	if RoleBindingName(nsSpec) != "team-dev-rb" {
		t.Fatalf("unexpected role binding name: %s", RoleBindingName(nsSpec))
	}
}
