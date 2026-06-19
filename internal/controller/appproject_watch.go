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

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"github.com/tjungbauer/project-onboarding-operator/internal/onboarding"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var appProjectGVK = schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "AppProject"}

// isArgocdAppProjectAPIAvailable reports whether the Argo CD AppProject API is registered
// on the cluster (appprojects.argoproj.io CRD). When false, the controller must not watch
// AppProject resources or controller-runtime will block startup on Kind and other clusters
// without GitOps. Reconcile still creates AppProjects when spec.gitOps is configured.
func isArgocdAppProjectAPIAvailable(mgr manager.Manager) bool {
	_, err := mgr.GetRESTMapper().RESTMapping(appProjectGVK.GroupKind(), appProjectGVK.Version)
	return err == nil
}

func appProjectWatchObject() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(appProjectGVK)
	return obj
}

func isManagedAppProject(obj client.Object) bool {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok || u.GroupVersionKind() != appProjectGVK {
		return false
	}
	return onboarding.IsManagedResource(obj)
}

func appProjectEnqueueMapper(r *ProjectOnboardingReconciler) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		if !isManagedAppProject(obj) {
			return nil
		}
		poName := obj.GetLabels()[onboardingv1beta1.ProjectOnboardingLabelKey]
		if poName == "" {
			return nil
		}
		return []reconcile.Request{projectOnboardingRequest(poName)}
	}
}
