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
	"os"
	"strings"
	"time"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"github.com/tjungbauer/project-onboarding-operator/internal/onboarding"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ProjectOnboardingReconciler reconciles a ProjectOnboarding object.
type ProjectOnboardingReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	OperatorNamespace string
	Recorder          record.EventRecorder
}

// +kubebuilder:rbac:groups=argoproj.io,resources=appprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=onboarding.stderr.at,resources=projectonboardings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=onboarding.stderr.at,resources=projectonboardings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=onboarding.stderr.at,resources=projectonboardings/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=resourcequotas;limitranges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,resourceNames=admin;edit;view,verbs=bind
// +kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.ovn.org,resources=egressips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=onboarding.stderr.at,resources=tshirtsizes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ProjectOnboardingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	po := &onboardingv1beta1.ProjectOnboarding{}
	if err := r.Get(ctx, req.NamespacedName, po); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !po.DeletionTimestamp.IsZero() {
		complete, pendingMessage, err := onboarding.FinalizeProjectOnboardingDeletion(ctx, r.Client, po)
		if err != nil {
			log.Error(err, "failed to finalize project onboarding deletion")
			return onboarding.ReconcileResultForError(err)
		}
		if !complete {
			status := po.Status.DeepCopy()
			onboarding.SetCondition(
				&status.Conditions,
				onboardingv1beta1.ConditionDeletionBlocked,
				metav1.ConditionTrue,
				"AwaitingOffboard",
				onboarding.TruncateStatusMessage(pendingMessage),
			)
			po.Status = *status
			if err := r.Status().Update(ctx, po); err != nil && !apierrors.IsConflict(err) {
				return onboarding.ReconcileResultForError(err)
			}
			recordWarning(r.Recorder, po, "DeletionBlocked", pendingMessage)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		if err := onboarding.RemoveFinalizer(ctx, r.Client, po); err != nil {
			return onboarding.ReconcileResultForError(err)
		}
		return ctrl.Result{}, nil
	}

	if err := onboarding.EnsureFinalizer(ctx, r.Client, po); err != nil {
		return onboarding.ReconcileResultForError(err)
	}

	if onboarding.IsReconciliationPaused(po) {
		log.Info("reconciliation paused", "annotation", onboarding.PauseReconciliationAnnotation)
		return ctrl.Result{}, nil
	}

	defaults, err := onboarding.LoadClusterDefaults(ctx, r.Client, r.operatorNamespace())
	if err != nil {
		log.Error(err, "failed to load cluster onboarding defaults")
		return onboarding.ReconcileResultForError(err)
	}
	effectivePO := onboarding.MergeClusterDefaults(defaults, po)

	if err := onboarding.PruneRemovedNamespaces(ctx, r.Client, po); err != nil {
		log.Error(err, "failed to prune removed namespaces")
		recordWarning(r.Recorder, po, "PruneFailed", err.Error())
		return onboarding.ReconcileResultForError(err)
	}

	status := po.Status.DeepCopy()
	status.Phase = onboardingv1beta1.PhasePending
	status.ObservedGeneration = po.Generation
	status.Namespaces = make([]onboardingv1beta1.NamespaceStatus, 0, len(po.Spec.Namespaces))

	allReady := true
	var firstError error

	for _, nsSpec := range po.Spec.Namespaces {
		nsStatus := onboardingv1beta1.NamespaceStatus{Name: nsSpec.Name}

		if onboarding.IsOffboard(nsSpec.Offboard) {
			if err := onboarding.CleanupNamespace(ctx, r.Client, po, nsSpec); err != nil {
				log.Error(err, "failed to offboard namespace", "namespace", nsSpec.Name)
				nsStatus.Ready = false
				nsStatus.Message = onboarding.TruncateStatusMessage(err.Error())
				allReady = false
				if firstError == nil {
					firstError = err
				}
			} else {
				nsStatus.Ready = true
				nsStatus.Message = "offboarded (resources removed)"
			}
			status.Namespaces = append(status.Namespaces, nsStatus)
			continue
		}

		if !onboarding.IsEnabled(nsSpec.Enabled) {
			nsStatus.Ready = true
			nsStatus.Message = "reconciliation disabled"
			status.Namespaces = append(status.Namespaces, nsStatus)
			continue
		}

		if err := onboarding.ReconcileNamespace(ctx, r.Client, r.Scheme, effectivePO, nsSpec); err != nil {
			log.Error(err, "failed to reconcile namespace", "namespace", nsSpec.Name)
			nsStatus.Ready = false
			nsStatus.Message = onboarding.TruncateStatusMessage(err.Error())
			allReady = false
			if firstError == nil {
				firstError = err
			}
		} else {
			nsStatus.Ready = true
			nsStatus.Message = "reconciled"
		}
		status.Namespaces = append(status.Namespaces, nsStatus)
	}

	if allReady {
		status.Phase = onboardingv1beta1.PhaseReady
		onboarding.SetCondition(&status.Conditions, onboardingv1beta1.ConditionReady, "True", "ReconcileSucceeded", "All namespaces reconciled")
		recordNormal(r.Recorder, po, "ReconcileSucceeded", "All namespaces reconciled")
	} else {
		status.Phase = onboardingv1beta1.PhaseFailed
		msg := onboarding.TruncateStatusMessage(fmt.Sprintf("one or more namespaces failed: %v", firstError))
		onboarding.SetCondition(&status.Conditions, onboardingv1beta1.ConditionReady, "False", "ReconcileFailed", msg)
		recordWarning(r.Recorder, po, "ReconcileFailed", msg)
	}

	po.Status = *status
	if err := r.Status().Update(ctx, po); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if firstError != nil {
		return onboarding.ReconcileResultForError(firstError)
	}

	return ctrl.Result{}, nil
}

