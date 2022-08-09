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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConnectionStatus displays whether the EdgeCluster is connected or not.
// +kubebuilder:validation:Enum=Disconnected;Connected
type ConnectionStatus string

const (
	Disconnected ConnectionStatus = "Disconnected"
	Connected    ConnectionStatus = "Connected"
)

// EdgeClusterSpec defines the desired state of EdgeCluster
type EdgeClusterSpec struct {
	// The zone of the EdgeCluster.
	// +optional
	Zone *string `json:"zone"`
	// The region where the EdgeCluster is located.
	// +optional
	Region *string `json:"region"`
	// The list of environments which are replicated to the EdgeCluster.
	Environments []string `json:"environments"`
}

// EdgeClusterStatus defines the observed state of EdgeCluster
type EdgeClusterStatus struct {
	// The time the EdgeCluster last reported.
	LastReportedAt string `json:"lastReportedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name=Zone,JSONPath=".spec.zone",type=string,priority=0
// +kubebuilder:printcolumn:name=Region,JSONPath=".spec.region",type=string,priority=0
// +kubebuilder:printcolumn:name=Environments,JSONPath=".spec.environments",type=string,priority=1
// +kubebuilder:printcolumn:name="Last Reported",JSONPath=".status.lastReportedAt",type=string,priority=1

// EdgeCluster is the Schema for the edgeclusters API
type EdgeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeClusterSpec   `json:"spec,omitempty"`
	Status EdgeClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeClusterList contains a list of EdgeCluster
type EdgeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeCluster{}, &EdgeClusterList{})
}
