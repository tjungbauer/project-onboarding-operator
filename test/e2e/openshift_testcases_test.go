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

package e2e

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tjungbauer/project-onboarding-operator/test/utils"
)

const (
	manifestTC01 = "tc01-core-onboarding.yaml"
	manifestTC02 = "tc02-tshirt-catalog.yaml"
	manifestTC03 = "tc03-tshirt-onboarding.yaml"
	manifestTC04 = "tc04-openshift-features.yaml"
	manifestTC05 = "tc05-custom-netpol.yaml"
	manifestTC12 = "tc12-api-conversion-v1alpha1.yaml"
	manifestTC13 = "tc13-gitops-onboarding.yaml"

	crTC01 = "tc01-core-onboarding"
	crTC03 = "tc03-tshirt-onboarding"
	crTC04 = "tc04-openshift-features"
	crTC05 = "tc05-custom-netpol"
	crTC12 = "tc12-conversion-test"
	crTC13 = "tc13-gitops-onboarding"

	nsTC01 = "ocp-test-core-dev"
	nsTC03 = "ocp-test-medium-dev"
	nsTC04 = "ocp-test-egress-dev"
	nsTC05 = "ocp-test-netpol-dev"
	nsTC12 = "ocp-test-conversion-dev"
	nsTC13 = "ocp-test-gitops-dev"

	ttsTC02 = "ocp-test-medium"
)