func (r *ProjectOnboardingReconciler) operatorNamespace() string {
	if ns := strings.TrimSpace(r.OperatorNamespace); ns != "" {
		return ns
	}
	if ns := strings.TrimSpace(os.Getenv("POD_NAMESPACE")); ns != "" {
		return ns
	}
	return onboarding.DefaultOperatorNamespace
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectOnboardingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	managedPred := predicate.NewPredicateFuncs(onboarding.IsManagedResource)
	setupLog := logf.Log.WithName("projectonboarding-setup")

	bldr := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		For(&onboardingv1beta1.ProjectOnboarding{}).
		Watches(
			&onboardingv1beta1.TShirtSize{},
			handler.EnqueueRequestsFromMapFunc(tShirtSizeEnqueueMapper(r)),
		).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(managedResourceEnqueueMapper(r)),
			builder.WithPredicates(managedPred),
		).
		Watches(
			&corev1.ResourceQuota{},
			handler.EnqueueRequestsFromMapFunc(managedResourceEnqueueMapper(r)),
			builder.WithPredicates(managedPred),
		).
		Watches(
			&corev1.LimitRange{},
			handler.EnqueueRequestsFromMapFunc(managedResourceEnqueueMapper(r)),
			builder.WithPredicates(managedPred),
		).
		Watches(
			&networkingv1.NetworkPolicy{},
			handler.EnqueueRequestsFromMapFunc(managedResourceEnqueueMapper(r)),
			builder.WithPredicates(managedPred),
		).
		Watches(
			&rbacv1.RoleBinding{},
			handler.EnqueueRequestsFromMapFunc(managedResourceEnqueueMapper(r)),
			builder.WithPredicates(managedPred),
		)

	if isArgocdAppProjectAPIAvailable(mgr) {
		setupLog.Info("Argo CD AppProject API available; watching AppProject resources")
		bldr = bldr.Watches(
			appProjectWatchObject(),
			handler.EnqueueRequestsFromMapFunc(appProjectEnqueueMapper(r)),
			builder.WithPredicates(predicate.NewPredicateFuncs(isManagedAppProject)),
		)
	} else {
		setupLog.Info("Argo CD AppProject API not installed; skipping AppProject watch")
	}

	return bldr.Named("projectonboarding").Complete(r)
}
