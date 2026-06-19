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
	"fmt"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileNamespace ensures all onboarding resources exist for one namespace entry.
func reconcileLocalAdminGroup(ctx context.Context, c client.Client, scheme *runtime.Scheme, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec, labels map[string]string) error {
	groupName := GroupName(nsSpec)
	group := &unstructured.Unstructured{}
	group.SetGroupVersionKind(schema.GroupVersionKind{Group: "user.openshift.io", Version: "v1", Kind: "Group"})

	err := c.Get(ctx, client.ObjectKey{Name: groupName}, group)
	if err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		return err
	}
	if meta.IsNoMatchError(err) {
		return fmt.Errorf("OpenShift Group API not available; localAdminGroup requires OpenShift")
	}

	users := make([]interface{}, 0, len(nsSpec.LocalAdminGroup.Users))
	for _, user := range nsSpec.LocalAdminGroup.Users {
		users = append(users, user)
	}

	if apierrors.IsNotFound(err) {
		group.SetName(groupName)
		group.SetLabels(labels)
		if err := unstructured.SetNestedStringSlice(group.Object, convertUsers(users), "users"); err != nil {
			return err
		}
		if err := ensureProjectOnboardingControllerRef(scheme, po, group); err != nil {
			return err
		}
		return c.Create(ctx, group)
	}

	patch := client.MergeFrom(group.DeepCopy())
	if group.GetLabels() == nil {
		group.SetLabels(map[string]string{})
	}
	for k, v := range labels {
		group.GetLabels()[k] = v
	}
	if err := unstructured.SetNestedStringSlice(group.Object, convertUsers(users), "users"); err != nil {
		return err
	}
	if err := ensureProjectOnboardingControllerRef(scheme, po, group); err != nil {
		return err
	}
	return c.Patch(ctx, group, patch)
}

func convertUsers(users []interface{}) []string {
	out := make([]string, 0, len(users))
	for _, user := range users {
		out = append(out, fmt.Sprint(user))
	}
	return out
}

func reconcileLocalAdminRoleBinding(ctx context.Context, c client.Client, scheme *runtime.Scheme, tenantNS *corev1.Namespace, nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) error {
	rbName := RoleBindingName(nsSpec)
	desired := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbName,
			Namespace: nsName,
			Labels:    labels,
		},
		Subjects: []rbacv1.Subject{{
			Kind:     rbacv1.GroupKind,
			APIGroup: rbacv1.GroupName,
			Name:     GroupName(nsSpec),
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     ClusterRoleName(nsSpec),
		},
	}

	current := &rbacv1.RoleBinding{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nsName, Name: rbName}, current)
	if apierrors.IsNotFound(err) {
		if err := ensureTenantNamespaceOwnerRef(scheme, tenantNS, desired); err != nil {
			return err
		}
		return c.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	patch := client.MergeFrom(current.DeepCopy())
	current.Labels = mergeStringMaps(current.Labels, labels)
	current.Subjects = desired.Subjects
	current.RoleRef = desired.RoleRef
	if err := ensureTenantNamespaceOwnerRef(scheme, tenantNS, current); err != nil {
		return err
	}
	return c.Patch(ctx, current, patch)
}

func reconcileEgressIP(ctx context.Context, c client.Client, scheme *runtime.Scheme, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) error {
	if len(nsSpec.EgressIPs.IPs) == 0 {
		return fmt.Errorf("egressIPs.enabled requires at least one IP")
	}

	egress := &unstructured.Unstructured{}
	egress.SetGroupVersionKind(schema.GroupVersionKind{Group: "k8s.ovn.org", Version: "v1", Kind: "EgressIP"})

	err := c.Get(ctx, client.ObjectKey{Name: nsName}, egress)
	if err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		return err
	}
	if meta.IsNoMatchError(err) {
		return fmt.Errorf("OpenShift EgressIP API not available; egressIPs requires OVN-Kubernetes on OpenShift")
	}

	egressIPs := make([]interface{}, 0, len(nsSpec.EgressIPs.IPs))
	for _, ip := range nsSpec.EgressIPs.IPs {
		egressIPs = append(egressIPs, ip)
	}

	namespaceSelector := map[string]interface{}{
		"matchLabels": map[string]interface{}{
			"env": nsName,
		},
	}

	if apierrors.IsNotFound(err) {
		egress.SetName(nsName)
		egress.SetLabels(labels)
		if err := unstructured.SetNestedSlice(egress.Object, egressIPs, "spec", "egressIPs"); err != nil {
			return err
		}
		if err := unstructured.SetNestedMap(egress.Object, namespaceSelector, "spec", "namespaceSelector"); err != nil {
			return err
		}
		if err := ensureProjectOnboardingControllerRef(scheme, po, egress); err != nil {
			return err
		}
		return c.Create(ctx, egress)
	}

	patch := client.MergeFrom(egress.DeepCopy())
	if egress.GetLabels() == nil {
		egress.SetLabels(map[string]string{})
	}
	for k, v := range labels {
		egress.GetLabels()[k] = v
	}
	if err := unstructured.SetNestedSlice(egress.Object, egressIPs, "spec", "egressIPs"); err != nil {
		return err
	}
	if err := unstructured.SetNestedMap(egress.Object, namespaceSelector, "spec", "namespaceSelector"); err != nil {
		return err
	}
	if err := ensureProjectOnboardingControllerRef(scheme, po, egress); err != nil {
		return err
	}
	return c.Patch(ctx, egress, patch)
}

func mergeStringMaps(base, overlay map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}
