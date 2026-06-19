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

package v1alpha1

const (
	// ArgoCDManagedByLabel is applied to tenant namespaces when GitOps defaults are configured.
	ArgoCDManagedByLabel = "argocd.argoproj.io/managed-by"
	// ArgoCDProjectInheritLabel mirrors helper-proj-onboarding AppProject metadata.
	ArgoCDProjectInheritLabel = "argocd.argoproj.io/project-inherit"
)

// GitOpsSpec holds tenant-level Argo CD defaults (maps to helper-proj-onboarding top-level values).
type GitOpsSpec struct {
	// ApplicationNamespace is a legacy CR-level fallback when applicationGitOpsNamespace
	// is unset on a namespace entry. Prefer spec.namespaces[].applicationGitOpsNamespace.
	// +kubebuilder:validation:MaxLength=253
	// +optional
	ApplicationNamespace string `json:"applicationNamespace,omitempty"`

	// Destinations lists allowed Argo CD destination clusters (maps to global.envs + allowed_envs).
	// +optional
	Destinations []GitOpsDestinationSpec `json:"destinations,omitempty"`

	// AllowedSourceRepos restricts Application source repositories when not overridden per project.
	// +optional
	AllowedSourceRepos []string `json:"allowedSourceRepos,omitempty"`

	// AllowedOIDCGroups are default OIDC groups for AppProject RBAC roles.
	// +optional
	AllowedOIDCGroups []string `json:"allowedOIDCGroups,omitempty"`

	// AllowedSourceNamespaces restricts where Applications may be defined from.
	// +optional
	AllowedSourceNamespaces []string `json:"allowedSourceNamespaces,omitempty"`
}

// GitOpsDestinationSpec names a destination cluster for AppProject.spec.destinations.
type GitOpsDestinationSpec struct {
	// Name is the Argo CD cluster secret name (e.g. in-cluster).
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Server is the Kubernetes API URL (e.g. https://kubernetes.default.svc).
	// +kubebuilder:validation:Required
	Server string `json:"server"`
}

// ArgoCDProjectSpec configures one Argo CD AppProject for a tenant namespace.
type ArgoCDProjectSpec struct {
	// Enabled controls whether the operator creates this AppProject. Opt-in; defaults to false.
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Name is the AppProject metadata.name in the Argo CD namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Name string `json:"name"`

	// Description is shown in Argo CD UI.
	// +optional
	Description string `json:"description,omitempty"`

	// SourceRepos overrides GitOpsSpec.allowedSourceRepos for this project.
	// +optional
	SourceRepos []string `json:"sourceRepos,omitempty"`

	// SourceNamespaces overrides GitOpsSpec.allowedSourceNamespaces for this project.
	// +optional
	SourceNamespaces []string `json:"sourceNamespaces,omitempty"`

	// Destinations overrides GitOpsSpec.destinations for this project.
	// +optional
	Destinations []GitOpsDestinationSpec `json:"destinations,omitempty"`

	// OIDCGroups overrides GitOpsSpec.allowedOIDCGroups for all roles in this project.
	// +optional
	OIDCGroups []string `json:"oidcGroups,omitempty"`

	// Roles defines AppProject.spec.roles (RBAC policies).
	// +optional
	Roles []ArgoCDRoleSpec `json:"roles,omitempty"`
}

// ArgoCDRoleSpec maps to one AppProject.spec.roles entry.
type ArgoCDRoleSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +optional
	Description string `json:"description,omitempty"`

	// OIDCGroups overrides project-level OIDC groups for this role.
	// +optional
	OIDCGroups []string `json:"oidcGroups,omitempty"`

	// FullAccess grants all application and repository actions on the project.
	// +optional
	FullAccess *bool `json:"fullAccess,omitempty"`

	// Policies are granular Casbin policies when fullAccess is false.
	// +optional
	Policies []ArgoCDPolicySpec `json:"policies,omitempty"`
}

// ArgoCDPolicySpec is one Casbin policy line component.
type ArgoCDPolicySpec struct {
	// Resource is usually "applications" or "repositories".
	// +optional
	Resource string `json:"resource,omitempty"`

	// Action is e.g. get, create, sync, delete.
	// +kubebuilder:validation:Required
	Action string `json:"action"`

	// Object is the application path within the project (default "*").
	// +optional
	Object string `json:"object,omitempty"`

	// AppProjectName overrides the Casbin proj: prefix for this policy. When unset, the tenant namespace name is used.
	// +optional
	AppProjectName string `json:"appProjectName,omitempty"`

	// Permission is allow or deny.
	// +optional
	Permission string `json:"permission,omitempty"`
}
