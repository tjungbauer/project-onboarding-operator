#!/usr/bin/env python3
"""Inject ProjectOnboarding OLM specDescriptors after operator-sdk generate bundle."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

POD_SECURITY_SELECT = """        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:select:privileged
        - urn:alm:descriptor:com.tectonic.ui:select:baseline
        - urn:alm:descriptor:com.tectonic.ui:select:restricted"""

POLICY_RESOURCE_SELECT = """        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:select:applications
        - urn:alm:descriptor:com.tectonic.ui:select:repositories"""

POLICY_ACTION_SELECT = """        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:select:get
        - urn:alm:descriptor:com.tectonic.ui:select:create
        - urn:alm:descriptor:com.tectonic.ui:select:update
        - urn:alm:descriptor:com.tectonic.ui:select:delete
        - urn:alm:descriptor:com.tectonic.ui:select:sync
        - urn:alm:descriptor:com.tectonic.ui:select:override
        - urn:alm:descriptor:com.tectonic.ui:select:action"""

POLICY_PERMISSION_SELECT = """        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:select:allow
        - urn:alm:descriptor:com.tectonic.ui:select:deny"""

CPU_QTY_DESC = (
    "Kubernetes CPU quantity — whole cores (4) or millicores (500m). "
    "Plain integers mean CPU cores, not gibibytes."
)
BYTE_QTY_DESC = (
    "Kubernetes memory or storage quantity with a required unit suffix (e.g. 4Gi, 500Mi, 50Gi). "
    "Plain numbers such as 4 are rejected by admission (Kubernetes would treat them as bytes). "
    "Lower-case gi/mi are normalized to Gi/Mi."
)
STORAGE_CLASS_VALUE_DESC = (
    "Quota value for the StorageClass key — use Gi/Mi for .../requests.storage keys (e.g. 10Gi), "
    "or a plain integer for count keys such as .../persistentvolumeclaims (e.g. 10)."
)

# Order matters for OpenShift console form layout (listed before undescribed fields).
RESOURCE_QUOTA_DESCRIPTORS = f"""      - description: Resource quota limits for the tenant namespace. Enable reconciliation, then set counts and quantities as needed.
        displayName: Resource Quotas
        path: namespaces[0].resourceQuotas
      - description: When enabled, the operator creates or updates a ResourceQuota in the tenant namespace.
        displayName: Enabled
        path: namespaces[0].resourceQuotas.enabled
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: {CPU_QTY_DESC}
        displayName: CPU
        path: namespaces[0].resourceQuotas.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Memory
        path: namespaces[0].resourceQuotas.memory
      - description: {BYTE_QTY_DESC}
        displayName: Ephemeral Storage
        path: namespaces[0].resourceQuotas.ephemeralStorage
      - description: Maximum number of pods in the namespace.
        displayName: Pods
        path: namespaces[0].resourceQuotas.pods
      - description: Maximum number of ReplicationControllers.
        displayName: Replication Controllers
        path: namespaces[0].resourceQuotas.replicationControllers
      - description: Maximum number of ResourceQuota objects in the namespace.
        displayName: Resource Quotas Count
        path: namespaces[0].resourceQuotas.resourceQuotas
      - description: Maximum number of Services.
        displayName: Services
        path: namespaces[0].resourceQuotas.services
      - description: Maximum number of Secrets.
        displayName: Secrets
        path: namespaces[0].resourceQuotas.secrets
      - description: Maximum number of ConfigMaps.
        displayName: ConfigMaps
        path: namespaces[0].resourceQuotas.configMaps
      - description: Maximum number of PersistentVolumeClaims.
        displayName: Persistent Volume Claims
        path: namespaces[0].resourceQuotas.persistentVolumeClaims
      - description: Cluster-wide limits on CPU, memory, and ephemeral storage for all containers in the namespace.
        displayName: Limits
        path: namespaces[0].resourceQuotas.limits
      - description: {CPU_QTY_DESC}
        displayName: Limits CPU
        path: namespaces[0].resourceQuotas.limits.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Limits Memory
        path: namespaces[0].resourceQuotas.limits.memory
      - description: {BYTE_QTY_DESC}
        displayName: Limits Ephemeral Storage
        path: namespaces[0].resourceQuotas.limits.ephemeralStorage
      - description: Cluster-wide requests on CPU, memory, storage, and ephemeral storage for all containers in the namespace.
        displayName: Requests
        path: namespaces[0].resourceQuotas.requests
      - description: {CPU_QTY_DESC}
        displayName: Requests CPU
        path: namespaces[0].resourceQuotas.requests.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Requests Memory
        path: namespaces[0].resourceQuotas.requests.memory
      - description: {BYTE_QTY_DESC}
        displayName: Requests Storage
        path: namespaces[0].resourceQuotas.requests.storage
      - description: {BYTE_QTY_DESC}
        displayName: Requests Ephemeral Storage
        path: namespaces[0].resourceQuotas.requests.ephemeralStorage
      - description: Per-storage-class quota keys mapped to ResourceQuota hard limits. Use Add entry for each StorageClass key such as bronze.storageclass.storage.k8s.io/requests.storage with value 10Gi.
        displayName: Storage Classes
        path: namespaces[0].resourceQuotas.storageClasses
      - description: ResourceQuota hard limit key (StorageClass annotation format).
        displayName: Key
        path: namespaces[0].resourceQuotas.storageClasses[0].key
      - description: {STORAGE_CLASS_VALUE_DESC}
        displayName: Value
        path: namespaces[0].resourceQuotas.storageClasses[0].value
