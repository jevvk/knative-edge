package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// +kubebuilder:object:root=true

// OperatorConfigSpec defines the desired state of OperatorConfig
type OperatorConfigSpec struct {
	metav1.TypeMeta `json:",inline"`

	// ControllerManagerConfigurationSpec returns the contfigurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	// Operator settings
	Options OperatorConfigSpecOptions `json:"operatorOptions"`
}

type OperatorConfigSpecOptions struct {
	// The namespaces which the operator listens to.
	// If no namespaces are provided, all namespaces will be listened to.
	// +optional
	Namespaces *[]string `json:"namespaces,omitempty"`

	// The remote cache sync period of the operator.
	// This applies to the remote clusters where EdgeClusters are defined.
	// +optional
	RemoteSyncPeriod *metav1.Duration `json:"remoteSyncPeriod,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced

// OperatorConfig is the Schema for the OperatorConfigs API
type OperatorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OperatorConfigSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// OperatorConfigList contains a list of OperatorConfig
type OperatorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperatorConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OperatorConfig{}, &OperatorConfigList{})
}
