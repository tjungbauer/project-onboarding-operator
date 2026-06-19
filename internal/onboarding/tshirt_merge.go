/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
you may obtain a copy of the License at

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

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveNamespaceSpec applies T-shirt sizing: loads TShirtSize when projectSize is set
// and merges namespace overrides only when overwriteTshirt is true.
func ResolveNamespaceSpec(ctx context.Context, c client.Client, nsSpec onboardingv1beta1.NamespaceSpec) (onboardingv1beta1.NamespaceSpec, error) {
	if nsSpec.ProjectSize == "" {
		return nsSpec, nil
	}

	tshirt := &onboardingv1beta1.TShirtSize{}
	if err := c.Get(ctx, client.ObjectKey{Name: nsSpec.ProjectSize}, tshirt); err != nil {
		if apierrors.IsNotFound(err) {
			return nsSpec, fmt.Errorf("TShirtSize %q not found", nsSpec.ProjectSize)
		}
		return nsSpec, fmt.Errorf("get TShirtSize %q: %w", nsSpec.ProjectSize, err)
	}

	resolved := nsSpec

	applyTshirtQuota := shouldApplyTshirtResourceQuotas(nsSpec)
	applyTshirtLimits := shouldApplyTshirtLimitRanges(nsSpec)

	if shouldMergeTshirtOverrides(nsSpec) {
		if applyTshirtQuota {
			resolved.ResourceQuotas = mergeResourceQuotaSpec(tshirt.Spec.ResourceQuotas, nsSpec.ResourceQuotas)
		} else {
			resolved.ResourceQuotas = nil
		}

		if applyTshirtLimits {
			resolved.LimitRanges = mergeLimitRangeSpec(tshirt.Spec.LimitRanges, nsSpec.LimitRanges)
		} else {
			resolved.LimitRanges = nil
		}
	} else {
		if applyTshirtQuota {
			resolved.ResourceQuotas = ensureQuotaEnabled(tshirt.Spec.ResourceQuotas)
		} else {
			resolved.ResourceQuotas = nil
		}
		if applyTshirtLimits {
			resolved.LimitRanges = ensureLimitRangeEnabled(tshirt.Spec.LimitRanges)
		} else {
			resolved.LimitRanges = nil
		}
	}
	return resolved, nil
}

// shouldApplyTshirtResourceQuotas mirrors helper-proj-onboarding tshirt-sizes/resourcequota.yaml:
// projectSize set, resourceQuotas block present, and resourceQuotas.enabled explicitly true.
func shouldApplyTshirtResourceQuotas(nsSpec onboardingv1beta1.NamespaceSpec) bool {
	return nsSpec.ProjectSize != "" &&
		nsSpec.ResourceQuotas != nil &&
		isExplicitlyEnabled(nsSpec.ResourceQuotas.Enabled)
}

// shouldApplyTshirtLimitRanges applies catalogue limit ranges when projectSize is set
// and limitRanges.enabled is explicitly true on the namespace entry.
func shouldApplyTshirtLimitRanges(nsSpec onboardingv1beta1.NamespaceSpec) bool {
	return nsSpec.ProjectSize != "" &&
		nsSpec.LimitRanges != nil &&
		isExplicitlyEnabled(nsSpec.LimitRanges.Enabled)
}

func isExplicitlyEnabled(enabled *bool) bool {
	return enabled != nil && *enabled
}

// shouldMergeTshirtOverrides reports whether namespace resourceQuotas/limitRanges
// merge onto the referenced TShirtSize (field-level). Requires overwriteTshirt: true.
func shouldMergeTshirtOverrides(nsSpec onboardingv1beta1.NamespaceSpec) bool {
	return IsOverwriteTshirt(nsSpec.OverwriteTshirt)
}

func IsOverwriteTshirt(overwrite *bool) bool {
	if overwrite == nil {
		return false
	}
	return *overwrite
}

func ensureQuotaEnabled(q *onboardingv1beta1.ResourceQuotaSpec) *onboardingv1beta1.ResourceQuotaSpec {
	if q == nil {
		return nil
	}
	out := q.DeepCopy()
	if out.Enabled == nil {
		out.Enabled = boolPtr(true)
	}
	return out
}

func ensureLimitRangeEnabled(lr *onboardingv1beta1.LimitRangeSpec) *onboardingv1beta1.LimitRangeSpec {
	if lr == nil {
		return nil
	}
	out := lr.DeepCopy()
	if out.Enabled == nil {
		out.Enabled = boolPtr(true)
	}
	return out
}

func mergeResourceQuotaSpec(base, override *onboardingv1beta1.ResourceQuotaSpec) *onboardingv1beta1.ResourceQuotaSpec {
	if base == nil && override == nil {
		return nil
	}

	var out onboardingv1beta1.ResourceQuotaSpec
	if base != nil {
		out = *base.DeepCopy()
	}
	if override == nil {
		return ensureQuotaEnabled(&out)
	}

	if override.Enabled != nil {
		out.Enabled = override.Enabled
	}
	if override.Pods != nil {
		out.Pods = override.Pods
	}
	if override.CPU != nil {
		out.CPU = override.CPU
	}
	if override.Memory != nil {
		out.Memory = override.Memory
	}
	if override.EphemeralStorage != nil {
		out.EphemeralStorage = override.EphemeralStorage
	}
	if override.ReplicationControllers != nil {
		out.ReplicationControllers = override.ReplicationControllers
	}
	if override.ResourceQuotas != nil {
		out.ResourceQuotas = override.ResourceQuotas
	}
	if override.Services != nil {
		out.Services = override.Services
	}
	if override.Secrets != nil {
		out.Secrets = override.Secrets
	}
	if override.ConfigMaps != nil {
		out.ConfigMaps = override.ConfigMaps
	}
	if override.PersistentVolumeClaims != nil {
		out.PersistentVolumeClaims = override.PersistentVolumeClaims
	}
	out.Limits = mergeResourceQuotaLimits(out.Limits, override.Limits)
	out.Requests = mergeResourceQuotaRequests(out.Requests, override.Requests)
	if len(override.StorageClasses) > 0 {
		outMap := storageClassQuotasToMap(out.StorageClasses)
		for _, entry := range override.StorageClasses {
			if entry.Key == "" {
				continue
			}
			outMap[entry.Key] = entry.Value
		}
		out.StorageClasses = storageClassQuotasFromMap(outMap)
	}

	return ensureQuotaEnabled(&out)
}