"""

LIMIT_RANGE_DESCRIPTORS = f"""      - description: LimitRange defaults and min/max bounds for pods, containers, and PVCs in the tenant namespace.
        displayName: Limit Ranges
        path: namespaces[0].limitRanges
      - description: When enabled, the operator creates or updates a LimitRange in the tenant namespace.
        displayName: Enabled
        path: namespaces[0].limitRanges.enabled
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Pod-type limits apply to the sum of resources across all containers in a pod.
        displayName: Pod
        path: namespaces[0].limitRanges.pod
      - description: Maximum CPU and memory for a pod (sum of its containers).
        displayName: Pod Max
        path: namespaces[0].limitRanges.pod.max
      - description: {CPU_QTY_DESC}
        displayName: Pod Max CPU
        path: namespaces[0].limitRanges.pod.max.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Pod Max Memory
        path: namespaces[0].limitRanges.pod.max.memory
      - description: Minimum CPU and memory for a pod (sum of its containers).
        displayName: Pod Min
        path: namespaces[0].limitRanges.pod.min
      - description: {CPU_QTY_DESC}
        displayName: Pod Min CPU
        path: namespaces[0].limitRanges.pod.min.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Pod Min Memory
        path: namespaces[0].limitRanges.pod.min.memory
      - description: Container-type limits apply per container in a pod.
        displayName: Container
        path: namespaces[0].limitRanges.container
      - description: Maximum CPU and memory per container.
        displayName: Container Max
        path: namespaces[0].limitRanges.container.max
      - description: {CPU_QTY_DESC}
        displayName: Container Max CPU
        path: namespaces[0].limitRanges.container.max.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Container Max Memory
        path: namespaces[0].limitRanges.container.max.memory
      - description: Minimum CPU and memory per container.
        displayName: Container Min
        path: namespaces[0].limitRanges.container.min
      - description: {CPU_QTY_DESC}
        displayName: Container Min CPU
        path: namespaces[0].limitRanges.container.min.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Container Min Memory
        path: namespaces[0].limitRanges.container.min.memory
      - description: Default limits applied to containers that specify none.
        displayName: Container Default
        path: namespaces[0].limitRanges.container.default
      - description: {CPU_QTY_DESC}
        displayName: Container Default CPU
        path: namespaces[0].limitRanges.container.default.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Container Default Memory
        path: namespaces[0].limitRanges.container.default.memory
      - description: Default requests applied to containers that specify none.
        displayName: Container Default Request
        path: namespaces[0].limitRanges.container.defaultRequest
      - description: {CPU_QTY_DESC}
        displayName: Container Default Request CPU
        path: namespaces[0].limitRanges.container.defaultRequest.cpu
      - description: {BYTE_QTY_DESC}
        displayName: Container Default Request Memory
        path: namespaces[0].limitRanges.container.defaultRequest.memory
      - description: PVC-type limits apply to PersistentVolumeClaim storage requests.
        displayName: Persistent Volume Claims
        path: namespaces[0].limitRanges.pvc
      - description: Minimum storage per PVC.
        displayName: Persistent Volume Claims Min
        path: namespaces[0].limitRanges.pvc.min
      - description: {BYTE_QTY_DESC}
        displayName: Persistent Volume Claims Min Storage
        path: namespaces[0].limitRanges.pvc.min.storage
      - description: Maximum storage per PVC.
        displayName: Persistent Volume Claims Max
        path: namespaces[0].limitRanges.pvc.max
      - description: {BYTE_QTY_DESC}
        displayName: Persistent Volume Claims Max Storage
        path: namespaces[0].limitRanges.pvc.max.storage
