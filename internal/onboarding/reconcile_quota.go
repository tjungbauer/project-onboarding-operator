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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileNamespace ensures all onboarding resources exist for one namespace entry.
func reconcileResourceQuota(ctx context.Context, c client.Client, scheme *runtime.Scheme, tenantNS *corev1.Namespace, nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) error {
	quotaName := nsName + "-quota"
	desired := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quotaName,
			Namespace: nsName,
			Labels:    labels,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: buildResourceQuotaHard(nsSpec.ResourceQuotas),
		},
	}

	current := &corev1.ResourceQuota{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nsName, Name: quotaName}, current)
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
	current.Spec.Hard = desired.Spec.Hard
	if err := ensureTenantNamespaceOwnerRef(scheme, tenantNS, current); err != nil {
		return err
	}
	return c.Patch(ctx, current, patch)
}

func buildResourceQuotaHard(spec *onboardingv1beta1.ResourceQuotaSpec) corev1.ResourceList {
	hard := corev1.ResourceList{}
	setQuantity := func(name corev1.ResourceName, value *string) {
		if value == nil || strings.TrimSpace(*value) == "" {
			return
		}
		hard[name] = resource.MustParse(NormalizeQuantity(*value))
	}
	setCount := func(name corev1.ResourceName, value *int32) {
		if value == nil {
			return
		}
		hard[name] = resource.MustParse(fmt.Sprintf("%d", *value))
	}

	setCount(corev1.ResourcePods, spec.Pods)
	setQuantity(corev1.ResourceCPU, spec.CPU)
	setQuantity(corev1.ResourceMemory, spec.Memory)
	setQuantity(corev1.ResourceEphemeralStorage, spec.EphemeralStorage)
	setCount(corev1.ResourceReplicationControllers, spec.ReplicationControllers)
	setCount(corev1.ResourceQuotas, spec.ResourceQuotas)
	setCount(corev1.ResourceServices, spec.Services)
	setCount(corev1.ResourceSecrets, spec.Secrets)
	setCount(corev1.ResourceConfigMaps, spec.ConfigMaps)
	setCount(corev1.ResourcePersistentVolumeClaims, spec.PersistentVolumeClaims)

	if spec.Limits != nil {
		setQuantity(corev1.ResourceLimitsCPU, spec.Limits.CPU)
		setQuantity(corev1.ResourceLimitsMemory, spec.Limits.Memory)
		setQuantity(corev1.ResourceLimitsEphemeralStorage, spec.Limits.EphemeralStorage)
	}
	if spec.Requests != nil {
		setQuantity(corev1.ResourceRequestsCPU, spec.Requests.CPU)
		setQuantity(corev1.ResourceRequestsMemory, spec.Requests.Memory)
		setQuantity(corev1.ResourceRequestsStorage, spec.Requests.Storage)
		setQuantity(corev1.ResourceRequestsEphemeralStorage, spec.Requests.EphemeralStorage)
	}
	for _, entry := range spec.StorageClasses {
		if entry.Key == "" {
			continue
		}
		hard[corev1.ResourceName(entry.Key)] = resource.MustParse(NormalizeQuantity(entry.Value))
	}
	return hard
}

func reconcileLimitRange(ctx context.Context, c client.Client, scheme *runtime.Scheme, tenantNS *corev1.Namespace, nsSpec onboardingv1beta1.NamespaceSpec, nsName string, labels map[string]string) error {
	limitName := nsName + "-limitrange"
	desired := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      limitName,
			Namespace: nsName,
			Labels:    labels,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: buildLimitRangeItems(nsSpec.LimitRanges),
		},
	}

	current := &corev1.LimitRange{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nsName, Name: limitName}, current)
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
	current.Spec.Limits = desired.Spec.Limits
	if err := ensureTenantNamespaceOwnerRef(scheme, tenantNS, current); err != nil {
		return err
	}
	return c.Patch(ctx, current, patch)
}

func buildLimitRangeItems(spec *onboardingv1beta1.LimitRangeSpec) []corev1.LimitRangeItem {
	items := []corev1.LimitRangeItem{}
	if spec.Pod != nil {
		items = append(items, corev1.LimitRangeItem{
			Type:                 corev1.LimitTypePod,
			Max:                  resourceAmount(spec.Pod.Max),
			Min:                  resourceAmount(spec.Pod.Min),
			DefaultRequest:       nil,
			Default:              nil,
			MaxLimitRequestRatio: nil,
		})
	}
	if spec.Container != nil {
		items = append(items, corev1.LimitRangeItem{
			Type:           corev1.LimitTypeContainer,
			Max:            resourceAmount(spec.Container.Max),
			Min:            resourceAmount(spec.Container.Min),
			Default:        resourceAmount(spec.Container.Default),
			DefaultRequest: resourceAmount(spec.Container.DefaultRequest),
		})
	}
	if spec.PVC != nil {
		item := corev1.LimitRangeItem{Type: corev1.LimitTypePersistentVolumeClaim}
		if spec.PVC.Min != nil && spec.PVC.Min.Storage != nil {
			item.Min = corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(NormalizeQuantity(*spec.PVC.Min.Storage)),
			}
		}
		if spec.PVC.Max != nil && spec.PVC.Max.Storage != nil {
			item.Max = corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(NormalizeQuantity(*spec.PVC.Max.Storage)),
			}
		}
		items = append(items, item)
	}
	return items
}

func resourceAmount(spec *onboardingv1beta1.ResourceAmountSpec) corev1.ResourceList {
	if spec == nil {
		return nil
	}
	out := corev1.ResourceList{}
	if spec.CPU != nil {
		out[corev1.ResourceCPU] = resource.MustParse(*spec.CPU)
	}
	if spec.Memory != nil {
		out[corev1.ResourceMemory] = resource.MustParse(NormalizeQuantity(*spec.Memory))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
