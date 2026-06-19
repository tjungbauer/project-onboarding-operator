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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Argo CD AppProject watch", func() {
	It("detects when AppProject API is not installed", func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: k8sClient.Scheme()})
		Expect(err).NotTo(HaveOccurred())
		Expect(isArgocdAppProjectAPIAvailable(mgr)).To(BeFalse())
	})

	It("registers ProjectOnboarding controller without AppProject CRD", func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: k8sClient.Scheme()})
		Expect(err).NotTo(HaveOccurred())

		reconciler := &ProjectOnboardingReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			Recorder: mgr.GetEventRecorderFor("projectonboarding-controller-test"),
		}
		Expect(reconciler.SetupWithManager(mgr)).To(Succeed())
	})
})
