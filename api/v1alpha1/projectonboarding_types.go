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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ProjectOnboardingFinalizer = "onboarding.stderr.at/finalizer"

	ProjectOnboardingLabelKey     = "onboarding.stderr.at/project-onboarding"
	ProjectOnboardingNamespaceKey = "onboarding.stderr.at/target-namespace"
	ProjectOnboardingManagedByKey = "app.kubernetes.io/managed-by"
	ProjectOnboardingManagedByVal = "project-onboarding-operator"

	PhasePending = "Pending"
	PhaseReady   = "Ready"
	PhaseFailed  = "Failed"

	ConditionReady           = "Ready"
	ConditionDeletionBlocked = "DeletionBlocked"
)

// ProjectOnboardingSpec defines the desired state of ProjectOnboarding.
// The CR is cluster-scoped; spec.namespaces[].name is the tenant Namespace to provision.
// See docs/api-design.md.
type ProjectOnboardingSpec struct {
	// GitOps configures tenant-level Argo CD defaults for AppProject provisioning.
	// +optional
	GitOps *GitOpsSpec `json:"gitOps,omitempty"`

	// Namespaces lists tenant namespaces and their onboarding configuration.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=64
	Namespaces []NamespaceSpec `json:"namespaces"`
}

// NamespaceSpec describes one onboarded namespace and its dependent resources.
// +kubebuilder:validation:XValidation:rule="!has(self.egressIPs) || !has(self.egressIPs.enabled) || !self.egressIPs.enabled || (has(self.egressIPs.ips) && self.egressIPs.ips.size() > 0)",message="egressIPs.ips must contain at least one IP when egressIPs is enabled"
// +kubebuilder:validation:XValidation:rule="!has(self.localAdminGroup) || !has(self.localAdminGroup.enabled) || !self.localAdminGroup.enabled || (has(self.localAdminGroup.users) && self.localAdminGroup.users.size() > 0)",message="localAdminGroup.users must be set when localAdminGroup is enabled"
// +kubebuilder:validation:XValidation:rule="!has(self.projectSize) || self.projectSize.size() == 0 || self.projectSize.matches('^[a-z0-9]([-a-z0-9]*[a-z0-9])?$')",message="projectSize must be a valid DNS-1123 label when set"
type NamespaceSpec struct {
	// Name is the Kubernetes namespace name to create or reconcile.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Name string `json:"name"`

	// Reconciliation Enabled controls whether the operator reconciles this namespace entry.
	// When false, the operator freezes the tenant namespace and managed resources (no updates, no teardown).
	// Use offboard to remove managed resources and delete the tenant namespace.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Offboard removes operator-managed resources for this namespace entry and deletes the tenant
	// namespace (including all workloads still inside it). Defaults to false (opt-in). Independent
	// of reconciliation enabled; when true, cleanup runs even if reconciliation is disabled.
	// Deleting the ProjectOnboarding CR is blocked until every managed tenant namespace is either
	// offboarded (offboard=true) or removed manually from the cluster.
	// +kubebuilder:default=false
	// +optional
	Offboard *bool `json:"offboard,omitempty"`

	// AdditionalSettings configures pod security, OpenShift monitoring, and custom namespace labels.
	// +optional
	AdditionalSettings *AdditionalSettingsSpec `json:"additionalSettings,omitempty"`

	// OverwriteTshirt merges resourceQuotas and limitRanges from this entry onto
	// the referenced TShirtSize when projectSize is set. Required for inline quota/limit
	// fields to override the catalogue. Fields set here win; unset fields use the T-shirt.
	// Ignored when projectSize is empty.
	// +optional
	OverwriteTshirt *bool `json:"overwriteTshirt,omitempty"`

	// ResourceQuotas configures a namespace ResourceQuota.
	// +optional
	ResourceQuotas *ResourceQuotaSpec `json:"resourceQuotas,omitempty"`

	// LimitRanges configures a namespace LimitRange.
	// +optional
	LimitRanges *LimitRangeSpec `json:"limitRanges,omitempty"`

	// DefaultPolicies configures the standard OpenShift network policy set.
	// +optional
	DefaultPolicies *DefaultPoliciesSpec `json:"defaultPolicies,omitempty"`

	// NetworkPolicies defines additional custom NetworkPolicy resources.
	// +optional
	NetworkPolicies []CustomNetworkPolicySpec `json:"networkPolicies,omitempty"`

	// LocalAdminGroup creates an OpenShift Group and RoleBinding for namespace admins.
	// +optional
	LocalAdminGroup *LocalAdminGroupSpec `json:"localAdminGroup,omitempty"`

	// EgressIPs configures an OpenShift OVN EgressIP for the namespace.
	// +optional
	EgressIPs *EgressIPSpec `json:"egressIPs,omitempty"`

	// ApplicationGitOpsNamespace is the namespace of the application-scoped Argo CD instance
	// for this tenant (Helm: application_gitops_namespace). AppProjects for this entry are
	// created there. Each namespace entry can target a different Argo CD instance.
	// +kubebuilder:validation:MaxLength=253
	// +optional
	ApplicationGitOpsNamespace string `json:"applicationGitOpsNamespace,omitempty"`

	// ArgoCDProjects creates Argo CD AppProject resources for GitOps RBAC.
	// +optional
	ArgoCDProjects []ArgoCDProjectSpec `json:"argoCDProjects,omitempty"`

	// ProjectSize references a cluster-scoped TShirtSize by name (metadata.name).
	// Quota and limit range values are taken from that TShirtSize. Inline
	// resourceQuotas/limitRanges on this entry are ignored unless overwriteTshirt is true.
	// Also sets label namespace-size on the tenant Namespace.
	// +kubebuilder:validation:MaxLength=63
	// +optional
	ProjectSize string `json:"projectSize,omitempty"`

	// Labels are additional labels applied to the Namespace resource.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are additional annotations applied to the Namespace resource.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NamespaceLabel is one key/value label applied to the tenant Namespace.
type NamespaceLabel struct {
	// Key is the Kubernetes label key.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`

	// Value is the label value.
	// +optional
	Value string `json:"value,omitempty"`
}

// StorageClassQuota is one StorageClass-related ResourceQuota hard limit key/value pair.
type StorageClassQuota struct {
	// Key is the ResourceQuota hard limit name, e.g. bronze.storageclass.storage.k8s.io/requests.storage.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`

	// Value is the quota value (e.g. 10Gi or 10).
	// +optional
	Value string `json:"value,omitempty"`
}

// AdditionalSettingsSpec configures pod security, monitoring, and custom labels on the namespace.
type AdditionalSettingsSpec struct {
	// PodSecurityEnforce sets the pod-security.kubernetes.io/enforce label (Pod Security Admission).
	// Blocks pods that violate the selected profile from being admitted.
	// privileged: unrestricted; baseline: prevents privileged workloads and host namespaces;
	// restricted: requires non-root, dropped capabilities, and strict volume types.
	// +kubebuilder:validation:Enum=privileged;baseline;restricted
	// +optional
	PodSecurityEnforce string `json:"podSecurityEnforce,omitempty"`

	// PodSecurityWarn sets the pod-security.kubernetes.io/warn label.
	// Admits pods but returns a warning to the user when the profile is violated.
	// Uses the same profile levels as podSecurityEnforce.
	// +kubebuilder:validation:Enum=privileged;baseline;restricted
	// +optional
	PodSecurityWarn string `json:"podSecurityWarn,omitempty"`

	// PodSecurityAudit sets the pod-security.kubernetes.io/audit label.
	// Records profile violations in the Kubernetes audit log without blocking or warning.
	// Uses the same profile levels as podSecurityEnforce.
	// +kubebuilder:validation:Enum=privileged;baseline;restricted
	// +optional
	PodSecurityAudit string `json:"podSecurityAudit,omitempty"`

	// EnableClusterMonitoring sets openshift.io/user-monitoring on the tenant namespace
	// so the user-workload Prometheus stack can scrape workloads in this project.
	// Opt-out: defaults to true; set false to explicitly opt out (label value "false").
	// +kubebuilder:default=true
	// +optional
	EnableClusterMonitoring *bool `json:"enableClusterMonitoring,omitempty"`

	// AdditionalLabels lists custom key/value labels applied to the tenant Namespace.
	// Operator-managed labels (onboarding, pod security, monitoring, project size, etc.) win on conflicts.
	// +optional
	// +listType=map
	// +listMapKey=key
	AdditionalLabels []NamespaceLabel `json:"additionalLabels,omitempty"`
}

// LocalAdminGroupSpec creates an OpenShift Group and binds it to a ClusterRole.
type LocalAdminGroupSpec struct {
	// Enabled controls whether the operator creates the Group and RoleBinding. Opt-in; defaults to false.
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// GroupName is the OpenShift Group name. Defaults to <namespace>-admins when omitted.
	// +optional
	GroupName string `json:"groupName,omitempty"`
	// ClusterRole is the ClusterRole bound in the tenant namespace (e.g. admin, edit).
	// Defaults to admin at reconcile time when omitted.
	// +optional
	ClusterRole string `json:"clusterRole,omitempty"`
	// Users lists OpenShift user names added to the Group. Required when enabled.
	// +optional
	Users []string `json:"users,omitempty"`
}

// EgressIPSpec configures an OpenShift OVN EgressIP object.
type EgressIPSpec struct {
	// Enabled controls whether an EgressIP is created. Defaults to false (opt-in).
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +optional
	IPs []string `json:"ips,omitempty"`
}

// ResourceQuotaSpec mirrors the helper-proj-onboarding quota configuration.
type ResourceQuotaSpec struct {
	// Enabled turns ResourceQuota reconciliation on or off for this namespace entry.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// CPU is the total CPU quota for all pods (e.g. "4" or "4000m").
	// +optional
	CPU *string `json:"cpu,omitempty"`
	// Memory is the total memory quota for all pods (e.g. "4Gi").
	// +optional
	Memory *string `json:"memory,omitempty"`
	// EphemeralStorage is the total ephemeral storage quota for all pods (e.g. "4Gi"). Lower-case suffixes (gi, mi) are normalized automatically.
	// +optional
	EphemeralStorage *string `json:"ephemeralStorage,omitempty"`
	// Pods is the maximum number of pods in the namespace.
	// +optional
	Pods *int32 `json:"pods,omitempty"`
	// ReplicationControllers is the maximum number of ReplicationControllers.
	// +optional
	ReplicationControllers *int32 `json:"replicationControllers,omitempty"`
	// ResourceQuotas is the maximum number of ResourceQuota objects in the namespace.
	// +optional
	ResourceQuotas *int32 `json:"resourceQuotas,omitempty"`
	// Services is the maximum number of Services.
	// +optional
	Services *int32 `json:"services,omitempty"`
	// Secrets is the maximum number of Secrets.
	// +optional
	Secrets *int32 `json:"secrets,omitempty"`
	// ConfigMaps is the maximum number of ConfigMaps.
	// +optional
	ConfigMaps *int32 `json:"configMaps,omitempty"`
	// PersistentVolumeClaims is the maximum number of PersistentVolumeClaims.
	// +optional
	PersistentVolumeClaims *int32 `json:"persistentVolumeClaims,omitempty"`
	// Limits sets cluster-wide limits.* ResourceQuota keys.
	// +optional
	Limits *ResourceQuotaLimitSpec `json:"limits,omitempty"`
	// Requests sets cluster-wide requests.* ResourceQuota keys.
	// +optional
	Requests *ResourceQuotaRequestSpec `json:"requests,omitempty"`
	// StorageClasses lists StorageClass quota keys and values for ResourceQuota hard limits.
	// +optional
	// +listType=map
	// +listMapKey=key
	StorageClasses []StorageClassQuota `json:"storageClasses,omitempty"`
}

type ResourceQuotaLimitSpec struct {
	// CPU is the limits.cpu quota (e.g. "4").
	// +optional
	CPU *string `json:"cpu,omitempty"`
	// Memory is the limits.memory quota (e.g. "4Gi"). Lower-case suffixes are normalized automatically.
	// +optional
	Memory *string `json:"memory,omitempty"`
	// EphemeralStorage is the limits.ephemeral-storage quota (e.g. "4Mi"). Lower-case suffixes are normalized automatically.
	// +optional
	EphemeralStorage *string `json:"ephemeralStorage,omitempty"`
}

type ResourceQuotaRequestSpec struct {
	// CPU is the requests.cpu quota (e.g. "1").
	// +optional
	CPU *string `json:"cpu,omitempty"`
	// Memory is the requests.memory quota (e.g. "2Gi").
	// +optional
	Memory *string `json:"memory,omitempty"`
	// Storage is the requests.storage quota (e.g. "50Gi").
	// +optional
	Storage *string `json:"storage,omitempty"`
	// EphemeralStorage is the requests.ephemeral-storage quota (e.g. "2Gi").
	// +optional
	EphemeralStorage *string `json:"ephemeralStorage,omitempty"`
}

// LimitRangeSpec configures LimitRange defaults and bounds.
type LimitRangeSpec struct {
	// Enabled turns LimitRange reconciliation on or off for this namespace entry.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// Pod sets LimitRange constraints for Pod-type limits (sum of containers in a pod).
	// +optional
	Pod *LimitRangePodSpec `json:"pod,omitempty"`
	// Container sets LimitRange defaults, default requests, and min/max for containers.
	// +optional
	Container *LimitRangeContainerSpec `json:"container,omitempty"`
	// PVC sets min/max storage bounds for PersistentVolumeClaim-type limits.
	// +optional
	PVC *LimitRangePVCSpec `json:"pvc,omitempty"`
}

type LimitRangePodSpec struct {
	// Max sets maximum CPU and memory for a pod (sum of its containers).
	// +optional
	Max *ResourceAmountSpec `json:"max,omitempty"`
	// Min sets minimum CPU and memory for a pod (sum of its containers).
	// +optional
	Min *ResourceAmountSpec `json:"min,omitempty"`
}

type LimitRangeContainerSpec struct {
	// Max sets maximum CPU and memory per container.
	// +optional
	Max *ResourceAmountSpec `json:"max,omitempty"`
	// Min sets minimum CPU and memory per container.
	// +optional
	Min *ResourceAmountSpec `json:"min,omitempty"`
	// Default sets default CPU and memory limits applied to containers without limits.
	// +optional
	Default *ResourceAmountSpec `json:"default,omitempty"`
	// DefaultRequest sets default CPU and memory requests applied to containers without requests.
	// +optional
	DefaultRequest *ResourceAmountSpec `json:"defaultRequest,omitempty"`
}

type LimitRangePVCSpec struct {
	// Min sets minimum storage for PVC-type limits.
	// +optional
	Min *StorageAmountSpec `json:"min,omitempty"`
	// Max sets maximum storage for PVC-type limits.
	// +optional
	Max *StorageAmountSpec `json:"max,omitempty"`
}

type ResourceAmountSpec struct {
	// CPU quantity (e.g. 4, 500m).
	// +optional
	CPU *string `json:"cpu,omitempty"`
	// Memory quantity (e.g. 4Gi, 500Mi).
	// +optional
	Memory *string `json:"memory,omitempty"`
}

type StorageAmountSpec struct {
	// Storage quantity (e.g. 1Gi, 20Gi).
	// +optional
	Storage *string `json:"storage,omitempty"`
}

// DefaultPoliciesSpec toggles the standard network policies installed for a namespace.
// Each allow* policy is opt-in (enabled only when explicitly set to true).
type DefaultPoliciesSpec struct {
	// AllowFromIngress permits ingress from the OpenShift router namespace (public Routes).
	// +optional
	AllowFromIngress *bool `json:"allowFromIngress,omitempty"`
	// AllowFromMonitoring permits ingress from the openshift-monitoring namespace (platform Prometheus).
	// +optional
	AllowFromMonitoring *bool `json:"allowFromMonitoring,omitempty"`
	// AllowKubeAPIServer permits ingress from the kube-apiserver operator (control plane health/metrics).
	// +optional
	AllowKubeAPIServer *bool `json:"allowKubeAPIServer,omitempty"`
	// AllowToDNS permits egress to openshift-dns on ports 53 and 5353 (TCP/UDP).
	// +optional
	AllowToDNS *bool `json:"allowToDNS,omitempty"`
	// DenyAllEgress adds a deny-all egress policy (use together with explicit egress allows).
	// +optional
	DenyAllEgress *bool `json:"denyAllEgress,omitempty"`
	// DenyAllIngress adds a deny-all ingress policy (use together with explicit ingress allows).
	// +optional
	DenyAllIngress *bool `json:"denyAllIngress,omitempty"`
	// AllowFromSameNamespace permits pod-to-pod ingress within the tenant namespace.
	// +optional
	AllowFromSameNamespace *bool `json:"allowFromSameNamespace,omitempty"`
}

// CustomNetworkPolicySpec defines one additional NetworkPolicy.
type CustomNetworkPolicySpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
	// +optional
	IngressRules []NetworkPolicyRuleSpec `json:"ingressRules,omitempty"`
	// +optional
	EgressRules []NetworkPolicyRuleSpec `json:"egressRules,omitempty"`
}

