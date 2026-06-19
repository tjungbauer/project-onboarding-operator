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
	"fmt"
	"regexp"
	"strings"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var bareNumberQuantity = regexp.MustCompile(`^[\d.]+$`)

func normalizeQuantity(value string) string {
	return strings.NewReplacer("gi", "Gi", "mi", "Mi", "GI", "Gi", "MI", "Mi").Replace(value)
}

func validateCPUQuantity(fieldPath, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if _, err := resource.ParseQuantity(trimmed); err != nil {
		return fmt.Errorf("%s: invalid CPU quantity %q: %v", fieldPath, value, err)
	}
	return nil
}

func validateByteQuantity(fieldPath, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if bareNumberQuantity.MatchString(trimmed) {
		return fmt.Errorf("%s: quantity %q must include a unit suffix (e.g. 4Gi, 500Mi); plain numbers are interpreted as bytes by Kubernetes", fieldPath, value)
	}
	normalized := normalizeQuantity(trimmed)
	if _, err := resource.ParseQuantity(normalized); err != nil {
		return fmt.Errorf("%s: invalid quantity %q: %v", fieldPath, value, err)
	}
	return nil
}

func validateStorageClassQuotaValue(fieldPath, key, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	keyLower := strings.ToLower(key)
	if strings.Contains(keyLower, "persistentvolumeclaims") {
		if bareNumberQuantity.MatchString(trimmed) {
			if _, err := resource.ParseQuantity(trimmed); err != nil {
				return fmt.Errorf("%s: invalid count %q: %v", fieldPath, value, err)
			}
			return nil
		}
	}
	if strings.Contains(keyLower, "storage") {
		return validateByteQuantity(fieldPath, trimmed)
	}
	if bareNumberQuantity.MatchString(trimmed) {
		if _, err := resource.ParseQuantity(trimmed); err != nil {
			return fmt.Errorf("%s: invalid count %q: %v", fieldPath, value, err)
		}
		return nil
	}
	return validateByteQuantity(fieldPath, trimmed)
}

func validateResourceAmountSpec(prefix string, spec *onboardingv1beta1.ResourceAmountSpec) error {
	if spec == nil {
		return nil
	}
	if spec.CPU != nil {
		if err := validateCPUQuantity(prefix+".cpu", *spec.CPU); err != nil {
			return err
		}
	}
	if spec.Memory != nil {
		if err := validateByteQuantity(prefix+".memory", *spec.Memory); err != nil {
			return err
		}
	}
	return nil
}

func validateStorageAmountSpec(prefix string, spec *onboardingv1beta1.StorageAmountSpec) error {
	if spec == nil || spec.Storage == nil {
		return nil
	}
	return validateByteQuantity(prefix+".storage", *spec.Storage)
}

func validateResourceQuotaLimitSpec(prefix string, spec *onboardingv1beta1.ResourceQuotaLimitSpec) error {
	if spec == nil {
		return nil
	}
	if spec.CPU != nil {
		if err := validateCPUQuantity(prefix+".cpu", *spec.CPU); err != nil {
			return err
		}
	}
	if spec.Memory != nil {
		if err := validateByteQuantity(prefix+".memory", *spec.Memory); err != nil {
			return err
		}
	}
	if spec.EphemeralStorage != nil {
		if err := validateByteQuantity(prefix+".ephemeralStorage", *spec.EphemeralStorage); err != nil {
			return err
		}
	}
	return nil
}

func validateResourceQuotaRequestSpec(prefix string, spec *onboardingv1beta1.ResourceQuotaRequestSpec) error {
	if spec == nil {
		return nil
	}
	if spec.CPU != nil {
		if err := validateCPUQuantity(prefix+".cpu", *spec.CPU); err != nil {
			return err
		}
	}
	if spec.Memory != nil {
		if err := validateByteQuantity(prefix+".memory", *spec.Memory); err != nil {
			return err
		}
	}
	if spec.EphemeralStorage != nil {
		if err := validateByteQuantity(prefix+".ephemeralStorage", *spec.EphemeralStorage); err != nil {
			return err
		}
	}
	if spec.Storage != nil {
		if err := validateByteQuantity(prefix+".storage", *spec.Storage); err != nil {
			return err
		}
	}
	return nil
}

func validateResourceQuotaSpec(prefix string, spec *onboardingv1beta1.ResourceQuotaSpec) error {
	if spec == nil {
		return nil
	}
	if spec.CPU != nil {
		if err := validateCPUQuantity(prefix+".cpu", *spec.CPU); err != nil {
			return err
		}
	}
	if spec.Memory != nil {
		if err := validateByteQuantity(prefix+".memory", *spec.Memory); err != nil {
			return err
		}
	}
	if spec.EphemeralStorage != nil {
		if err := validateByteQuantity(prefix+".ephemeralStorage", *spec.EphemeralStorage); err != nil {
			return err
		}
	}
	if err := validateResourceQuotaLimitSpec(prefix+".limits", spec.Limits); err != nil {
		return err
	}
	if err := validateResourceQuotaRequestSpec(prefix+".requests", spec.Requests); err != nil {
		return err
	}
	for i, entry := range spec.StorageClasses {
		fieldPath := fmt.Sprintf("%s.storageClasses[%d].value", prefix, i)
		if err := validateStorageClassQuotaValue(fieldPath, entry.Key, entry.Value); err != nil {
			return err
		}
	}
	return nil
}

func validateLimitRangeSpec(prefix string, spec *onboardingv1beta1.LimitRangeSpec) error {
	if spec == nil {
		return nil
	}
	if spec.Pod != nil {
		if err := validateResourceAmountSpec(prefix+".pod.max", spec.Pod.Max); err != nil {
			return err
		}
		if err := validateResourceAmountSpec(prefix+".pod.min", spec.Pod.Min); err != nil {
			return err
		}
	}
	if spec.Container != nil {
		if err := validateResourceAmountSpec(prefix+".container.max", spec.Container.Max); err != nil {
			return err
		}
		if err := validateResourceAmountSpec(prefix+".container.min", spec.Container.Min); err != nil {
			return err
		}
		if err := validateResourceAmountSpec(prefix+".container.default", spec.Container.Default); err != nil {
			return err
		}
		if err := validateResourceAmountSpec(prefix+".container.defaultRequest", spec.Container.DefaultRequest); err != nil {
			return err
		}
	}
	if spec.PVC != nil {
		if err := validateStorageAmountSpec(prefix+".pvc.min", spec.PVC.Min); err != nil {
			return err
		}
		if err := validateStorageAmountSpec(prefix+".pvc.max", spec.PVC.Max); err != nil {
			return err
		}
	}
	return nil
}
