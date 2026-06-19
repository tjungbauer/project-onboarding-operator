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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tjungbauer/project-onboarding-operator/test/utils"
)

const (
	envOpenShiftE2E = "OPENSHIFT_E2E"
	envOperatorNS   = "OPERATOR_NS"
)

func openshiftE2EEnabled() bool {
	return os.Getenv(envOpenShiftE2E) == "true"
}

func operatorNamespace() string {
	if ns := strings.TrimSpace(os.Getenv(envOperatorNS)); ns != "" {
		return ns
	}
	return "project-onboarding-operator"
}

func openshiftManifest(name string) string {
	dir, err := utils.GetProjectDir()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(dir, "test", "openshift", "manifests", name)
}

func openshiftCleanupScript() string {
	dir, err := utils.GetProjectDir()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(dir, "test", "openshift", "cleanup.sh")
}

func kubectlOutput(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	return utils.Run(cmd)
}

func kubectlApplyFile(path string) {
	cmd := exec.Command("kubectl", "apply", "-f", path)
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "kubectl apply %s", path)
}

func kubectlDeleteFile(path string) {
	cmd := exec.Command("kubectl", "delete", "-f", path, "--ignore-not-found")
	_, _ = utils.Run(cmd)
}

func kubectlApplyManifestShouldFail(manifest string) {
	cmd := exec.Command("kubectl", "apply", "--dry-run=server", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).To(HaveOccurred())
}

func kubectlDeleteDryRunShouldFail(args ...string) {
	cmdArgs := append([]string{"delete", "--dry-run=server"}, args...)
	cmd := exec.Command("kubectl", cmdArgs...)
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).To(HaveOccurred())
}

func waitProjectOnboardingReady(name string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		out, err := kubectlOutput("get", "projectonboarding", name, "-o", "jsonpath={.status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(Equal("Ready"))
	}, timeout, time.Second).Should(Succeed())
}

func waitTShirtSizeReady(name string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		out, err := kubectlOutput("get", "tshirtsize", name, "-o", "jsonpath={.status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(Equal("Ready"))
	}, timeout, time.Second).Should(Succeed())
}

func waitNamespaceDeleted(name string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		_, err := kubectlOutput("get", "namespace", name)
		g.Expect(err).To(HaveOccurred())
	}, timeout, time.Second).Should(Succeed())
}

func networkPolicyNames(namespace string) []string {
	out, err := kubectlOutput("get", "networkpolicy", "-n", namespace, "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
	Expect(err).NotTo(HaveOccurred())
	names := utils.GetNonEmptyLines(out)
	sort.Strings(names)
	return names
}

func ensureNamespace(name string) {
	cmd := exec.Command("kubectl", "create", "namespace", name, "--dry-run=client", "-o", "yaml")
	manifest, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	apply := exec.Command("kubectl", "apply", "-f", "-")
	apply.Stdin = strings.NewReader(manifest)
	_, err = utils.Run(apply)
	Expect(err).NotTo(HaveOccurred())
}

func runOpenShiftCleanup() {
	script := openshiftCleanupScript()
	cmd := exec.Command("bash", script)
	_, _ = utils.Run(cmd)
}

func dumpOperatorLogsOnFailure() {
	if !CurrentSpecReport().Failed() {
		return
	}
	ns := operatorNamespace()
	out, err := kubectlOutput("logs", "-n", ns, "-l", "control-plane=controller-manager", "-c", "manager", "--tail=80")
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "operator logs:\n%s\n", out)
	}
}

// OLM registers kubebuilder webhooks with generated suffixes (e.g. vprojectonboarding.kb.io-4knj6).
var operatorValidatingWebhookPrefixes = []string{
	"vprojectonboarding.kb.io",
	"vprojectonboardingv1beta1.kb.io",
	"vtshirtsize.kb.io",
	"vtshirtsizev1beta1.kb.io",
}

func expectOperatorValidatingWebhooks() {
	out, err := kubectlOutput("get", "validatingwebhookconfiguration", "-o", "name")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	for _, prefix := range operatorValidatingWebhookPrefixes {
		ExpectWithOffset(1, out).To(ContainSubstring(prefix), "missing validating webhook %q", prefix)
	}
}
