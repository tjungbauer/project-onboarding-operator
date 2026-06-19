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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

var _ = Describe("ProjectOnboarding Controller", func() {
	It("should ignore a missing ProjectOnboarding", func() {
		ctx := context.Background()
		reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "does-not-exist"}})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const targetNamespace = "team-test-dev"

		ctx := context.Background()

		objectKey := types.NamespacedName{Name: resourceName}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ProjectOnboarding")
			po := &onboardingv1beta1.ProjectOnboarding{}
			err := k8sClient.Get(ctx, objectKey, po)
			if err != nil && apierrors.IsNotFound(err) {
				resource := &onboardingv1beta1.ProjectOnboarding{
					ObjectMeta: metav1.ObjectMeta{Name: resourceName},
					Spec: onboardingv1beta1.ProjectOnboardingSpec{
						Namespaces: []onboardingv1beta1.NamespaceSpec{{
							Name: targetNamespace,
							ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
								Enabled: boolPtr(true),
								Pods:    int32Ptr(10),
								CPU:     strPtr("2"),
								Memory:  strPtr("4Gi"),
							},
							DefaultPolicies: &onboardingv1beta1.DefaultPoliciesSpec{
								AllowFromMonitoring:    boolPtr(false),
								AllowKubeAPIServer:     boolPtr(false),
								AllowToDNS:             boolPtr(false),
								AllowFromSameNamespace: boolPtr(true),
							},
						}},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &onboardingv1beta1.ProjectOnboarding{}
			err := k8sClient.Get(ctx, objectKey, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}

			ns := &corev1.Namespace{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: targetNamespace}, ns)
			if err == nil {
				Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
			}
		})

		It("should freeze disabled namespaces without teardown and still report Ready", func() {
			disabledPO := &onboardingv1beta1.ProjectOnboarding{
				ObjectMeta: metav1.ObjectMeta{Name: "test-disabled"},
				Spec: onboardingv1beta1.ProjectOnboardingSpec{
					Namespaces: []onboardingv1beta1.NamespaceSpec{
						{Name: "team-disabled-a", Enabled: boolPtr(false)},
						{
							Name:    "team-disabled-b",
							Enabled: boolPtr(true),
							ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
								Enabled: boolPtr(true),
								Pods:    int32Ptr(5),
							},
							DefaultPolicies: &onboardingv1beta1.DefaultPoliciesSpec{
								AllowFromMonitoring:    boolPtr(false),
								AllowKubeAPIServer:     boolPtr(false),
								AllowToDNS:             boolPtr(false),
								AllowFromSameNamespace: boolPtr(true),
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, disabledPO)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, disabledPO)
				ns := &corev1.Namespace{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "team-disabled-b"}, ns); err == nil {
					_ = k8sClient.Delete(ctx, ns)
				}
			}()

			reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-disabled"}})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "team-disabled-a"}, &corev1.Namespace{})).To(Not(Succeed()))
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "team-disabled-b"}, &corev1.Namespace{})).To(Succeed())

			po := &onboardingv1beta1.ProjectOnboarding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-disabled"}, po)).To(Succeed())
			Expect(po.Status.Phase).To(Equal(onboardingv1beta1.PhaseReady))
			Expect(po.Status.Namespaces).To(HaveLen(2))
			Expect(po.Status.Namespaces[0].Message).To(Equal("reconciliation disabled"))
		})

		It("should tear down namespaces when offboard is true", func() {
			offboardPO := &onboardingv1beta1.ProjectOnboarding{
				ObjectMeta: metav1.ObjectMeta{Name: "test-offboard"},
				Spec: onboardingv1beta1.ProjectOnboardingSpec{
					Namespaces: []onboardingv1beta1.NamespaceSpec{{
						Name:     "team-offboard-dev",
						Offboard: boolPtr(true),
						DefaultPolicies: &onboardingv1beta1.DefaultPoliciesSpec{
							AllowFromMonitoring:    boolPtr(false),
							AllowKubeAPIServer:     boolPtr(false),
							AllowToDNS:             boolPtr(false),
							AllowFromSameNamespace: boolPtr(true),
						},
					}},
				},
			}
			Expect(k8sClient.Create(ctx, offboardPO)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, offboardPO)
			}()

			reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-offboard"}})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "team-offboard-dev"}, &corev1.Namespace{})).To(Not(Succeed()))

			po := &onboardingv1beta1.ProjectOnboarding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-offboard"}, po)).To(Succeed())
			Expect(po.Status.Phase).To(Equal(onboardingv1beta1.PhaseReady))
			Expect(po.Status.Namespaces[0].Message).To(Equal("offboarded (resources removed)"))
		})

		It("should offboard a previously reconciled namespace", func() {
			const offboardNS = "team-offboard-existing"
			activePO := &onboardingv1beta1.ProjectOnboarding{
				ObjectMeta: metav1.ObjectMeta{Name: "test-offboard-existing"},
				Spec: onboardingv1beta1.ProjectOnboardingSpec{
					Namespaces: []onboardingv1beta1.NamespaceSpec{{
						Name: offboardNS,
						DefaultPolicies: &onboardingv1beta1.DefaultPoliciesSpec{
							AllowFromMonitoring:    boolPtr(false),
							AllowKubeAPIServer:     boolPtr(false),
							AllowToDNS:             boolPtr(false),
							AllowFromSameNamespace: boolPtr(true),
						},
					}},
				},
			}
			Expect(k8sClient.Create(ctx, activePO)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, activePO)
			}()

			reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-offboard-existing"}})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: offboardNS}, &corev1.Namespace{})).To(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-offboard-existing"}, activePO)).To(Succeed())
			activePO.Spec.Namespaces[0].Offboard = boolPtr(true)
			Expect(k8sClient.Update(ctx, activePO)).To(Succeed())

			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-offboard-existing"}})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				ns := &corev1.Namespace{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: offboardNS}, ns)
				if apierrors.IsNotFound(err) {
					return
				}
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ns.DeletionTimestamp).NotTo(BeZero())
			}).Should(Succeed())
		})

		It("should create the target namespace and update status", func() {
			controllerReconciler := &ProjectOnboardingReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).NotTo(HaveOccurred())

			ns := &corev1.Namespace{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: targetNamespace}, ns)).To(Succeed())

			po := &onboardingv1beta1.ProjectOnboarding{}
			Expect(k8sClient.Get(ctx, objectKey, po)).To(Succeed())
			Expect(po.Status.Phase).To(Equal(onboardingv1beta1.PhaseReady))
			Expect(po.Status.Namespaces).To(HaveLen(1))
			Expect(po.Status.Namespaces[0].Ready).To(BeTrue())
		})

	})

	Context("When deleting a reconciled resource", func() {
		ctx := context.Background()

		createReconciledPO := func(resourceName, targetNamespace string) types.NamespacedName {
			objectKey := types.NamespacedName{Name: resourceName}
			po := &onboardingv1beta1.ProjectOnboarding{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName},
				Spec: onboardingv1beta1.ProjectOnboardingSpec{
					Namespaces: []onboardingv1beta1.NamespaceSpec{{
						Name: targetNamespace,
						DefaultPolicies: &onboardingv1beta1.DefaultPoliciesSpec{
							AllowFromIngress:       boolPtr(false),
							AllowFromMonitoring:    boolPtr(false),
							AllowKubeAPIServer:     boolPtr(false),
							AllowToDNS:             boolPtr(false),
							AllowFromSameNamespace: boolPtr(true),
							DenyAllEgress:          boolPtr(false),
							DenyAllIngress:         boolPtr(false),
						},
					}},
				},
			}
			Expect(k8sClient.Create(ctx, po)).To(Succeed())

			reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).NotTo(HaveOccurred())
			return objectKey
		}

		forceRemovePO := func(objectKey types.NamespacedName, targetNamespace string) {
			po := &onboardingv1beta1.ProjectOnboarding{}
			err := k8sClient.Get(ctx, objectKey, po)
			if err != nil {
				return
			}
			po.Spec.Namespaces[0].Offboard = boolPtr(true)
			Expect(k8sClient.Update(ctx, po)).To(Succeed())
			if po.DeletionTimestamp.IsZero() {
				Expect(k8sClient.Delete(ctx, po)).To(Succeed())
			}
			reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			Eventually(func(g Gomega) {
				_, recErr := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
				g.Expect(recErr).NotTo(HaveOccurred())
				getErr := k8sClient.Get(ctx, objectKey, po)
				g.Expect(apierrors.IsNotFound(getErr)).To(BeTrue())
			}).Should(Succeed())

			ns := &corev1.Namespace{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: targetNamespace}, ns); err == nil {
				_ = k8sClient.Delete(ctx, ns)
			}
		}

		It("should block CR deletion while managed namespaces exist", func() {
			const resourceName = "test-delete-block"
			const targetNamespace = "team-delete-block"
			objectKey := createReconciledPO(resourceName, targetNamespace)
			defer forceRemovePO(objectKey, targetNamespace)

			reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			po := &onboardingv1beta1.ProjectOnboarding{}
			Expect(k8sClient.Get(ctx, objectKey, po)).To(Succeed())
			Expect(k8sClient.Delete(ctx, po)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: targetNamespace}, &corev1.Namespace{})).To(Succeed())
			Expect(k8sClient.Get(ctx, objectKey, po)).To(Succeed())
			Expect(po.DeletionTimestamp).NotTo(BeZero())
		})

		It("should allow CR deletion after offboard is set while terminating", func() {
			const resourceName = "test-delete-offboard"
			const targetNamespace = "team-delete-offboard"
			objectKey := createReconciledPO(resourceName, targetNamespace)
			defer forceRemovePO(objectKey, targetNamespace)

			reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			po := &onboardingv1beta1.ProjectOnboarding{}
			Expect(k8sClient.Get(ctx, objectKey, po)).To(Succeed())
			Expect(k8sClient.Delete(ctx, po)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, objectKey, po)).To(Succeed())
			po.Spec.Namespaces[0].Offboard = boolPtr(true)
			Expect(k8sClient.Update(ctx, po)).To(Succeed())

			Eventually(func(g Gomega) {
				_, recErr := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
				g.Expect(recErr).NotTo(HaveOccurred())
				nsErr := k8sClient.Get(ctx, types.NamespacedName{Name: targetNamespace}, &corev1.Namespace{})
				if !apierrors.IsNotFound(nsErr) {
					g.Expect(nsErr).NotTo(HaveOccurred())
					ns := &corev1.Namespace{}
					g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: targetNamespace}, ns)).To(Succeed())
					g.Expect(ns.DeletionTimestamp).NotTo(BeZero())
				}
				poErr := k8sClient.Get(ctx, objectKey, po)
				g.Expect(apierrors.IsNotFound(poErr)).To(BeTrue())
			}).Should(Succeed())
		})

		It("should allow CR deletion after the tenant namespace is deleted manually", func() {
			const resourceName = "test-delete-manual"
			const targetNamespace = "team-delete-manual"
			objectKey := createReconciledPO(resourceName, targetNamespace)
			defer forceRemovePO(objectKey, targetNamespace)

			reconciler := &ProjectOnboardingReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			ns := &corev1.Namespace{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: targetNamespace}, ns)).To(Succeed())
			Expect(k8sClient.Delete(ctx, ns)).To(Succeed())

			po := &onboardingv1beta1.ProjectOnboarding{}
			Expect(k8sClient.Get(ctx, objectKey, po)).To(Succeed())
			Expect(k8sClient.Delete(ctx, po)).To(Succeed())

			Eventually(func(g Gomega) {
				_, recErr := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
				g.Expect(recErr).NotTo(HaveOccurred())
				poErr := k8sClient.Get(ctx, objectKey, po)
				g.Expect(apierrors.IsNotFound(poErr)).To(BeTrue())
			}).Should(Succeed())
		})
	})
})

func boolPtr(v bool) *bool    { return &v }
func int32Ptr(v int32) *int32 { return &v }
func strPtr(v string) *string { return &v }
