package conversion

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type metaObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              map[string]string `json:"spec,omitempty"`
}

func TestViaJSONStripsAPIVersionAndKind(t *testing.T) {
	t.Parallel()

	src := metaObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "onboarding.stderr.at/v1beta1",
			Kind:       "ProjectOnboarding",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec:       map[string]string{"key": "value"},
	}

	dst := metaObject{}
	if err := ViaJSON(&src, &dst); err != nil {
		t.Fatalf("ViaJSON: %v", err)
	}
	if dst.APIVersion != "" || dst.Kind != "" {
		t.Fatalf("expected empty TypeMeta, got apiVersion=%q kind=%q", dst.APIVersion, dst.Kind)
	}
	if dst.Name != "tenant-a" {
		t.Fatalf("name = %q", dst.Name)
	}
}
