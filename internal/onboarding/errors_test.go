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
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsTransientError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
		{name: "conflict", err: apierrors.NewConflict(schema.GroupResource{Group: "", Resource: "configmaps"}, "x", errors.New("conflict")), want: true},
		{name: "timeout", err: apierrors.NewTimeoutError("timeout", 1), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsTransientError(tt.err); got != tt.want {
				t.Fatalf("IsTransientError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileResultForError(t *testing.T) {
	t.Parallel()

	result, err := ReconcileResultForError(errors.New("permanent"))
	if err == nil {
		t.Fatal("expected permanent error to be returned for metrics")
	}
	if result.RequeueAfter != RequeueAfterFailure {
		t.Fatalf("expected RequeueAfter %v, got %v", RequeueAfterFailure, result.RequeueAfter)
	}

	result, err = ReconcileResultForError(apierrors.NewConflict(schema.GroupResource{Group: "", Resource: "pods"}, "x", errors.New("conflict")))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.RequeueAfter != RequeueAfterTransient {
		t.Fatalf("expected RequeueAfter %v, got %v", RequeueAfterTransient, result.RequeueAfter)
	}

	result, err = ReconcileResultForError(nil)
	if err != nil || result.RequeueAfter != 0 {
		t.Fatalf("nil error should not requeue: result=%+v err=%v", result, err)
	}
}
