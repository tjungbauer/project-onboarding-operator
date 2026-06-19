package onboarding

import (
	"testing"

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
)

func TestBuildRolePoliciesFullAccess(t *testing.T) {
	policies := buildRolePolicies("tenant1-app-1", "tenant1-app-1", onboardingv1beta1.ArgoCDRoleSpec{
		Name:       "write",
		FullAccess: boolPtr(true),
	})
	if len(policies) != 10 {
		t.Fatalf("expected 10 fullaccess policies, got %d", len(policies))
	}
	if policies[0] != "p, proj:tenant1-app-1:write, applications, get, tenant1-app-1/*, allow" {
		t.Fatalf("unexpected fullaccess policy: %s", policies[0])
	}
}

func TestBuildRolePoliciesCasbinPrefix(t *testing.T) {
	defaultPolicy := buildRolePolicies("argo-project", "tenant-ns", onboardingv1beta1.ArgoCDRoleSpec{
		Name: "write",
		Policies: []onboardingv1beta1.ArgoCDPolicySpec{
			{Action: "get"},
		},
	})
	if len(defaultPolicy) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(defaultPolicy))
	}
	wantDefault := "p, proj:tenant-ns:write, applications, get, argo-project/*, allow"
	if defaultPolicy[0] != wantDefault {
		t.Fatalf("default prefix:\n  got  %q\n  want %q", defaultPolicy[0], wantDefault)
	}

	overridePolicy := buildRolePolicies("argo-project", "tenant-ns", onboardingv1beta1.ArgoCDRoleSpec{
		Name: "write",
		Policies: []onboardingv1beta1.ArgoCDPolicySpec{
			{Action: "get", AppProjectName: "legacy-project"},
		},
	})
	wantOverride := "p, proj:legacy-project:write, applications, get, argo-project/*, allow"
	if overridePolicy[0] != wantOverride {
		t.Fatalf("override prefix:\n  got  %q\n  want %q", overridePolicy[0], wantOverride)
	}
}

func TestResolveSourceReposDefaults(t *testing.T) {
	po := &onboardingv1beta1.ProjectOnboarding{
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			GitOps: &onboardingv1beta1.GitOpsSpec{
				AllowedSourceRepos: []string{"https://git.example.com/repo"},
			},
		},
	}
	repos := resolveSourceRepos(po, onboardingv1beta1.ArgoCDProjectSpec{Name: "tenant1-app-1"})
	if len(repos) != 1 || repos[0] != "https://git.example.com/repo" {
		t.Fatalf("unexpected repos: %v", repos)
	}
}

func TestGitOpsManagedByLabel(t *testing.T) {
	po := &onboardingv1beta1.ProjectOnboarding{
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			GitOps: &onboardingv1beta1.GitOpsSpec{ApplicationNamespace: "gitops-legacy"},
		},
	}
	ns := onboardingv1beta1.NamespaceSpec{
		Name:                       "tenant-a",
		ApplicationGitOpsNamespace: "gitops-application",
	}
	if gitOpsManagedByLabel(po, ns) != "gitops-application" {
		t.Fatalf("namespace entry should win over legacy spec.gitOps")
	}
	if gitOpsManagedByLabel(po, onboardingv1beta1.NamespaceSpec{Name: "tenant-b"}) != "gitops-legacy" {
		t.Fatalf("expected legacy spec.gitOps fallback")
	}
}

func TestResolveApplicationGitOpsNamespacePrecedence(t *testing.T) {
	po := &onboardingv1beta1.ProjectOnboarding{
		Spec: onboardingv1beta1.ProjectOnboardingSpec{
			GitOps: &onboardingv1beta1.GitOpsSpec{ApplicationNamespace: "gitops-default"},
		},
	}
	ns := onboardingv1beta1.NamespaceSpec{ApplicationGitOpsNamespace: "gitops-tenant"}
	if got := ResolveApplicationGitOpsNamespace(po, ns); got != "gitops-tenant" {
		t.Fatalf("want gitops-tenant, got %q", got)
	}
}
