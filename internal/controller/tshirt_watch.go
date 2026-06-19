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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ProjectOnboardingReconciler) findProjectOnboardingsForTShirtSize(ctx context.Context, tshirt client.Object) []reconcile.Request {
	size, ok := tshirt.(*onboardingv1beta1.TShirtSize)
	if !ok {
		return nil
	}

	list := &onboardingv1beta1.ProjectOnboardingList{}
	if err := r.List(ctx, list); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0)
	for _, po := range list.Items {
		for _, ns := range po.Spec.Namespaces {
			if ns.ProjectSize == size.Name {
				requests = append(requests, projectOnboardingRequest(po.Name))
				break
			}
		}
	}
	return requests
}

func tShirtSizeEnqueueMapper(r *ProjectOnboardingReconciler) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		return r.findProjectOnboardingsForTShirtSize(ctx, obj)
	}
}
