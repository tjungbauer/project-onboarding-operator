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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tjungbauer/project-onboarding-operator/test/utils"
)

// namespace where the project is deployed in
const namespace = "project-onboarding-operator"

// serviceAccountName created for the project
const serviceAccountName = "project-onboarding-operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "project-onboarding-operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "project-onboarding-operator-metrics-binding"

// curlE2EImage is preloaded into Kind in BeforeSuite so the metrics probe does not depend on registry pulls.
const curlE2EImage = "curlimages/curl:8.12.1"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		if openshiftE2EEnabled() {
			Skip("Kind manager tests skipped when OPENSHIFT_E2E=true; run OpenShift test cases instead")
		}

		By("creating manager namespace")
		nsManifest := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    pod-security.kubernetes.io/enforce: restricted
    metrics: enabled
`, namespace)
		cmd := exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(nsManifest)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to ensure manager namespace")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("cleaning up the metrics ClusterRoleBinding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found")
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy-e2e")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall-e2e")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--ignore-not-found", "--wait=false")
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			_, _ = utils.Run(exec.Command(
				"kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found",
			))
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=project-onboarding-operator-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(SatisfyAny(
					ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					ContainSubstring(`"logger":"controller-runtime.metrics"`),
				), "Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("creating the curl-metrics pod to access the metrics endpoint")
			_, _ = utils.Run(exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace, "--ignore-not-found"))
			Expect(createCurlMetricsPod(token)).To(Succeed(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				if output == "Failed" {
					logs, _ := utils.Run(exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace))
					g.Expect(output).To(Equal("Succeeded"), "curl pod failed:\n%s", logs)
				}
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 2*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring(
				"controller_runtime_reconcile_total",
			))
		})

		It("should reject a TShirtSize without sizing via API validation", func() {
			By("applying an invalid TShirtSize with server dry-run")
			manifest := `apiVersion: onboarding.stderr.at/v1alpha1
kind: TShirtSize
metadata:
  name: e2e-invalid-size
spec: {}
`
			cmd := exec.Command("kubectl", "apply", "--dry-run=server", "-f", "-")
			cmd.Stdin = strings.NewReader(manifest)
			_, err := utils.Run(cmd)
			Expect(err).To(HaveOccurred(), "invalid TShirtSize should be rejected")
		})

	})

	Context("ProjectOnboarding", func() {
		const (
			crName       = "e2e-onboarding"
			targetNS     = "e2e-team-dev"
			testdataPath = "test/e2e/testdata/projectonboarding_e2e.yaml"
		)

		AfterEach(func() {
			By("deleting the ProjectOnboarding CR if it still exists")
			cmd := exec.Command("kubectl", "delete", "-f", testdataPath, "--ignore-not-found", "--wait=false")
			_, _ = utils.Run(cmd)
		})

		It("should reconcile onboarding resources and clean them up on delete", func() {
			By("applying a ProjectOnboarding custom resource")
			cmd := exec.Command("kubectl", "apply", "-f", testdataPath)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply ProjectOnboarding CR")

			By("waiting for the ProjectOnboarding status to become Ready")
			verifyReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "projectonboarding", crName,
					"-o", "jsonpath={.status.phase}",
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Ready"))
			}
			Eventually(verifyReady, 3*time.Minute).Should(Succeed())

			By("validating the onboarded namespace exists")
			cmd = exec.Command("kubectl", "get", "namespace", targetNS)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Onboarded namespace should exist")

			By("validating the resource quota was created")
			cmd = exec.Command("kubectl", "get", "resourcequota", targetNS+"-quota", "-n", targetNS)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Resource quota should exist")

			By("offboarding the tenant namespace before CR delete")
			offboardPatch := `[{"op":"replace","path":"/spec/namespaces/0/offboard","value":true}]`
			cmd = exec.Command("kubectl", "patch", "projectonboarding", crName, "--type=json", "-p", offboardPatch)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to patch ProjectOnboarding for offboard")

			By("waiting for the onboarded namespace to be removed after offboard")
			verifyNamespaceDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "namespace", targetNS)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred())
			}
			Eventually(verifyNamespaceDeleted, 3*time.Minute).Should(Succeed())

			By("deleting the ProjectOnboarding CR")
			cmd = exec.Command("kubectl", "delete", "-f", testdataPath, "--timeout=120s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete ProjectOnboarding CR")

			By("waiting for the ProjectOnboarding CR to be removed")
			verifyCRDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "projectonboarding", crName)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred())
			}
			Eventually(verifyCRDeleted, time.Minute).Should(Succeed())
		})
	})
})

// createCurlMetricsPod runs a one-shot curl pod that scrapes the operator metrics endpoint.
func createCurlMetricsPod(token string) error {
	metricsURL := fmt.Sprintf("https://%s.%s.svc.cluster.local:8443/metrics", metricsServiceName, namespace)
	curlCmd := fmt.Sprintf(
		`curl -v -k -f --connect-timeout 15 -H "Authorization: Bearer ${METRICS_TOKEN}" %s`,
		metricsURL,
	)
	overrides := map[string]any{
		"spec": map[string]any{
			"serviceAccountName": serviceAccountName,
			"containers": []map[string]any{
				{
					"name":            "curl",
					"image":           curlE2EImage,
					"imagePullPolicy": "IfNotPresent",
					"command":         []string{"/bin/sh", "-c"},
					"args":            []string{curlCmd},
					"env": []map[string]string{
						{"name": "METRICS_TOKEN", "value": token},
					},
					"securityContext": map[string]any{
						"allowPrivilegeEscalation": false,
						"capabilities": map[string]any{
							"drop": []string{"ALL"},
						},
						"runAsNonRoot": true,
						"runAsUser":    101,
						"seccompProfile": map[string]string{
							"type": "RuntimeDefault",
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(overrides)
	if err != nil {
		return err
	}

	cmd := exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
		"--namespace", namespace,
		"--image="+curlE2EImage,
		"--overrides", string(raw),
	)
	_, err = utils.Run(cmd)
	return err
}

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
