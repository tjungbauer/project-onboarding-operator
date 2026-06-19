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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	RequeueAfterTransient = 30 * time.Second
	RequeueAfterFailure   = 5 * time.Minute
)

// IsTransientError reports API errors that are likely to succeed on retry.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}
	return apierrors.IsConflict(err) ||
		apierrors.IsTimeout(err) ||
		apierrors.IsServerTimeout(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsServiceUnavailable(err) ||
		apierrors.IsInternalError(err)
}

// ReconcileResultForError returns a controller result that avoids tight error-driven
// requeue loops while still retrying transient and persistent failures.
func ReconcileResultForError(err error) (ctrl.Result, error) {
	if err == nil {
		return ctrl.Result{}, nil
	}
	if IsTransientError(err) {
		return ctrl.Result{RequeueAfter: RequeueAfterTransient}, nil
	}
	return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
}