var _ = Describe("OpenShift test cases", Ordered, func() {
	BeforeAll(func() {
		if !openshiftE2EEnabled() {
			Skip("OpenShift E2E disabled; set OPENSHIFT_E2E=true. See docs/openshift-testcases.md")
		}
		runOpenShiftCleanup()
	})

	AfterEach(func() {
		dumpOperatorLogsOnFailure()
	})

	AfterAll(func() {
		if !openshiftE2EEnabled() {
			return
		}
		runOpenShiftCleanup()
	})

	SetDefaultEventuallyTimeout(3 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	It("TC-00 should verify operator health", func() {
		ns := operatorNamespace()

		By("checking CSV phase")
		out, err := kubectlOutput("get", "csv", "-n", ns, "-o", "jsonpath={range .items[*]}{.metadata.name}{\" \"}{.status.phase}{\"\\n\"}{end}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("Succeeded"))

		By("checking controller pod is Running")
		Eventually(func(g Gomega) {
			out, err := kubectlOutput("get", "pods", "-n", ns, "-l", "control-plane=controller-manager",
				"-o", "jsonpath={.items[0].status.phase}")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out).To(Equal("Running"))
		}).Should(Succeed())

		By("checking CRD storage version is v1beta1")
		out, err = kubectlOutput("get", "crd", "projectonboardings.onboarding.stderr.at",
			"-o", "jsonpath={.spec.versions[?(@.storage==true)].name}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("v1beta1"))

		By("checking conversion webhook strategy")
		out, err = kubectlOutput("get", "crd", "projectonboardings.onboarding.stderr.at",
			"-o", "jsonpath={.spec.conversion.strategy}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("Webhook"))

		By("checking validating webhooks for v1alpha1 and v1beta1")
		expectOperatorValidatingWebhooks()

		By("checking operator NetworkPolicies")
		out, err = kubectlOutput("get", "networkpolicy", "-n", ns, "-o", "name")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("allow-metrics-traffic"))
		Expect(out).To(ContainSubstring("allow-webhook-traffic"))
	})

	It("TC-01 should reconcile core onboarding resources", func() {
		kubectlApplyFile(openshiftManifest(manifestTC01))
		waitProjectOnboardingReady(crTC01, 3*time.Minute)

		out, err := kubectlOutput("get", "projectonboarding", crTC01, "-o", "jsonpath={.apiVersion}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("onboarding.stderr.at/v1beta1"))

		_, err = kubectlOutput("get", "namespace", nsTC01)
		Expect(err).NotTo(HaveOccurred())

		_, err = kubectlOutput("get", "resourcequota", nsTC01+"-quota", "-n", nsTC01)
		Expect(err).NotTo(HaveOccurred())

		_, err = kubectlOutput("get", "limitrange", nsTC01+"-limitrange", "-n", nsTC01)
		Expect(err).NotTo(HaveOccurred())

		names := networkPolicyNames(nsTC01)
		Expect(names).To(ContainElements(
			"allow-from-kube-apiserver-operator",
			"allow-from-openshift-ingress",
			"allow-from-openshift-monitoring",
			"allow-same-namespace",
			"allow-to-openshift-dns",
		))
	})

	It("TC-02 should reconcile TShirtSize catalogue entry", func() {
		kubectlApplyFile(openshiftManifest(manifestTC02))
		waitTShirtSizeReady(ttsTC02, 2*time.Minute)

		out, err := kubectlOutput("get", "tshirtsize", ttsTC02, "-o", "jsonpath={.apiVersion}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("onboarding.stderr.at/v1beta1"))

		out, err = kubectlOutput("get", "tshirtsize", ttsTC02, "-o", "jsonpath={.status.referencedBy}")
		Expect(err).NotTo(HaveOccurred())
		if out == "" {
			out = "0" // status.referencedBy omits zero values in JSON
		}
		Expect(out).To(Equal("0"))
	})

	It("TC-03 should merge T-shirt sizing with overwriteTshirt", func() {
		kubectlApplyFile(openshiftManifest(manifestTC03))
		waitProjectOnboardingReady(crTC03, 3*time.Minute)

		out, err := kubectlOutput("get", "namespace", nsTC03, "-o", "jsonpath={.metadata.labels.namespace-size}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(ttsTC02))

		out, err = kubectlOutput("get", "resourcequota", nsTC03+"-quota", "-n", nsTC03,
			"-o", "jsonpath={.spec.hard.cpu}{\" \"}{.spec.hard.memory}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("3 4Gi"))

		out, err = kubectlOutput("get", "tshirtsize", ttsTC02, "-o", "jsonpath={.status.referencedBy}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("1"))

		names := networkPolicyNames(nsTC03)
		Expect(names).To(Equal([]string{"allow-same-namespace"}))
	})

	It("TC-04 should reconcile OpenShift Group, RoleBinding, and EgressIP", func() {
		kubectlApplyFile(openshiftManifest(manifestTC04))
		waitProjectOnboardingReady(crTC04, 3*time.Minute)

		_, err := kubectlOutput("get", "group", nsTC04+"-admins")
		Expect(err).NotTo(HaveOccurred())

		out, err := kubectlOutput("get", "rolebinding", nsTC04+"-rb", "-n", nsTC04,
			"-o", "jsonpath={.roleRef.name}{\" \"}{.subjects[0].name}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("admin " + nsTC04 + "-admins"))

		_, err = kubectlOutput("get", "egressip", nsTC04)
		Expect(err).NotTo(HaveOccurred())

		out, err = kubectlOutput("get", "namespace", nsTC04, "-o", "jsonpath={.metadata.labels.env}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(nsTC04))
	})

	It("TC-05 should reconcile custom NetworkPolicy", func() {
		kubectlApplyFile(openshiftManifest(manifestTC05))
		waitProjectOnboardingReady(crTC05, 3*time.Minute)

		out, err := kubectlOutput("get", "networkpolicy", "allow-from-openshift-monitoring-custom", "-n", nsTC05,
			"-o", "jsonpath={.spec.podSelector.matchLabels.app}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("metrics-exporter"))

		names := networkPolicyNames(nsTC05)
		Expect(names).To(ContainElements(
			"allow-same-namespace",
			"allow-to-openshift-dns",
			"allow-from-openshift-monitoring-custom",
		))
	})

	It("TC-06 should reject invalid TShirtSize via admission", func() {
		kubectlApplyManifestShouldFail(`apiVersion: onboarding.stderr.at/v1beta1
kind: TShirtSize
metadata:
  name: tc06-invalid
spec: {}
`)
	})

	It("TC-07 should reject unknown projectSize via webhook", func() {
		kubectlApplyManifestShouldFail(`apiVersion: onboarding.stderr.at/v1beta1
kind: ProjectOnboarding
metadata:
  name: tc07-bad-size
spec:
  namespaces:
    - name: should-not-exist
      projectSize: does-not-exist
`)
	})

	It("TC-08 should block TShirtSize delete while referenced", func() {
		kubectlDeleteDryRunShouldFail("tshirtsize", ttsTC02)
	})

	It("TC-09 should remove tenant namespace when ProjectOnboarding is deleted", func() {
		kubectlDeleteFile(openshiftManifest(manifestTC01))
		waitNamespaceDeleted(nsTC01, 3*time.Minute)
	})

	It("TC-10 should restore drifted ResourceQuota", func() {
		waitProjectOnboardingReady(crTC03, 3*time.Minute)
		_, err := kubectlOutput("get", "resourcequota", nsTC03+"-quota", "-n", nsTC03)
		Expect(err).NotTo(HaveOccurred())

		cmdPatch := []string{"patch", "resourcequota", nsTC03 + "-quota", "-n", nsTC03,
			"--type=merge", "-p", `{"spec":{"hard":{"cpu":"99"}}}`}
		_, err = kubectlOutput(cmdPatch...)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			out, err := kubectlOutput("get", "resourcequota", nsTC03+"-quota", "-n", nsTC03,
				"-o", "jsonpath={.spec.hard.cpu}")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out).To(Equal("3"))
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	})

	It("TC-11 should propagate TShirtSize updates to tenant quota", func() {
		_, err := kubectlOutput("patch", "tshirtsize", ttsTC02, "--type=merge",
			"-p", `{"spec":{"resourceQuotas":{"memory":"6Gi"}}}`)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			out, err := kubectlOutput("get", "resourcequota", nsTC03+"-quota", "-n", nsTC03,
				"-o", "jsonpath={.spec.hard.memory}")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out).To(Equal("6Gi"))
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	})

	It("TC-12 should convert v1alpha1 apply to v1beta1 storage", func() {
		kubectlApplyFile(openshiftManifest(manifestTC12))
		waitProjectOnboardingReady(crTC12, 3*time.Minute)

		out, err := kubectlOutput("get", "projectonboarding", crTC12, "-o", "jsonpath={.apiVersion}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("onboarding.stderr.at/v1beta1"))

		raw, err := kubectlOutput("get", "--raw",
			"/apis/onboarding.stderr.at/v1alpha1/projectonboardings/"+crTC12)
		Expect(err).NotTo(HaveOccurred())
		Expect(raw).To(MatchRegexp(`"apiVersion"\s*:\s*"onboarding.stderr.at/v1alpha1"`))

		_, err = kubectlOutput("get", "namespace", nsTC12)
		Expect(err).NotTo(HaveOccurred())

		_, err = kubectlOutput("get", "resourcequota", nsTC12+"-quota", "-n", nsTC12)
		Expect(err).NotTo(HaveOccurred())
	})

	It("TC-13 should reconcile GitOps AppProject when Argo CD CRD is present", func() {
		if !utils.IsCRDInstalled("appprojects.argoproj.io") {
			Skip("appprojects.argoproj.io not installed; skipping GitOps test")
		}

		const argoNS = "gitops-application"
		ensureNamespace(argoNS)

		kubectlApplyFile(openshiftManifest(manifestTC13))
		waitProjectOnboardingReady(crTC13, 3*time.Minute)

		out, err := kubectlOutput("get", "namespace", nsTC13,
			"-o", "jsonpath={.metadata.labels.argocd\\.argoproj\\.io/managed-by}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(argoNS))

		_, err = kubectlOutput("get", "appproject", nsTC13, "-n", argoNS)
		Expect(err).NotTo(HaveOccurred())

		out, err = kubectlOutput("get", "appproject", nsTC13, "-n", argoNS,
			"-o", "jsonpath={.spec.destinations[0].namespace}")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(nsTC13))

		out, err = kubectlOutput("get", "appproject", nsTC13, "-n", argoNS,
			"-o", "jsonpath={.spec.roles[*].name}")
		Expect(err).NotTo(HaveOccurred())
		for _, role := range []string{"write", "read"} {
			Expect(strings.Split(out, " ")).To(ContainElement(role))
		}
	})

	It("TC-14 should expose observability resources when Prometheus Operator is present", func() {
		if !utils.IsPrometheusCRDsInstalled() {
			Skip("Prometheus Operator CRDs not installed; skipping observability test")
		}

		ns := operatorNamespace()
		_, err := kubectlOutput("get", "servicemonitor",
			"project-onboarding-operator-controller-manager-metrics-monitor", "-n", ns)
		Expect(err).NotTo(HaveOccurred())

		_, err = kubectlOutput("get", "prometheusrule",
			"project-onboarding-operator-controller-manager-rules", "-n", ns)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			out, err := kubectlOutput("get", "endpoints", "-n", ns,
				"project-onboarding-operator-controller-manager-metrics-service",
				"-o", "jsonpath={.subsets[0].ports[0].port}")
			g.Expect(err).NotTo(HaveOccurred())
			lines := utils.GetNonEmptyLines(out)
			g.Expect(lines[len(lines)-1]).To(Equal("8443"))
		}).Should(Succeed())
	})
})
