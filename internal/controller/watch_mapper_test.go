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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	"github.com/tjungbauer/project-onboarding-operator/internal/onboarding"
)

var _ = Describe("Watch mappers", func() {
	It("enqueues ProjectOnboarding when a referenced TShirtSize changes", func() {
		reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		size := &onboardingv1beta1.TShirtSize{
			ObjectMeta: metav1.ObjectMeta{Name: "mapper-size"},
		}
		reqs := reconciler.findProjectOnboardingsForTShirtSize(context.Background(), size)
		Expect(reqs).To(BeEmpty())

		po := &onboardingv1beta1.ProjectOnboarding{
			ObjectMeta: metav1.ObjectMeta{Name: "mapper-tenant"},
			Spec: onboardingv1beta1.ProjectOnboardingSpec{
				Namespaces: []onboardingv1beta1.NamespaceSpec{{
					Name:        "team-mapper",
					ProjectSize: "mapper-size",
				}},
			},
		}
		Expect(k8sClient.Create(context.Background(), po)).To(Succeed())
		defer func() { _ = k8sClient.Delete(context.Background(), po) }()

		reqs = reconciler.findProjectOnboardingsForTShirtSize(context.Background(), size)
		Expect(reqs).To(HaveLen(1))
		Expect(reqs[0].Name).To(Equal("mapper-tenant"))
	})

	It("enqueues ProjectOnboarding when a managed child carries onboarding labels", func() {
		reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		mapper := managedResourceEnqueueMapper(reconciler)

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ignored",
				Labels: map[string]string{
					onboardingv1beta1.ProjectOnboardingLabelKey:     "tenant-a",
					onboardingv1beta1.ProjectOnboardingManagedByKey: onboardingv1beta1.ProjectOnboardingManagedByVal,
				},
			},
		}
		reqs := mapper(context.Background(), ns)
		Expect(reqs).To(HaveLen(1))
		Expect(reqs[0].Name).To(Equal("tenant-a"))

		unmanaged := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "other", Labels: map[string]string{"app": "other"}}}
		Expect(mapper(context.Background(), unmanaged)).To(BeEmpty())
		Expect(onboarding.IsManagedResource(unmanaged)).To(BeFalse())
	})
})
