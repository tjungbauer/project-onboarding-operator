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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func projectOnboardingToTShirtSizeMapper() func(context.Context, client.Object) []reconcile.Request {
	return func(_ context.Context, obj client.Object) []reconcile.Request {
		po, ok := obj.(*onboardingv1beta1.ProjectOnboarding)
		if !ok {
			return nil
		}

		seen := map[string]struct{}{}
		requests := make([]reconcile.Request, 0)
		for _, ns := range po.Spec.Namespaces {
			if ns.ProjectSize == "" {
				continue
			}
			if _, ok := seen[ns.ProjectSize]; ok {
				continue
			}
			seen[ns.ProjectSize] = struct{}{}
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: ns.ProjectSize},
			})
		}
		return requests
	}
}
