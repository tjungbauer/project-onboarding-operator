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

	onboardingv1beta1 "github.com/tjungbauer/project-onboarding-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileNamespace ensures all onboarding resources exist for one namespace entry.
func reconcileDefaultNetworkPolicies(ctx context.Context, c client.Client, scheme *runtime.Scheme, tenantNS *corev1.Namespace, nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) error {
	policies := defaultNetworkPolicies(nsSpec, nsName, labels)
	for _, desired := range policies {
		current := &networkingv1.NetworkPolicy{}
		err := c.Get(ctx, client.ObjectKey{Namespace: nsName, Name: desired.Name}, current)
		if apierrors.IsNotFound(err) {
			if err := ensureTenantNamespaceOwnerRef(scheme, tenantNS, desired); err != nil {
				return err
			}
			if err := c.Create(ctx, desired); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		patch := client.MergeFrom(current.DeepCopy())
		current.Labels = mergeStringMaps(current.Labels, labels)
		current.Spec = desired.Spec
		if err := ensureTenantNamespaceOwnerRef(scheme, tenantNS, current); err != nil {
			return err
		}
		if err := c.Patch(ctx, current, patch); err != nil {
			return err
		}
	}
	return nil
}

func defaultNetworkPolicies(nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) []*networkingv1.NetworkPolicy {
	defaults := nsSpec.DefaultPolicies
	if defaults == nil {
		return nil
	}
	policies := []*networkingv1.NetworkPolicy{}

	if IsOptInEnabled(defaults.AllowFromIngress) {
		policies = append(policies, &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-from-openshift-ingress", Namespace: nsName, Labels: labels},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				Ingress: []networkingv1.NetworkPolicyIngressRule{{
					From: []networkingv1.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"policy-group.network.openshift.io/ingress": ""},
						},
					}},
				}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		})
	}

	if IsOptInEnabled(defaults.AllowFromMonitoring) {
		policies = append(policies, &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-from-openshift-monitoring", Namespace: nsName, Labels: labels},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				Ingress: []networkingv1.NetworkPolicyIngressRule{{
					From: []networkingv1.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"network.openshift.io/policy-group": "monitoring"},
						},
					}},
				}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		})
	}

	if IsOptInEnabled(defaults.AllowToDNS) {
		tcp53, tcp5353, udp53, udp5353 := corev1.ProtocolTCP, corev1.ProtocolTCP, corev1.ProtocolUDP, corev1.ProtocolUDP
		port53, port5353 := intstr.FromInt32(53), intstr.FromInt32(5353)
		policies = append(policies, &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-to-openshift-dns", Namespace: nsName, Labels: labels},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				Egress: []networkingv1.NetworkPolicyEgressRule{{
					To: []networkingv1.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-dns"},
						},
					}},
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &tcp5353, Port: &port5353},
						{Protocol: &tcp53, Port: &port53},
						{Protocol: &udp53, Port: &port53},
						{Protocol: &udp5353, Port: &port5353},
					},
				}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			},
		})
	}

	if IsOptInEnabled(defaults.AllowKubeAPIServer) {
		policies = append(policies, &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-from-kube-apiserver-operator", Namespace: nsName, Labels: labels},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				Ingress: []networkingv1.NetworkPolicyIngressRule{{
					From: []networkingv1.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver-operator"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "kube-apiserver-operator"},
						},
					}},
				}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		})
	}

	if IsOptInEnabled(defaults.DenyAllEgress) {
		policies = append(policies, &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "deny-all-egress", Namespace: nsName, Labels: labels},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			},
		})
	}

	if IsOptInEnabled(defaults.DenyAllIngress) {
		policies = append(policies, &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "deny-all-ingress", Namespace: nsName, Labels: labels},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		})
	}

	if IsOptInEnabled(defaults.AllowFromSameNamespace) {
		policies = append(policies, &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-same-namespace", Namespace: nsName, Labels: labels},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				Ingress: []networkingv1.NetworkPolicyIngressRule{{
					From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{}}},
				}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		})
	}

	return policies
}

func reconcileCustomNetworkPolicies(ctx context.Context, c client.Client, scheme *runtime.Scheme, tenantNS *corev1.Namespace, nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) error {
	for _, polSpec := range nsSpec.NetworkPolicies {
		if !IsEnabled(polSpec.Active) {
			continue
		}
		desired := buildCustomNetworkPolicy(polSpec, nsName, labels)
		current := &networkingv1.NetworkPolicy{}
		err := c.Get(ctx, client.ObjectKey{Namespace: nsName, Name: desired.Name}, current)
		if apierrors.IsNotFound(err) {
			if err := ensureTenantNamespaceOwnerRef(scheme, tenantNS, desired); err != nil {
				return err
			}
			if err := c.Create(ctx, desired); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		patch := client.MergeFrom(current.DeepCopy())
		current.Labels = mergeStringMaps(current.Labels, labels)
		current.Spec = desired.Spec
		if err := ensureTenantNamespaceOwnerRef(scheme, tenantNS, current); err != nil {
			return err
		}
		if err := c.Patch(ctx, current, patch); err != nil {
			return err
		}
	}
	return nil
}

func buildCustomNetworkPolicy(spec onboardingv1beta1.CustomNetworkPolicySpec, nsName string, labels map[string]string) *networkingv1.NetworkPolicy {
	podSelector := metav1.LabelSelector{}
	if spec.PodSelector != nil {
		podSelector = *spec.PodSelector
	}

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SanitizeName(spec.Name),
			Namespace: nsName,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: podSelector,
		},
	}

	for _, rule := range spec.IngressRules {
		np.Spec.Ingress = append(np.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
			From:  buildNetworkPolicyPeers(rule.Selectors),
			Ports: buildNetworkPolicyPorts(rule.Ports),
		})
	}
	for _, rule := range spec.EgressRules {
		np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
			To:    buildNetworkPolicyPeers(rule.Selectors),
			Ports: buildNetworkPolicyPorts(rule.Ports),
		})
	}
	if len(np.Spec.Ingress) > 0 {
		np.Spec.PolicyTypes = append(np.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	}
	if len(np.Spec.Egress) > 0 {
		np.Spec.PolicyTypes = append(np.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
	}
	return np
}

func buildNetworkPolicyPeers(selectors []onboardingv1beta1.NetworkPolicyPeerSpec) []networkingv1.NetworkPolicyPeer {
	peers := make([]networkingv1.NetworkPolicyPeer, 0, len(selectors))
	for _, selector := range selectors {
		peer := networkingv1.NetworkPolicyPeer{}
		if selector.PodSelector != nil {
			peer.PodSelector = selector.PodSelector
		}
		if selector.NamespaceSelector != nil {
			peer.NamespaceSelector = selector.NamespaceSelector
		}
		if selector.IPBlock != nil {
			peer.IPBlock = &networkingv1.IPBlock{
				CIDR:   selector.IPBlock.CIDR,
				Except: selector.IPBlock.Except,
			}
		}
		peers = append(peers, peer)
	}
	return peers
}

func buildNetworkPolicyPorts(ports []onboardingv1beta1.NetworkPolicyPortSpec) []networkingv1.NetworkPolicyPort {
	out := make([]networkingv1.NetworkPolicyPort, 0, len(ports))
	for _, port := range ports {
		npPort := networkingv1.NetworkPolicyPort{}
		if port.Protocol != nil {
			proto := corev1.Protocol(*port.Protocol)
			npPort.Protocol = &proto
		}
		if port.Port != nil {
			p := intstr.FromInt32(*port.Port)
			npPort.Port = &p
		}
		out = append(out, npPort)
	}
	return out
}