func mergeResourceQuotaLimits(base, override *onboardingv1beta1.ResourceQuotaLimitSpec) *onboardingv1beta1.ResourceQuotaLimitSpec {
	if base == nil && override == nil {
		return nil
	}
	var out onboardingv1beta1.ResourceQuotaLimitSpec
	if base != nil {
		out = *base.DeepCopy()
	}
	if override == nil {
		if base == nil {
			return nil
		}
		return &out
	}
	if override.CPU != nil {
		out.CPU = override.CPU
	}
	if override.Memory != nil {
		out.Memory = override.Memory
	}
	if override.EphemeralStorage != nil {
		out.EphemeralStorage = override.EphemeralStorage
	}
	return &out
}

func mergeResourceQuotaRequests(base, override *onboardingv1beta1.ResourceQuotaRequestSpec) *onboardingv1beta1.ResourceQuotaRequestSpec {
	if base == nil && override == nil {
		return nil
	}
	var out onboardingv1beta1.ResourceQuotaRequestSpec
	if base != nil {
		out = *base.DeepCopy()
	}
	if override == nil {
		if base == nil {
			return nil
		}
		return &out
	}
	if override.CPU != nil {
		out.CPU = override.CPU
	}
	if override.Memory != nil {
		out.Memory = override.Memory
	}
	if override.Storage != nil {
		out.Storage = override.Storage
	}
	if override.EphemeralStorage != nil {
		out.EphemeralStorage = override.EphemeralStorage
	}
	return &out
}

func mergeLimitRangeSpec(base, override *onboardingv1beta1.LimitRangeSpec) *onboardingv1beta1.LimitRangeSpec {
	if base == nil && override == nil {
		return nil
	}

	var out onboardingv1beta1.LimitRangeSpec
	if base != nil {
		out = *base.DeepCopy()
	}
	if override == nil {
		return ensureLimitRangeEnabled(&out)
	}

	if override.Enabled != nil {
		out.Enabled = override.Enabled
	}
	out.Pod = mergeLimitRangePod(out.Pod, override.Pod)
	out.Container = mergeLimitRangeContainer(out.Container, override.Container)
	out.PVC = mergeLimitRangePVC(out.PVC, override.PVC)

	return ensureLimitRangeEnabled(&out)
}

func mergeLimitRangePod(base, override *onboardingv1beta1.LimitRangePodSpec) *onboardingv1beta1.LimitRangePodSpec {
	if base == nil && override == nil {
		return nil
	}
	var out onboardingv1beta1.LimitRangePodSpec
	if base != nil {
		out = *base.DeepCopy()
	}
	if override == nil {
		if base == nil {
			return nil
		}
		return &out
	}
	out.Max = mergeResourceAmount(out.Max, override.Max)
	out.Min = mergeResourceAmount(out.Min, override.Min)
	return &out
}

func mergeLimitRangeContainer(base, override *onboardingv1beta1.LimitRangeContainerSpec) *onboardingv1beta1.LimitRangeContainerSpec {
	if base == nil && override == nil {
		return nil
	}
	var out onboardingv1beta1.LimitRangeContainerSpec
	if base != nil {
		out = *base.DeepCopy()
	}
	if override == nil {
		if base == nil {
			return nil
		}
		return &out
	}
	out.Max = mergeResourceAmount(out.Max, override.Max)
	out.Min = mergeResourceAmount(out.Min, override.Min)
	out.Default = mergeResourceAmount(out.Default, override.Default)
	out.DefaultRequest = mergeResourceAmount(out.DefaultRequest, override.DefaultRequest)
	return &out
}

func mergeLimitRangePVC(base, override *onboardingv1beta1.LimitRangePVCSpec) *onboardingv1beta1.LimitRangePVCSpec {
	if base == nil && override == nil {
		return nil
	}
	var out onboardingv1beta1.LimitRangePVCSpec
	if base != nil {
		out = *base.DeepCopy()
	}
	if override == nil {
		if base == nil {
			return nil
		}
		return &out
	}
	if override.Min != nil {
		if out.Min == nil {
			out.Min = &onboardingv1beta1.StorageAmountSpec{}
		}
		if override.Min.Storage != nil {
			out.Min.Storage = override.Min.Storage
		}
	}
	if override.Max != nil {
		if out.Max == nil {
			out.Max = &onboardingv1beta1.StorageAmountSpec{}
		}
		if override.Max.Storage != nil {
			out.Max.Storage = override.Max.Storage
		}
	}
	return &out
}

func mergeResourceAmount(base, override *onboardingv1beta1.ResourceAmountSpec) *onboardingv1beta1.ResourceAmountSpec {
	if base == nil && override == nil {
		return nil
	}
	var out onboardingv1beta1.ResourceAmountSpec
	if base != nil {
		out = *base.DeepCopy()
	}
	if override == nil {
		if base == nil {
			return nil
		}
		return &out
	}
	if override.CPU != nil {
		out.CPU = override.CPU
	}
	if override.Memory != nil {
		out.Memory = override.Memory
	}
	return &out
}