"""

DEFAULT_POLICIES_DESCRIPTORS = """      - description: Built-in NetworkPolicy set for tenant namespaces. Each toggle is opt-in; leave all off for no default policies.
        displayName: Default Network Policies
        path: namespaces[0].defaultPolicies
      - description: Allow ingress from the OpenShift router (Routes). Creates allow-from-openshift-ingress.
        displayName: Allow From Ingress
        path: namespaces[0].defaultPolicies.allowFromIngress
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Allow ingress from openshift-monitoring (platform Prometheus scrapes).
        displayName: Allow From Monitoring
        path: namespaces[0].defaultPolicies.allowFromMonitoring
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Allow ingress from the kube-apiserver operator namespace (control plane probes).
        displayName: Allow Kube API Server
        path: namespaces[0].defaultPolicies.allowKubeAPIServer
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Allow egress to openshift-dns on ports 53 and 5353 (TCP/UDP).
        displayName: Allow To DNS
        path: namespaces[0].defaultPolicies.allowToDNS
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Allow pod-to-pod ingress within the tenant namespace.
        displayName: Allow From Same Namespace
        path: namespaces[0].defaultPolicies.allowFromSameNamespace
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Deny all egress (combine with explicit egress allows such as DNS).
        displayName: Deny All Egress
        path: namespaces[0].defaultPolicies.denyAllEgress
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Deny all ingress (combine with explicit ingress allows such as same-namespace or router).
        displayName: Deny All Ingress
        path: namespaces[0].defaultPolicies.denyAllIngress
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
"""

GITOPS_HIDDEN_DESCRIPTOR = """      - description: Cluster GitOps defaults (repos, OIDC groups, destinations). Configure via onboarding-defaults ConfigMap or YAML only — hidden in the form.
        displayName: GitOps
        path: gitOps
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:hidden
"""

ARGO_CD_PROJECT_DESCRIPTORS = f"""      - description: Namespace of the application-scoped Argo CD instance for this tenant entry (Helm application_gitops_namespace). AppProjects are created in this namespace. Each namespace entry can use a different instance.
        displayName: Application GitOps Namespace
        path: namespaces[0].applicationGitOpsNamespace
      - description: Argo CD AppProject for this tenant namespace. Set Application GitOps Namespace above before enabling a project.
        displayName: Argo CD Project
        path: namespaces[0].argoCDProjects
      - description: Opt-in. When enabled, the operator creates or updates the AppProject in the Argo CD namespace.
        displayName: Enabled
        path: namespaces[0].argoCDProjects[0].enabled
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: AppProject metadata.name in the Argo CD namespace (DNS-1123 label). Required when enabled; no default is pre-filled in the form.
        displayName: Project Name
        path: namespaces[0].argoCDProjects[0].name
      - description: Description shown in the Argo CD UI.
        displayName: Description
        path: namespaces[0].argoCDProjects[0].description
      - description: Allowed Git repository URLs for this AppProject (overrides cluster GitOps defaults when set).
        displayName: Source Repos
        path: namespaces[0].argoCDProjects[0].sourceRepos
      - description: Namespaces Applications may be defined from (overrides cluster GitOps defaults when set).
        displayName: Source Namespaces
        path: namespaces[0].argoCDProjects[0].sourceNamespaces
      - description: Allowed destination clusters for this AppProject (overrides cluster GitOps defaults when set).
        displayName: Destinations
        path: namespaces[0].argoCDProjects[0].destinations
      - description: Argo CD cluster secret name for this destination (e.g. in-cluster).
        displayName: Destination Name
        path: namespaces[0].argoCDProjects[0].destinations[0].name
      - description: Kubernetes API server URL for this destination.
        displayName: Destination Server
        path: namespaces[0].argoCDProjects[0].destinations[0].server
      - description: OIDC groups for all roles in this AppProject (overrides cluster GitOps defaults when set).
        displayName: OIDC Groups
        path: namespaces[0].argoCDProjects[0].oidcGroups
      - description: AppProject RBAC roles (Casbin policies).
        displayName: Roles
        path: namespaces[0].argoCDProjects[0].roles
      - description: Role name in AppProject.spec.roles.
        displayName: Role Name
        path: namespaces[0].argoCDProjects[0].roles[0].name
      - description: Role description shown in Argo CD.
        displayName: Role Description
        path: namespaces[0].argoCDProjects[0].roles[0].description
      - description: OIDC groups for this role (overrides project-level OIDC groups).
        displayName: Role OIDC Groups
        path: namespaces[0].argoCDProjects[0].roles[0].oidcGroups
      - description: Grant all application and repository actions on this project.
        displayName: Full Access
        path: namespaces[0].argoCDProjects[0].roles[0].fullAccess
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Granular Casbin policies when full access is disabled.
        displayName: Policies
        path: namespaces[0].argoCDProjects[0].roles[0].policies
      - description: Argo CD RBAC resource (applications or repositories). Required when adding a policy; no default in the form.
        displayName: Policy Resource
        path: namespaces[0].argoCDProjects[0].roles[0].policies[0].resource
{POLICY_RESOURCE_SELECT}
      - description: Argo CD RBAC action (e.g. get, sync, create). Required when adding a policy.
        displayName: Policy Action
        path: namespaces[0].argoCDProjects[0].roles[0].policies[0].action
{POLICY_ACTION_SELECT}
      - description: Application path within the project (use * for all).
        displayName: Policy Object
        path: namespaces[0].argoCDProjects[0].roles[0].policies[0].object
      - description: allow or deny. Required when adding a policy; no default in the form.
        displayName: Policy Permission
        path: namespaces[0].argoCDProjects[0].roles[0].policies[0].permission
{POLICY_PERMISSION_SELECT}
"""

NAMESPACE_EXTENDED_DESCRIPTORS = """      - description: Custom NetworkPolicy resources reconciled in the tenant namespace.
        displayName: Network Policies
        path: namespaces[0].networkPolicies
