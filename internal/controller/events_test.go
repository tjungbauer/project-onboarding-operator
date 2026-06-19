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
	"testing"
	"time"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestProjectOnboardingReconcilerEmitsEventOnSuccess(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = onboardingv1beta1.AddToScheme(scheme)

	po := &onboardingv1beta1.ProjectOnboarding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "tenant",
			Finalizers: []string{onboardingv1beta1.ProjectOnboardingFinalizer},
		},
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			Namespaces: []onboardingv1beta1.NamespaceSpec{{
				Name: "team-a",
				ResourceQuotas: &onboardingv1beta1.ResourceQuotaSpec{
					Enabled: boolPtr(true),
					Pods:    int32Ptr(5),
				},
				DefaultPolicies: minimalDefaultPoliciesForTest(),
			}},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(po).WithStatusSubresource(po).Build()
	broadcaster := record.NewBroadcaster()
	recorder := broadcaster.NewRecorder(scheme, corev1.EventSource{Component: "test"})
	reasons := make(chan string, 1)
	broadcaster.StartEventWatcher(func(e *corev1.Event) {
		select {
		case reasons <- e.Reason:
		default:
		}
	})
	defer broadcaster.Shutdown()

	r := &ProjectOnboardingReconciler{Client: c, Scheme: scheme, Recorder: recorder}
	if _, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "tenant"}}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	select {
	case reason := <-reasons:
		if reason != "ReconcileSucceeded" {
			t.Fatalf("event reason: want ReconcileSucceeded, got %q", reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected ReconcileSucceeded event")
	}
}

func minimalDefaultPoliciesForTest() *onboardingv1beta1.DefaultPoliciesSpec {
	f := false
	t := true
	return &onboardingv1beta1.DefaultPoliciesSpec{
		AllowFromIngress:       &f,
		AllowFromMonitoring:    &f,
		AllowKubeAPIServer:     &f,
		AllowToDNS:             &f,
		AllowFromSameNamespace: &t,
		DenyAllEgress:          &f,
		DenyAllIngress:         &f,
	}
}
