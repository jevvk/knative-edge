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

// EdgeResourceSpec defines the desired state of EdgeResource
type EdgeResourceSpec struct {
	// The apiVersion of the remote resource
	// +kubebuilder:printcolumn
	ApiVersion string `json:"resource.apiVersion"`
	// The kind of the remote resource
	// +kubebuilder:printcolumn
	Kind string `json:"resource.kind"`

	// The resourceVersion of the remote resource
	// +kubebuilder:printcolumn
	RemoteResourceVersion string `json:"resource.remote.version"`
	// The definition of the remote resource
	Data string `json:"data"`
}

// EdgeResourceStatus defines the observed state of EdgeResource
type EdgeResourceStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespace

// EdgeResource is the Schema for the edgeclusters API
type EdgeResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeResourceSpec   `json:"spec,omitempty"`
	Status EdgeResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeResourceList contains a list of EdgeResource
type EdgeResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeResource{}, &EdgeResourceList{})
}
