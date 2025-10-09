/*
Copyright 2023.

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
	featuretypes "github.com/kyma-project/cloud-manager/pkg/feature/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AzureManagedRedisInstanceSpec defines the desired state for runtime plane (mostly projection + ipRange binding)
type AzureManagedRedisInstanceSpec struct {
	// +optional
	IpRange IpRangeRef `json:"ipRange"`
}

// AzureManagedRedisInstanceStatus defines the observed state
type AzureManagedRedisInstanceStatus struct {
	// +optional
	Id string `json:"id,omitempty"`

	// +optional
	PrimaryEndpoint string `json:"primaryEndpoint,omitempty"`

	// +optional
	ReadEndpoint string `json:"readEndpoint,omitempty"`

	// +optional
	AuthString string `json:"authString,omitempty"`

	// +optional
	CaCert string `json:"caCert,omitempty"`

	// +optional
	State string `json:"state,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AzureManagedRedisInstance is the Schema for the azuremanagedredisinstances API
type AzureManagedRedisInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureManagedRedisInstanceSpec   `json:"spec,omitempty"`
	Status AzureManagedRedisInstanceStatus `json:"status,omitempty"`
}

func (in *AzureManagedRedisInstance) GetIpRangeRef() IpRangeRef         { return in.Spec.IpRange }
func (in *AzureManagedRedisInstance) Conditions() *[]metav1.Condition   { return &in.Status.Conditions }
func (in *AzureManagedRedisInstance) GetObjectMeta() *metav1.ObjectMeta { return &in.ObjectMeta }
func (in *AzureManagedRedisInstance) SpecificToFeature() featuretypes.FeatureName {
	return featuretypes.FeatureRedis
}
func (in *AzureManagedRedisInstance) SpecificToProviders() []string { return []string{"azure"} }
func (in *AzureManagedRedisInstance) State() string                 { return in.Status.State }
func (in *AzureManagedRedisInstance) SetState(v string)             { in.Status.State = v }

// +kubebuilder:object:root=true

// AzureManagedRedisInstanceList contains a list of AzureManagedRedisInstance
type AzureManagedRedisInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureManagedRedisInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzureManagedRedisInstance{}, &AzureManagedRedisInstanceList{})
}
