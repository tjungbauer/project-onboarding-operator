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
	"context"
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLoadClusterDefaultsNotFoundReturnsNil(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	got, err := LoadClusterDefaults(context.Background(), c, DefaultOperatorNamespace)
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil defaults, got %+v", got)
	}
}

func TestLoadClusterDefaultsReadsGitOpsYAML(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultsConfigMapName,
			Namespace: DefaultOperatorNamespace,
		},
		Data: map[string]string{
			DefaultsConfigMapKey: `gitOps:
  applicationNamespace: gitops-apps
  allowedSourceRepos:
    - https://example.com/repo.git
`,
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cm).Build()

	got, err := LoadClusterDefaults(context.Background(), c, DefaultOperatorNamespace)
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if got == nil {
		t.Fatal("expected gitOps defaults")
	}
	if got.ApplicationNamespace != "gitops-apps" {
		t.Fatalf("applicationNamespace: want gitops-apps, got %q", got.ApplicationNamespace)
	}
	if len(got.AllowedSourceRepos) != 1 || got.AllowedSourceRepos[0] != "https://example.com/repo.git" {
		t.Fatalf("allowedSourceRepos: got %+v", got.AllowedSourceRepos)
	}
}

func TestMergeClusterDefaultsFillsMissingGitOpsFields(t *testing.T) {
	t.Parallel()

	defaults := &onboardingv1beta1.GitOpsSpec{
		ApplicationNamespace: "gitops-apps",
		AllowedSourceRepos:   []string{"https://example.com/repo.git"},
		Destinations: []onboardingv1beta1.GitOpsDestinationSpec{{
			Name:   "in-cluster",
			Server: "https://kubernetes.default.svc",
		}},
	}
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			GitOps: &onboardingv1beta1.GitOpsSpec{
				ApplicationNamespace: "tenant-gitops",
			},
		},
	}

	merged := MergeClusterDefaults(defaults, po)
	if merged.Spec.GitOps.ApplicationNamespace != "tenant-gitops" {
		t.Fatalf("CR applicationNamespace should win, got %q", merged.Spec.GitOps.ApplicationNamespace)
	}
	if len(merged.Spec.GitOps.AllowedSourceRepos) != 1 {
		t.Fatalf("expected defaults repos merged, got %+v", merged.Spec.GitOps.AllowedSourceRepos)
	}
	if len(merged.Spec.GitOps.Destinations) != 1 {
		t.Fatalf("expected defaults destinations merged, got %+v", merged.Spec.GitOps.Destinations)
	}
}

func TestMergeClusterDefaultsNilDefaultsReturnsOriginal(t *testing.T) {
	t.Parallel()

	po := &onboardingv1beta1.ProjectOnboarding{ObjectMeta: metav1.ObjectMeta{Name: "tenant"}}
	if got := MergeClusterDefaults(nil, po); got != po {
		t.Fatal("expected same pointer when defaults nil")
	}
}
