package main

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

func TestSchemeRegistersDistinctGVKs(t *testing.T) {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(onboardingv1beta1.AddToScheme(s))

	assertGVK(t, s, &onboardingv1beta1.ProjectOnboarding{}, schema.GroupVersionKind{
		Group: "onboarding.stderr.at", Version: "v1beta1", Kind: "ProjectOnboarding",
	})
	assertGVK(t, s, &onboardingv1beta1.TShirtSize{}, schema.GroupVersionKind{
		Group: "onboarding.stderr.at", Version: "v1beta1", Kind: "TShirtSize",
	})
}

func assertGVK(t *testing.T, s *runtime.Scheme, obj runtime.Object, want schema.GroupVersionKind) {
	t.Helper()
	gvks, _, err := s.ObjectKinds(obj)
	if err != nil {
		t.Fatalf("ObjectKinds: %v", err)
	}
	if len(gvks) != 1 {
		t.Fatalf("expected one GVK for %T, got %v", obj, gvks)
	}
	if gvks[0] != want {
		t.Fatalf("GVK for %T = %v, want %v", obj, gvks[0], want)
	}
}
