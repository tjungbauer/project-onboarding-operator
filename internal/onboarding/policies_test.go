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
	"sort"
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

func TestDefaultNetworkPolicies(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"app": "test"}

	tests := []struct {
		name     string
		defaults *onboardingv1beta1.DefaultPoliciesSpec
		want     []string
	}{
		{
			name:     "minimal deny extras",
			defaults: minimalDefaultPolicies(),
			want:     []string{"allow-same-namespace"},
		},
		{
			name: "omitted deny-all flags stay off",
			defaults: &onboardingv1beta1.DefaultPoliciesSpec{
				AllowFromIngress:       boolPtr(false),
				AllowFromMonitoring:    boolPtr(false),
				AllowKubeAPIServer:     boolPtr(false),
				AllowToDNS:             boolPtr(false),
				AllowFromSameNamespace: boolPtr(true),
			},
			want: []string{"allow-same-namespace"},
		},
		{
			name: "ingress and deny all",
			defaults: &onboardingv1beta1.DefaultPoliciesSpec{
				AllowFromIngress:       boolPtr(true),
				AllowFromMonitoring:    boolPtr(false),
				AllowKubeAPIServer:     boolPtr(false),
				AllowToDNS:             boolPtr(false),
				AllowFromSameNamespace: boolPtr(true),
				DenyAllEgress:          boolPtr(true),
				DenyAllIngress:         boolPtr(true),
			},
			want: []string{
				"allow-from-openshift-ingress",
				"allow-same-namespace",
				"deny-all-egress",
				"deny-all-ingress",
			},
		},
		{
			name:     "nil defaults create no policies",
			defaults: nil,
			want:     []string{},
		},
		{
			name:     "platform baseline bundle",
			defaults: platformDefaultPolicies(),
			want: []string{
				"allow-from-kube-apiserver-operator",
				"allow-from-openshift-monitoring",
				"allow-same-namespace",
				"allow-to-openshift-dns",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			nsSpec := onboardingv1beta1.NamespaceSpec{
				Name:            "team-test",
				DefaultPolicies: tt.defaults,
			}
			got := defaultNetworkPolicies(nsSpec, "team-test", labels)
			gotNames := make([]string, 0, len(got))
			for _, p := range got {
				gotNames = append(gotNames, p.Name)
			}
			sort.Strings(gotNames)
			if len(gotNames) != len(tt.want) {
				t.Fatalf("policy count: got %d %v, want %d %v", len(gotNames), gotNames, len(tt.want), tt.want)
			}
			for i := range tt.want {
				if gotNames[i] != tt.want[i] {
					t.Fatalf("policy[%d]: got %q want %q (full got=%v)", i, gotNames[i], tt.want[i], gotNames)
				}
			}
		})
	}
}
