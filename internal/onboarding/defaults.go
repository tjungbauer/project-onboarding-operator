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
	"fmt"
	"strings"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	// DefaultsConfigMapName is the cluster-wide GitOps defaults ConfigMap in the operator namespace.
	DefaultsConfigMapName = "onboarding-defaults"
	// DefaultsConfigMapKey holds YAML with a partial ProjectOnboarding spec (typically spec.gitOps).
	DefaultsConfigMapKey = "defaults.yaml"
	// DefaultOperatorNamespace is used when POD_NAMESPACE is unset.
	DefaultOperatorNamespace = "project-onboarding-operator"
)

// LoadClusterDefaults reads optional GitOps defaults from a ConfigMap in the operator namespace.
func LoadClusterDefaults(ctx context.Context, c client.Client, operatorNamespace string) (*onboardingv1beta1.GitOpsSpec, error) {
	ns := strings.TrimSpace(operatorNamespace)
	if ns == "" {
		ns = DefaultOperatorNamespace
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: DefaultsConfigMapName}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	raw, ok := cm.Data[DefaultsConfigMapKey]
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var partial struct {
		GitOps *onboardingv1beta1.GitOpsSpec `json:"gitOps"`
	}
	if err := yaml.Unmarshal([]byte(raw), &partial); err != nil {
		return nil, fmt.Errorf("parse %s in ConfigMap %q: %w", DefaultsConfigMapKey, DefaultsConfigMapName, err)
	}
	if err := validateGitOpsDefaults(partial.GitOps); err != nil {
		return nil, fmt.Errorf("invalid %s in ConfigMap %q: %w", DefaultsConfigMapKey, DefaultsConfigMapName, err)
	}
	return partial.GitOps, nil
}

func validateGitOpsDefaults(spec *onboardingv1beta1.GitOpsSpec) error {
	if spec == nil {
		return nil
	}
	for i, dest := range spec.Destinations {
		if strings.TrimSpace(dest.Name) == "" && strings.TrimSpace(dest.Server) == "" {
			return fmt.Errorf("gitOps.destinations[%d] requires name or server", i)
		}
	}
	return nil
}

// MergeClusterDefaults returns a copy of po with cluster GitOps defaults applied (CR spec wins on conflict).
func MergeClusterDefaults(defaults *onboardingv1beta1.GitOpsSpec, po *onboardingv1beta1.ProjectOnboarding) *onboardingv1beta1.ProjectOnboarding {
	if defaults == nil {
		return po
	}
	out := po.DeepCopy()
	if out.Spec.GitOps == nil {
		out.Spec.GitOps = defaults.DeepCopy()
		return out
	}
	mergeGitOpsDefaults(out.Spec.GitOps, defaults)
	return out
}

func mergeGitOpsDefaults(target, defaults *onboardingv1beta1.GitOpsSpec) {
	if strings.TrimSpace(target.ApplicationNamespace) == "" {
		target.ApplicationNamespace = defaults.ApplicationNamespace
	}
	if len(target.Destinations) == 0 {
		target.Destinations = append([]onboardingv1beta1.GitOpsDestinationSpec(nil), defaults.Destinations...)
	}
	if len(target.AllowedSourceRepos) == 0 {
		target.AllowedSourceRepos = append([]string(nil), defaults.AllowedSourceRepos...)
	}
	if len(target.AllowedOIDCGroups) == 0 {
		target.AllowedOIDCGroups = append([]string(nil), defaults.AllowedOIDCGroups...)
	}
	if len(target.AllowedSourceNamespaces) == 0 {
		target.AllowedSourceNamespaces = append([]string(nil), defaults.AllowedSourceNamespaces...)
	}
}