"""

LOCAL_ADMIN_GROUP_DESCRIPTORS = """      - description: Creates an OpenShift Group and namespace RoleBinding for tenant administrators.
        displayName: Namespace Admins
        path: namespaces[0].localAdminGroup
      - description: Opt-in. When enabled, the operator creates the Group, adds users, and binds the Cluster Role in the tenant namespace.
        displayName: Enabled
        path: namespaces[0].localAdminGroup.enabled
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: ClusterRole bound in the tenant namespace (e.g. admin or edit). Defaults to admin at reconcile time when omitted.
        displayName: Cluster Role
        path: namespaces[0].localAdminGroup.clusterRole
      - description: OpenShift Group name. Defaults to <namespace>-admins when omitted.
        displayName: Group Name
        path: namespaces[0].localAdminGroup.groupName
      - description: OpenShift user names added to the Group. Required when enabled.
        displayName: Users
        path: namespaces[0].localAdminGroup.users
"""

DESCRIPTOR_BLOCK = f"""      specDescriptors:
      - description: When enabled, the operator creates and updates platform resources for this tenant namespace. When disabled, reconciliation stops and existing resources are left unchanged (frozen).
        displayName: Reconciliation Enabled
        path: namespaces[0].enabled
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: When enabled, removes operator-managed resources and deletes the tenant namespace, including all workloads still inside it. Opt-in teardown; required before the ProjectOnboarding CR can finish deleting unless the tenant namespace was removed manually from the cluster.
        displayName: Offboard
        path: namespaces[0].offboard
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Tenant namespace to create and manage (DNS-1123 label, e.g. team-payments-dev). Required; no default is pre-filled in the form.
        displayName: Namespace Name
        path: namespaces[0].name
      - description: Pod security, user workload monitoring, and custom labels for the tenant namespace.
        displayName: Additional Namespace Settings
        path: namespaces[0].additionalSettings
      - description: Admission control that blocks pods violating the selected Pod Security profile. Sets pod-security.kubernetes.io/enforce. privileged allows unrestricted workloads; baseline blocks privileged pods and host namespaces; restricted enforces hardened defaults (non-root, dropped capabilities, strict volumes).
        displayName: Pod Security Enforce
        path: namespaces[0].additionalSettings.podSecurityEnforce
{POD_SECURITY_SELECT}
      - description: Warns users when pods violate the profile but still admits them. Sets pod-security.kubernetes.io/warn. Use the same privileged, baseline, or restricted levels as Enforce; often set one level stricter than enforce for early feedback.
        displayName: Pod Security Warn
        path: namespaces[0].additionalSettings.podSecurityWarn
{POD_SECURITY_SELECT}
      - description: Records profile violations in the Kubernetes audit log without blocking or warning. Sets pod-security.kubernetes.io/audit. Useful for observing impact before enabling warn or enforce.
        displayName: Pod Security Audit
        path: namespaces[0].additionalSettings.podSecurityAudit
{POD_SECURITY_SELECT}
      - description: Opt-out setting for user workload monitoring. Enabled by default (sets openshift.io/user-monitoring=true). Uncheck to opt out; the operator sets openshift.io/user-monitoring=false rather than removing the label, because an absent label would re-enable monitoring (OpenShift default is true).
        displayName: User Workload Monitoring
        path: namespaces[0].additionalSettings.enableClusterMonitoring
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
      - description: Custom key/value labels applied to the tenant Namespace. Use Add label to define each key and value; no defaults are pre-filled in the form.
        displayName: Custom Namespace Labels
        path: namespaces[0].additionalSettings.additionalLabels
      - description: Kubernetes label key.
        displayName: Key
        path: namespaces[0].additionalSettings.additionalLabels[0].key
      - description: Kubernetes label value.
        displayName: Value
        path: namespaces[0].additionalSettings.additionalLabels[0].value
      - description: Name of a cluster-scoped TShirtSize (metadata.name). Applies catalogue ResourceQuota and LimitRange presets to this namespace.
        displayName: Project T-Shirt Size (TSS Resource Name)
        path: namespaces[0].projectSize
      - description: Merge inline resourceQuotas and limitRanges onto the referenced T-shirt when projectSize is set.
        displayName: Overwrite T-Shirt
        path: namespaces[0].overwriteTshirt
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:booleanSwitch
{RESOURCE_QUOTA_DESCRIPTORS}{LIMIT_RANGE_DESCRIPTORS}{DEFAULT_POLICIES_DESCRIPTORS}{LOCAL_ADMIN_GROUP_DESCRIPTORS}{NAMESPACE_EXTENDED_DESCRIPTORS}{GITOPS_HIDDEN_DESCRIPTOR}{ARGO_CD_PROJECT_DESCRIPTORS}"""

OWNED_HEAD = re.compile(
    r"(    - description: ProjectOnboarding is a cluster-scoped schema for project onboarding\n"
    r"        API\.\n"
    r"      displayName: Project Onboarding\n"
    r"      kind: ProjectOnboarding\n"
    r"      name: projectonboardings\.onboarding\.stderr\.at\n)"
    r"(?:      specDescriptors:\n(?:(?!      version:).*\n)+)?"
    r"(      version: v1(?:alpha|beta)1\n)"
)

ALM_EXAMPLES = re.compile(
    r"(    alm-examples: \|-\n)(?P<body>(?:      .*\n)+?)(    capabilities:)",
    re.MULTILINE,
)


def strip_projectonboarding_quota_limit_examples(text: str) -> str:
    """Strip form defaults from ProjectOnboarding alm-examples so the console starts empty."""

    def repl(match: re.Match[str]) -> str:
        prefix = match.group(1)
        body = match.group("body")
        suffix = match.group(3)
        json_lines = [line[6:] for line in body.splitlines()]
        examples = json.loads("\n".join(json_lines))
        cleaned = []
        for example in examples:
            if example.get("kind") != "ProjectOnboarding":
                cleaned.append(example)
                continue
            if example.get("metadata", {}).get("name") == "tenant1-onboarding":
                continue
            spec = example.get("spec", {})
            spec.pop("gitOps", None)
            for ns in spec.get("namespaces", []):
                ns.pop("name", None)
                ns.pop("resourceQuotas", None)
                ns.pop("limitRanges", None)
                ns.pop("defaultPolicies", None)
                ns.pop("localAdminGroup", None)
                ns.pop("argoCDProjects", None)
                add_settings = ns.get("additionalSettings")
                if isinstance(add_settings, dict):
                    add_settings.pop("additionalLabels", None)
            cleaned.append(example)
        examples = cleaned
        dumped = json.dumps(examples, indent=2)
        reindented = "\n".join(f"      {line}" for line in dumped.splitlines()) + "\n"
        return prefix + reindented + suffix

    updated, count = ALM_EXAMPLES.subn(repl, text)
    if count != 1:
        sys.exit(f"expected 1 alm-examples block, patched {count}")
    return updated


def patch_csv(path: Path) -> None:
    text = path.read_text()
    updated, count = OWNED_HEAD.subn(r"\1" + DESCRIPTOR_BLOCK + r"\2", text)
    if count != 2:
        sys.exit(f"expected 2 ProjectOnboarding owned entries, patched {count}")
    updated = strip_projectonboarding_quota_limit_examples(updated)
    path.write_text(updated)


def main() -> None:
    repo_root = Path(__file__).resolve().parent.parent
    csv_path = repo_root / "bundle/manifests/project-onboarding-operator.clusterserviceversion.yaml"
    if not csv_path.is_file():
        sys.exit(f"CSV not found: {csv_path}")
    patch_csv(csv_path)


if __name__ == "__main__":
    main()
