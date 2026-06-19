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
	"context"

	onboardingv1alpha1 "github.com/tjungbauer/project-onboarding-operator/api/v1alpha1"
	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"github.com/tjungbauer/project-onboarding-operator/internal/validation"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-onboarding-stderr-at-v1alpha1-projectonboarding,mutating=false,failurePolicy=fail,sideEffects=None,groups=onboarding.stderr.at,resources=projectonboardings,verbs=create;update,versions=v1alpha1,name=vprojectonboarding.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-onboarding-stderr-at-v1beta1-projectonboarding,mutating=false,failurePolicy=fail,sideEffects=None,groups=onboarding.stderr.at,resources=projectonboardings,verbs=create;update,versions=v1beta1,name=vprojectonboardingv1beta1.kb.io,admissionReviewVersions=v1

type ProjectOnboardingCustomValidator struct {
	Validator *validation.Validator
}

var _ webhook.CustomValidator = &ProjectOnboardingCustomValidator{}

var projectOnboardingValidatorLog = ctrl.Log.WithName("projectonboarding-validator")

func (v *ProjectOnboardingCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	po, err := asProjectOnboardingHub(obj)
	if err != nil {
		return nil, err
	}
	projectOnboardingValidatorLog.V(1).Info("validate create", "name", po.Name)
	return v.Validator.ValidateProjectOnboarding(ctx, po)
}

func (v *ProjectOnboardingCustomValidator) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	po, err := asProjectOnboardingHub(newObj)
	if err != nil {
		return nil, err
	}
	projectOnboardingValidatorLog.V(1).Info("validate update", "name", po.Name)
	return v.Validator.ValidateProjectOnboarding(ctx, po)
}

func (v *ProjectOnboardingCustomValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// SetupProjectOnboardingWebhookWithManager registers the validating webhook.
func SetupProjectOnboardingWebhookWithManager(mgr ctrl.Manager, validator *validation.Validator) error {
	v := &ProjectOnboardingCustomValidator{Validator: validator}
	if err := ctrl.NewWebhookManagedBy(mgr).
		For(&onboardingv1alpha1.ProjectOnboarding{}).
		WithValidator(v).
		Complete(); err != nil {
		return err
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&onboardingv1beta1.ProjectOnboarding{}).
		WithValidator(v).
		Complete()
}
