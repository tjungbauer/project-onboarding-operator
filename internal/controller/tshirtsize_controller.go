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

package controller

import (
	"context"
	"fmt"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"github.com/tjungbauer/project-onboarding-operator/internal/onboarding"
	"github.com/tjungbauer/project-onboarding-operator/internal/validation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// TShirtSizeReconciler maintains catalogue status for cluster T-shirt size entries.
type TShirtSizeReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=onboarding.stderr.at,resources=tshirtsizes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=onboarding.stderr.at,resources=tshirtsizes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=onboarding.stderr.at,resources=projectonboardings,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *TShirtSizeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	size := &onboardingv1beta1.TShirtSize{}
	if err := r.Get(ctx, req.NamespacedName, size); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return onboarding.ReconcileResultForError(err)
	}

	refs, err := validation.CountProjectOnboardingReferences(ctx, r.Client, size.Name)
	if err != nil {
		log.Error(err, "failed to count ProjectOnboarding references")
		return onboarding.ReconcileResultForError(err)
	}

	status := size.Status.DeepCopy()
	status.ObservedGeneration = size.Generation
	status.ReferencedBy = int32(refs)

	if validation.TShirtSizeSpecHasSizing(size.Spec) {
		status.Phase = onboardingv1beta1.TShirtSizePhaseReady
		onboarding.SetCondition(&status.Conditions, onboardingv1beta1.TShirtSizeConditionReady, metav1.ConditionTrue, "CatalogueValid", "T-shirt size catalogue entry is valid")
		recordNormal(r.Recorder, size, "CatalogueValid", fmt.Sprintf("referenced by %d ProjectOnboarding resources", refs))
	} else {
		status.Phase = onboardingv1beta1.TShirtSizePhaseInvalid
		msg := "spec must define resourceQuotas and/or limitRanges with at least one limit value"
		onboarding.SetCondition(&status.Conditions, onboardingv1beta1.TShirtSizeConditionReady, metav1.ConditionFalse, "CatalogueInvalid", msg)
		recordWarning(r.Recorder, size, "CatalogueInvalid", msg)
	}

	size.Status = *status
	if err := r.Status().Update(ctx, size); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return onboarding.ReconcileResultForError(fmt.Errorf("update TShirtSize status: %w", err))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TShirtSizeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		For(&onboardingv1beta1.TShirtSize{}).
		Watches(
			&onboardingv1beta1.ProjectOnboarding{},
			handler.EnqueueRequestsFromMapFunc(projectOnboardingToTShirtSizeMapper()),
		).
		Named("tshirtsize").
		Complete(r)
}
