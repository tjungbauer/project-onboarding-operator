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
	"strings"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var appProjectGVK = schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "AppProject"}

func reconcileArgoCDProjects(ctx context.Context, c client.Client, scheme *runtime.Scheme, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) error {
	if len(nsSpec.ArgoCDProjects) == 0 {
		return nil
	}

	argoNS := ResolveApplicationGitOpsNamespace(po, nsSpec)
	if argoNS == "" {
		return fmt.Errorf("argoCDProjects on namespace %q require applicationGitOpsNamespace (Helm application_gitops_namespace)", nsSpec.Name)
	}

	for _, projectSpec := range nsSpec.ArgoCDProjects {
		if !IsOptInEnabled(projectSpec.Enabled) {
			continue
		}
		if err := reconcileAppProject(ctx, c, scheme, po, projectSpec, nsName, argoNS, labels); err != nil {
			return err
		}
	}
	return nil
}

func reconcileAppProject(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	po *onboardingv1beta1.ProjectOnboarding,
	projectSpec onboardingv1beta1.ArgoCDProjectSpec,
	tenantNS string,
	argoNS string,
	labels map[string]string,
) error {
	projectName := SanitizeName(projectSpec.Name)
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(appProjectGVK)

	key := client.ObjectKey{Namespace: argoNS, Name: projectName}
	err := c.Get(ctx, key, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	projectLabels := map[string]string{
		onboardingv1beta1.ProjectOnboardingManagedByKey: onboardingv1beta1.ProjectOnboardingManagedByVal,
		onboardingv1beta1.ProjectOnboardingLabelKey:     po.Name,
		onboardingv1beta1.ProjectOnboardingNamespaceKey: tenantNS,
		onboardingv1beta1.ArgoCDProjectInheritLabel:     "global",
	}
	for k, v := range labels {
		projectLabels[k] = v
	}

	description := projectSpec.Description
	if description == "" {
		description = projectName + " GitOps Project"
	}

	spec := map[string]interface{}{
		"description":              description,
		"clusterResourceWhitelist": []interface{}{},
		"roles":                    buildArgoCDRoles(po, projectSpec, tenantNS),
		"sourceNamespaces":         stringList(resolveSourceNamespaces(po, projectSpec)),
		"sourceRepos":              stringList(resolveSourceRepos(po, projectSpec)),
		"destinations":             buildDestinations(po, projectSpec, tenantNS),
	}

	if apierrors.IsNotFound(err) {
		obj = &unstructured.Unstructured{}
		obj.SetGroupVersionKind(appProjectGVK)
		obj.SetName(projectName)
		obj.SetNamespace(argoNS)
		obj.SetLabels(projectLabels)
		obj.SetAnnotations(map[string]string{"argocd.argoproj.io/sync-wave": "1"})
		if err := unstructured.SetNestedMap(obj.Object, spec, "spec"); err != nil {
			return err
		}
		if err := controllerutil.SetControllerReference(po, obj, scheme); err != nil {
			return err
		}
		return c.Create(ctx, obj)
	}

	patchBase := obj.DeepCopy()
	obj.SetLabels(mergeStringMaps(obj.GetLabels(), projectLabels))
	if err := unstructured.SetNestedMap(obj.Object, spec, "spec"); err != nil {
		return err
	}
	if err := controllerutil.SetControllerReference(po, obj, scheme); err != nil {
		return err
	}
	return c.Patch(ctx, obj, client.MergeFrom(patchBase))
}

func buildDestinations(po *onboardingv1beta1.ProjectOnboarding, projectSpec onboardingv1beta1.ArgoCDProjectSpec, tenantNS string) []interface{} {
	destinations := projectSpec.Destinations
	if len(destinations) == 0 && po.Spec.GitOps != nil {
		destinations = po.Spec.GitOps.Destinations
	}
	out := make([]interface{}, 0, len(destinations))
	for _, dest := range destinations {
		out = append(out, map[string]interface{}{
			"name":      dest.Name,
			"namespace": tenantNS,
			"server":    dest.Server,
		})
	}
	return out
}

func resolveSourceRepos(po *onboardingv1beta1.ProjectOnboarding, projectSpec onboardingv1beta1.ArgoCDProjectSpec) []string {
	if len(projectSpec.SourceRepos) > 0 {
		return projectSpec.SourceRepos
	}
	if po.Spec.GitOps != nil && len(po.Spec.GitOps.AllowedSourceRepos) > 0 {
		return po.Spec.GitOps.AllowedSourceRepos
	}
	return []string{"*"}
}

func resolveSourceNamespaces(po *onboardingv1beta1.ProjectOnboarding, projectSpec onboardingv1beta1.ArgoCDProjectSpec) []string {
	if len(projectSpec.SourceNamespaces) > 0 {
		return projectSpec.SourceNamespaces
	}
	if po.Spec.GitOps != nil && len(po.Spec.GitOps.AllowedSourceNamespaces) > 0 {
		return po.Spec.GitOps.AllowedSourceNamespaces
	}
	return []string{"*"}
}

func resolveRoleGroups(po *onboardingv1beta1.ProjectOnboarding, projectSpec onboardingv1beta1.ArgoCDProjectSpec, role onboardingv1beta1.ArgoCDRoleSpec) []string {
	if len(role.OIDCGroups) > 0 {
		return role.OIDCGroups
	}
	if len(projectSpec.OIDCGroups) > 0 {
		return projectSpec.OIDCGroups
	}
	if po.Spec.GitOps != nil && len(po.Spec.GitOps.AllowedOIDCGroups) > 0 {
		return po.Spec.GitOps.AllowedOIDCGroups
	}
	return []string{"dummy-group"}
}

func buildArgoCDRoles(po *onboardingv1beta1.ProjectOnboarding, projectSpec onboardingv1beta1.ArgoCDProjectSpec, tenantNS string) []interface{} {
	projectName := SanitizeName(projectSpec.Name)
	tenantName := SanitizeName(tenantNS)
	roles := make([]interface{}, 0, len(projectSpec.Roles))
	for _, role := range projectSpec.Roles {
		roles = append(roles, map[string]interface{}{
			"name":        role.Name,
			"description": role.Description,
			"groups":      stringList(resolveRoleGroups(po, projectSpec, role)),
			"policies":    stringList(buildRolePolicies(projectName, tenantName, role)),
		})
	}
	return roles
}

func buildRolePolicies(projectName, tenantName string, role onboardingv1beta1.ArgoCDRoleSpec) []string {
	if IsOptInEnabled(role.FullAccess) {
		prefix := tenantName
		return []string{
			fmt.Sprintf("p, proj:%s:%s, applications, get, %s/*, allow", prefix, role.Name, projectName),
			fmt.Sprintf("p, proj:%s:%s, applications, create, %s/*, allow", prefix, role.Name, projectName),
			fmt.Sprintf("p, proj:%s:%s, applications, update, %s/*, allow", prefix, role.Name, projectName),
			fmt.Sprintf("p, proj:%s:%s, applications, delete, %s/*, allow", prefix, role.Name, projectName),
			fmt.Sprintf("p, proj:%s:%s, applications, sync, %s/*, allow", prefix, role.Name, projectName),
			fmt.Sprintf("p, proj:%s:%s, applications, override, %s/*, allow", prefix, role.Name, projectName),
			fmt.Sprintf("p, proj:%s:%s, repositories, create, *, allow", prefix, role.Name),
			fmt.Sprintf("p, proj:%s:%s, repositories, get, *, allow", prefix, role.Name),
			fmt.Sprintf("p, proj:%s:%s, repositories, update, *, allow", prefix, role.Name),
			fmt.Sprintf("p, proj:%s:%s, repositories, delete, *, allow", prefix, role.Name),
		}
	}
	policies := make([]string, 0, len(role.Policies))
	for _, policy := range role.Policies {
		resource := policy.Resource
		if resource == "" {
			resource = "applications"
		}
		object := policy.Object
		if object == "" {
			object = "*"
		}
		permission := policy.Permission
		if permission == "" {
			permission = "allow"
		}
		prefix := casbinProjectPrefix(tenantName, policy)
		policies = append(policies, fmt.Sprintf(
			"p, proj:%s:%s, %s, %s, %s/%s, %s",
			prefix, role.Name, resource, policy.Action, projectName, object, permission,
		))
	}
	return policies
}

func casbinProjectPrefix(tenantName string, policy onboardingv1beta1.ArgoCDPolicySpec) string {
	if name := strings.TrimSpace(policy.AppProjectName); name != "" {
		return SanitizeName(name)
	}
	return tenantName
}

func stringList(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func gitOpsManagedByLabel(po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) string {
	return ResolveApplicationGitOpsNamespace(po, nsSpec)
}

// ResolveApplicationGitOpsNamespace returns the Argo CD control-plane namespace for one tenant entry.
// Precedence: namespace entry, legacy spec.gitOps.applicationNamespace, cluster defaults (already merged into po).
func ResolveApplicationGitOpsNamespace(po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) string {
	if ns := strings.TrimSpace(nsSpec.ApplicationGitOpsNamespace); ns != "" {
		return ns
	}
	if po.Spec.GitOps != nil {
		return strings.TrimSpace(po.Spec.GitOps.ApplicationNamespace)
	}
	return ""
}

func deleteManagedAppProjects(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec) error {
	return deleteAppProjectsForNamespace(ctx, c, po, SanitizeName(nsSpec.Name), nil)
}

func pruneAppProjectsForNamespace(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsSpec onboardingv1beta1.NamespaceSpec, nsName string) error {
	keep := make(map[string]struct{})
	for _, projectSpec := range nsSpec.ArgoCDProjects {
		if IsOptInEnabled(projectSpec.Enabled) {
			keep[SanitizeName(projectSpec.Name)] = struct{}{}
		}
	}
	return deleteAppProjectsForNamespace(ctx, c, po, nsName, keep)
}

func deleteAppProjectsForNamespace(ctx context.Context, c client.Client, po *onboardingv1beta1.ProjectOnboarding, nsName string, keep map[string]struct{}) error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Group: appProjectGVK.Group, Version: appProjectGVK.Version, Kind: appProjectGVK.Kind + "List"})
	if err := c.List(ctx, list, client.MatchingLabels{
		onboardingv1beta1.ProjectOnboardingLabelKey:     po.Name,
		onboardingv1beta1.ProjectOnboardingNamespaceKey: nsName,
	}); err != nil {
		if meta.IsNoMatchError(err) {
			return nil
		}
		return err
	}
	for i := range list.Items {
		if keep != nil {
			if _, ok := keep[list.Items[i].GetName()]; ok {
				continue
			}
		}
		if err := c.Delete(ctx, &list.Items[i]); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
