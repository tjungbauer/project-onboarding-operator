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

package onboarding

import (
	"context"
	"strings"
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFinalizeProjectOnboardingDeletionBlockedWhenNamespaceExists(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name: "team-active",
			}},
		},
	}
	labels := ManagedLabels(po, onboardingv1beta1.NamespaceSpec{Name: "team-active"})
	managedNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "team-active", Labels: labels},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po, managedNS).Build()
	ctx := context.Background()

	complete, pendingMessage, err := FinalizeProjectOnboardingDeletion(ctx, c, po)
	if err != nil {
		t.Fatalf("finalize deletion: %v", err)
	}
	if complete {
		t.Fatal("expected deletion to be blocked")
	}
	if !strings.Contains(pendingMessage, "team-active") {
		t.Fatalf("expected pending namespace in message, got %q", pendingMessage)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: "team-active"}, ns); err != nil {
		t.Fatalf("namespace should remain, got err=%v", err)
	}
}

func TestFinalizeProjectOnboardingDeletionCompletesWhenOffboarded(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name:     "team-offboard",
				Offboard: boolPtr(true),
			}},
		},
	}
	labels := ManagedLabels(po, onboardingv1beta1.NamespaceSpec{Name: "team-offboard"})
	managedNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "team-offboard", Labels: labels},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po, managedNS).Build()
	ctx := context.Background()

	complete, pendingMessage, err := FinalizeProjectOnboardingDeletion(ctx, c, po)
	if err != nil {
		t.Fatalf("finalize deletion: %v", err)
	}
	if !complete {
		t.Fatalf("expected deletion to complete, pending=%q", pendingMessage)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: "team-offboard"}, ns); !apierrors.IsNotFound(err) {
		t.Fatalf("offboarded namespace should be deleted, got err=%v", err)
	}
}

func TestFinalizeProjectOnboardingDeletionCompletesWhenNamespaceManuallyRemoved(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name: "team-gone",
			}},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po).Build()
	ctx := context.Background()

	complete, pendingMessage, err := FinalizeProjectOnboardingDeletion(ctx, c, po)
	if err != nil {
		t.Fatalf("finalize deletion: %v", err)
	}
	if !complete {
		t.Fatalf("expected deletion to complete when namespace is absent, pending=%q", pendingMessage)
	}
}

func TestFinalizeProjectOnboardingDeletionBlocksOrphanManagedNamespace(t *testing.T) {
	t.Parallel()

	scheme := newOnboardingTestScheme(t)
	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name: "team-active",
			}},
		},
	}
	orphanLabels := ManagedLabels(po, onboardingv1beta1.NamespaceSpec{Name: "team-removed"})
	orphanNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "team-removed", Labels: orphanLabels},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po, orphanNS).Build()
	ctx := context.Background()

	complete, pendingMessage, err := FinalizeProjectOnboardingDeletion(ctx, c, po)
	if err != nil {
		t.Fatalf("finalize deletion: %v", err)
	}
	if complete {
		t.Fatal("expected deletion to be blocked by orphan managed namespace")
	}
	if !strings.Contains(pendingMessage, "team-removed") {
		t.Fatalf("expected orphan namespace in message, got %q", pendingMessage)
	}
}