// NetworkPolicyRuleSpec defines ingress or egress peers and ports.
type NetworkPolicyRuleSpec struct {
	// +optional
	Selectors []NetworkPolicyPeerSpec `json:"selectors,omitempty"`
	// +optional
	Ports []NetworkPolicyPortSpec `json:"ports,omitempty"`
}

// NetworkPolicyPeerSpec is a simplified NetworkPolicy peer definition.
type NetworkPolicyPeerSpec struct {
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	// +optional
	IPBlock *IPBlockSpec `json:"ipBlock,omitempty"`
}

type IPBlockSpec struct {
	// +kubebuilder:validation:Required
	CIDR string `json:"cidr"`
	// +optional
	Except []string `json:"except,omitempty"`
}

type NetworkPolicyPortSpec struct {
	// +optional
	Protocol *string `json:"protocol,omitempty"`
	// +optional
	Port *int32 `json:"port,omitempty"`
}

// ProjectOnboardingStatus defines the observed state of ProjectOnboarding.
type ProjectOnboardingStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	Phase string `json:"phase,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	Namespaces []NamespaceStatus `json:"namespaces,omitempty"`
}

// NamespaceStatus reports reconciliation state for one target namespace.
type NamespaceStatus struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	Ready bool `json:"ready,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=pob
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ProjectOnboarding is a cluster-scoped schema for project onboarding API.
type ProjectOnboarding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectOnboardingSpec   `json:"spec,omitempty"`
	Status ProjectOnboardingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectOnboardingList contains a list of ProjectOnboarding.
type ProjectOnboardingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectOnboarding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectOnboarding{}, &ProjectOnboardingList{})
}
