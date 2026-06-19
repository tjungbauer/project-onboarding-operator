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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TShirtSizePhaseReady   = "Ready"
	TShirtSizePhaseInvalid = "Invalid"

	TShirtSizeConditionReady = "Ready"
)

// TShirtSizeSpec defines quota and limit presets for a named project size.
// +kubebuilder:validation:XValidation:rule="has(self.resourceQuotas) || has(self.limitRanges)",message="at least one of resourceQuotas or limitRanges must be set"
type TShirtSizeSpec struct {
	// Description is a human-readable summary of this size.
	// +optional
	Description string `json:"description,omitempty"`

	// ResourceQuotas are applied to namespaces that reference this T-shirt size.
	// +optional
	ResourceQuotas *ResourceQuotaSpec `json:"resourceQuotas,omitempty"`

	// LimitRanges are applied to namespaces that reference this T-shirt size.
	// +optional
	LimitRanges *LimitRangeSpec `json:"limitRanges,omitempty"`
}

// TShirtSizeStatus defines the observed state of TShirtSize.
type TShirtSizeStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	Phase string `json:"phase,omitempty"`
	// +optional
	ReferencedBy int32 `json:"referencedBy,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=tts
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Refs",type=integer,JSONPath=`.status.referencedBy`
// +kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`

// TShirtSize is a cluster-scoped catalogue entry to pre-define project sizes (S, M, L, ...).
type TShirtSize struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TShirtSizeSpec   `json:"spec,omitempty"`
	Status TShirtSizeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TShirtSizeList contains a list of TShirtSize.
type TShirtSizeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TShirtSize `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TShirtSize{}, &TShirtSizeList{})
}
