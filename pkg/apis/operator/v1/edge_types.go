/*
Copyright 2022 jevv k.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EdgeSpec defines the desired state of Edge
type EdgeSpec struct {
	// The name of the Edge Cluster. This is used for retrieving EdgeCluster from the remote cluster.
	ClusterName string `json:"clusterName,omitempty"`
	// Override the proxy image for forwarding edge requests to the cloud.
	// +optional
	OverrideProxyImage string `json:"overrideProxyImage,omitempty"`

	// The secret containing the kubeconfig to the remote cluster.
	// +optional
	SecretRef corev1.SecretReference `json:"secretRef,omitempty"`
}

// EdgeStatus defines the observed state of Edge
type EdgeStatus struct {
	// The zone of the edge cluster.
	// +optional
	Zone *string `json:"zone"`
	// The region where the edge cluster is located.
	// +optional
	Region *string `json:"region"`
	// The list of environments which are replicated to the edge cluster.
	Environments []string `json:"environments"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespace
// +kubebuilder:printcolumn:name=Zone,JSONPath=".status.zone",type=string,priority=0
// +kubebuilder:printcolumn:name=Region,JSONPath=".status.region",type=string,priority=0
// +kubebuilder:printcolumn:name=Environments,JSONPath=".status.environments",type=string,priority=1

// Edge is the Schema for the Edges API
type Edge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeSpec   `json:"spec,omitempty"`
	Status EdgeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeList contains a list of Edge
type EdgeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Edge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Edge{}, &EdgeList{})
}
