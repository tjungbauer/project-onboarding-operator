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

package webhook

import (
	"fmt"

	apiconversion "github.com/tjungbauer/project-onboarding-operator/api/conversion"
	onboardingv1alpha1 "github.com/tjungbauer/project-onboarding-operator/api/v1alpha1"
	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func asProjectOnboardingHub(obj runtime.Object) (*onboardingv1beta1.ProjectOnboarding, error) {
	switch t := obj.(type) {
	case *onboardingv1beta1.ProjectOnboarding:
		return t, nil
	case *onboardingv1alpha1.ProjectOnboarding:
		out := &onboardingv1beta1.ProjectOnboarding{}
		if err := apiconversion.ViaJSON(t, out); err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected ProjectOnboarding, got %T", obj)
	}
}

func asTShirtSizeHub(obj runtime.Object) (*onboardingv1beta1.TShirtSize, error) {
	switch t := obj.(type) {
	case *onboardingv1beta1.TShirtSize:
		return t, nil
	case *onboardingv1alpha1.TShirtSize:
		out := &onboardingv1beta1.TShirtSize{}
		if err := apiconversion.ViaJSON(t, out); err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected TShirtSize, got %T", obj)
	}
}
