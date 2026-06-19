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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

var _ = Describe("TShirtSize Controller", func() {
	It("should ignore a missing TShirtSize", func() {
		ctx := context.Background()
		reconciler := &TShirtSizeReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "missing-size"}})
		Expect(err).NotTo(HaveOccurred())
	})

	const sizeName = "test-size-s"

	ctx := context.Background()
	objectKey := types.NamespacedName{Name: sizeName}

	AfterEach(func() {
		size := &onboardingv1beta1.TShirtSize{}
		err := k8sClient.Get(ctx, objectKey, size)
		if err == nil {
			Expect(k8sClient.Delete(ctx, size)).To(Succeed())
		}
	})

	It("marks a valid catalogue entry Ready", func() {
		size := &onboardingv1beta1.TShirtSize{
			ObjectMeta: metav1.ObjectMeta{Name: sizeName},
			Spec: onboardingv1beta1.TShirtSizeSpec{
				ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
					Enabled: boolPtr(true),
					Pods:    int32Ptr(10),
				},
			},
		}
		Expect(k8sClient.Create(ctx, size)).To(Succeed())

		reconciler := &TShirtSizeReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClient.Get(ctx, objectKey, size)).To(Succeed())
		Expect(size.Status.Phase).To(Equal(onboardingv1beta1.TShirtSizePhaseReady))
		Expect(size.Status.ReferencedBy).To(Equal(int32(0)))
	})

	It("marks a sizing spec without values Invalid", func() {
		size := &onboardingv1beta1.TShirtSize{
			ObjectMeta: metav1.ObjectMeta{Name: sizeName},
			Spec: onboardingv1beta1.TShirtSizeSpec{
				ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{Enabled: boolPtr(true)},
			},
		}
		Expect(k8sClient.Create(ctx, size)).To(Succeed())

		reconciler := &TShirtSizeReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClient.Get(ctx, objectKey, size)).To(Succeed())
		Expect(size.Status.Phase).To(Equal(onboardingv1beta1.TShirtSizePhaseInvalid))
	})

	It("counts ProjectOnboarding references", func() {
		size := &onboardingv1beta1.TShirtSize{
			ObjectMeta: metav1.ObjectMeta{Name: sizeName},
			Spec: onboardingv1beta1.TShirtSizeSpec{
				ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
					Enabled: boolPtr(true),
					Pods:    int32Ptr(5),
				},
			},
		}
		Expect(k8sClient.Create(ctx, size)).To(Succeed())

		po := &onboardingv1beta1.ProjectOnboarding{
			ObjectMeta: metav1.ObjectMeta{Name: "ref-tenant"},
			Spec: onboardingv1beta1.ProjectOnboardingSpec{
				Namespaces: []onboardingv1beta1.NamespaceSpec{{
					Name:        "team-ref",
					ProjectSize: sizeName,
				}},
			},
		}
		Expect(k8sClient.Create(ctx, po)).To(Succeed())
		defer func() {
			_ = k8sClient.Delete(ctx, po)
		}()

		reconciler := &TShirtSizeReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClient.Get(ctx, objectKey, size)).To(Succeed())
		Expect(size.Status.ReferencedBy).To(Equal(int32(1)))
	})
})
