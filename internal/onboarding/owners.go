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

package onboarding

import (
	"context"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// IsManagedResource reports whether obj was created by this operator and can be mapped back to a ProjectOnboarding.
func IsManagedResource(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	return labels[onboardingv1beta1.ProjectOnboardingManagedByKey] == onboardingv1beta1.ProjectOnboardingManagedByVal &&
		labels[onboardingv1beta1.ProjectOnboardingLabelKey] != ""
}

func getTenantNamespace(ctx context.Context, c client.Client, nsName string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{}
	if err := c.Get(ctx, client.ObjectKey{Name: nsName}, ns); err != nil {
		return nil, err
	}
	return ns, nil
}

func ensureProjectOnboardingControllerRef(scheme *runtime.Scheme, po *onboardingv1beta1.ProjectOnboarding, obj metav1.Object) error {
	if scheme == nil || po == nil {
		return nil
	}
	if metav1.IsControlledBy(obj, po) {
		return nil
	}
	return controllerutil.SetControllerReference(po, obj, scheme)
}

// ensureTenantNamespaceOwnerRef sets the tenant Namespace as owner of namespaced
// resources inside that namespace (quotas, network policies, role bindings).
func ensureTenantNamespaceOwnerRef(scheme *runtime.Scheme, tenantNS *corev1.Namespace, obj metav1.Object) error {
	if scheme == nil || tenantNS == nil {
		return nil
	}
	if metav1.IsControlledBy(obj, tenantNS) {
		return nil
	}
	return controllerutil.SetOwnerReference(tenantNS, obj, scheme)
}
