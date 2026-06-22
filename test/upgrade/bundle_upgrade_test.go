package upgrade_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Validates the committed OLM bundle declares a semver upgrade path (replaces).
func TestBundleCSVReplacesPreviousVersion(t *testing.T) {
	csv := filepath.Join("..", "..", "bundle", "manifests", "project-onboarding-operator.clusterserviceversion.yaml")
	data, err := os.ReadFile(csv)
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "replaces:") {
		t.Fatal("CSV missing replaces: field for OLM upgrade path")
	}
	versionFile := filepath.Join("..", "..", "VERSION")
	ver, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	current := strings.TrimSpace(string(ver))
	if !strings.Contains(content, "version: "+current) {
		t.Fatalf("CSV version does not match VERSION file (%s)", current)
	}
}
