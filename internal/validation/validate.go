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

package validation

import (
	"context"
	"fmt"
	"strings"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"github.com/tjungbauer/project-onboarding-operator/internal/onboarding"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Validator performs cross-field and cross-resource admission checks.
type Validator struct {
	Client client.Client
}

func NewValidator(c client.Client) *Validator {
	return &Validator{Client: c}
}

func (v *Validator) ValidateProjectOnboarding(ctx context.Context, po *onboardingv1beta1.ProjectOnboarding) (admission.Warnings, error) {
	if err := validateUniqueNamespaceNames(po.Spec.Namespaces); err != nil {
		return nil, err
	}
	if err := validateGitOpsRequirements(po); err != nil {
		return nil, err
	}

	for _, ns := range po.Spec.Namespaces {
		if err := validateNamespaceSpec(ctx, v.Client, ns); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func validateGitOpsRequirements(po *onboardingv1beta1.ProjectOnboarding) error {
	for _, ns := range po.Spec.Namespaces {
		if !isEnabledPtr(ns.Enabled) {
			continue
		}
		hasProjects := false
		for _, project := range ns.ArgoCDProjects {
			if isOptInEnabledPtr(project.Enabled) {
				hasProjects = true
				break
			}
		}
		if !hasProjects {
			continue
		}
		if onboarding.ResolveApplicationGitOpsNamespace(po, ns) == "" {
			return fmt.Errorf("spec.namespaces[%q].applicationGitOpsNamespace is required when argoCDProjects are enabled (legacy fallback: spec.gitOps.applicationNamespace or onboarding-defaults ConfigMap)", ns.Name)
		}
	}
	return nil
}

func isOptInEnabledPtr(enabled *bool) bool {
	return enabled != nil && *enabled
}

func isEnabledPtr(enabled *bool) bool {
	if enabled == nil {
		return true
	}
	return *enabled
}

func (v *Validator) ValidateTShirtSize(ctx context.Context, size *onboardingv1beta1.TShirtSize) (admission.Warnings, error) {
	if errs := validation.IsDNS1123Subdomain(size.Name); len(errs) > 0 {
		return nil, fmt.Errorf("metadata.name %q is not a valid DNS-1123 subdomain: %s", size.Name, strings.Join(errs, ", "))
	}
	if !TShirtSizeSpecHasSizing(size.Spec) {
		return nil, fmt.Errorf("spec must define resourceQuotas and/or limitRanges with at least one limit value")
	}
	if err := validateResourceQuotaSpec("spec.resourceQuotas", size.Spec.ResourceQuotas); err != nil {
		return nil, err
	}
	if err := validateLimitRangeSpec("spec.limitRanges", size.Spec.LimitRanges); err != nil {
		return nil, err
	}
	return nil, nil
}

func (v *Validator) ValidateTShirtSizeDelete(ctx context.Context, size *onboardingv1beta1.TShirtSize) (admission.Warnings, error) {
	refs, err := v.countProjectOnboardingReferences(ctx, size.Name)
	if err != nil {
		return nil, err
	}
	if refs > 0 {
		return nil, fmt.Errorf("TShirtSize %q is referenced by %d ProjectOnboarding namespace entries; remove references before delete", size.Name, refs)
	}
	return nil, nil
}

func (v *Validator) countProjectOnboardingReferences(ctx context.Context, sizeName string) (int, error) {
	return CountProjectOnboardingReferences(ctx, v.Client, sizeName)
}

// CountProjectOnboardingReferences returns how many namespace entries reference a T-shirt size.
func CountProjectOnboardingReferences(ctx context.Context, c client.Client, sizeName string) (int, error) {
	list := &onboardingv1beta1.ProjectOnboardingList{}
	if err := c.List(ctx, list); err != nil {
		return 0, err
	}
	count := 0
	for _, po := range list.Items {
		for _, ns := range po.Spec.Namespaces {
			if ns.ProjectSize == sizeName {
				count++
			}
		}
	}
	return count, nil
}

func validateUniqueNamespaceNames(namespaces []onboardingv1beta1.NamespaceSpec) error {
	seen := make(map[string]struct{}, len(namespaces))
	for _, ns := range namespaces {
		name := strings.TrimSpace(ns.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate tenant namespace name %q in spec.namespaces", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func validateNamespaceSpec(ctx context.Context, c client.Client, ns onboardingv1beta1.NamespaceSpec) error {
	prefix := fmt.Sprintf("spec.namespaces[%q]", ns.Name)
	if err := validateResourceQuotaSpec(prefix+".resourceQuotas", ns.ResourceQuotas); err != nil {
		return err
	}
	if err := validateLimitRangeSpec(prefix+".limitRanges", ns.LimitRanges); err != nil {
		return err
	}

	if ns.ProjectSize == "" {
		return nil
	}

	tshirt := &onboardingv1beta1.TShirtSize{}
	if err := c.Get(ctx, client.ObjectKey{Name: ns.ProjectSize}, tshirt); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("TShirtSize %q not found (referenced by namespace entry %q)", ns.ProjectSize, ns.Name)
		}
		return fmt.Errorf("get TShirtSize %q: %w", ns.ProjectSize, err)
	}
	if !TShirtSizeSpecHasSizing(tshirt.Spec) {
		return fmt.Errorf("TShirtSize %q has no quota or limit values", ns.ProjectSize)
	}
	return nil
}

// TShirtSizeSpecHasSizing reports whether a catalogue entry defines usable quota or limit data.
func TShirtSizeSpecHasSizing(spec onboardingv1beta1.TShirtSizeSpec) bool {
	return resourceQuotaSpecHasValues(spec.ResourceQuotas) || limitRangeSpecHasValues(spec.LimitRanges)
}

func resourceQuotaSpecHasValues(spec *onboardingv1beta1.ResourceQuotaSpec) bool {
	if spec == nil {
		return false
	}
	if spec.Enabled != nil && !*spec.Enabled {
		return false
	}
	return spec.Pods != nil || spec.CPU != nil || spec.Memory != nil || spec.EphemeralStorage != nil ||
		spec.ReplicationControllers != nil || spec.ResourceQuotas != nil || spec.Services != nil ||
		spec.Secrets != nil || spec.ConfigMaps != nil || spec.PersistentVolumeClaims != nil ||
		spec.Limits != nil || spec.Requests != nil || len(spec.StorageClasses) > 0
}

func limitRangeSpecHasValues(spec *onboardingv1beta1.LimitRangeSpec) bool {
	if spec == nil {
		return false
	}
	if spec.Enabled != nil && !*spec.Enabled {
		return false
	}
	return spec.Pod != nil || spec.Container != nil || spec.PVC != nil
}
