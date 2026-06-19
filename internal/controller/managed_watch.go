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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func managedResourceEnqueueMapper(r *ProjectOnboardingReconciler) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		if !onboarding.IsManagedResource(obj) {
			return nil
		}

		labels := obj.GetLabels()
		poName := labels[onboardingv1beta1.ProjectOnboardingLabelKey]
		if poName == "" {
			return nil
		}
		return []reconcile.Request{projectOnboardingRequest(poName)}
	}
}
