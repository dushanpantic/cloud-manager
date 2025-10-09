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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type AzureManagedRedisSKU struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=1;2;3;4;5
	Capacity int `json:"capacity"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule=(self == oldSelf), message="Family is immutable."
	Family string `json:"family,omitempty"`
}

// AzureManagedRedisInstanceSpec defines the desired state of AzureManagedRedisInstance.
// This mirrors the Azure portion of RedisInstance spec, limited to fields supported for Managed Redis.
type AzureManagedRedisInstanceSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule=(self == oldSelf), message="RemoteRef is immutable."
	RemoteRef RemoteRef `json:"remoteRef"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule=(size(self.name) > 0), message="IpRange name must not be empty."
	IpRange IpRangeRef `json:"ipRange"`

	// +kubebuilder:validation:Required
	Scope ScopeRef `json:"scope"`

	// +kubebuilder:validation:Required
	SKU AzureManagedRedisSKU `json:"sku"`

	// +optional
	RedisConfiguration RedisInstanceAzureConfigs `json:"redisConfiguration,omitempty"`

	// +optional
	RedisVersion string `json:"redisVersion,omitempty"`

	// +optional
	ShardCount int `json:"shardCount,omitempty"`
}

// AzureManagedRedisInstanceStatus defines the observed state of AzureManagedRedisInstance
type AzureManagedRedisInstanceStatus struct {
	State StatusState `json:"state,omitempty"`

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
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AzureManagedRedisInstance is the Schema for the azuremanagedredisinstances API
// +kubebuilder:resource:categories={kyma-cloud-manager}
// +kubebuilder:printcolumn:name="Scope",type="string",JSONPath=".spec.scope.name"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type AzureManagedRedisInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureManagedRedisInstanceSpec   `json:"spec,omitempty"`
	Status AzureManagedRedisInstanceStatus `json:"status,omitempty"`
}

func (in *AzureManagedRedisInstance) ScopeRef() ScopeRef                { return in.Spec.Scope }
func (in *AzureManagedRedisInstance) SetScopeRef(scopeRef ScopeRef)     { in.Spec.Scope = scopeRef }
func (in *AzureManagedRedisInstance) Conditions() *[]metav1.Condition   { return &in.Status.Conditions }
func (in *AzureManagedRedisInstance) State() string                     { return string(in.Status.State) }
func (in *AzureManagedRedisInstance) SetState(v string)                 { in.Status.State = StatusState(v) }
func (in *AzureManagedRedisInstance) GetObjectMeta() *metav1.ObjectMeta { return &in.ObjectMeta }
func (in *AzureManagedRedisInstance) CloneForPatchStatus() client.Object {
	result := &AzureManagedRedisInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureManagedRedisInstance",
			APIVersion: GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: in.Namespace,
			Name:      in.Name,
		},
		Status: in.Status,
	}
	if result.Status.Conditions == nil {
		result.Status.Conditions = []metav1.Condition{}
	}
	return result
}
func (in *AzureManagedRedisInstance) SetStatusStateToReady() { in.Status.State = StateReady }
func (in *AzureManagedRedisInstance) SetStatusStateToError() { in.Status.State = StateError }

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
