package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// +kubebuilder:object:root=true

// OperatorConfig is the Schema for the operatorconfigs API
type OperatorConfig struct {
	metav1.TypeMeta `json:",inline"`

	// ControllerManagerConfigurationSpec returns the contfigurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	// Operator settings
	Options OperatorConfigOptions `json:"operatorOptions"`
}

type OperatorConfigOptions struct {
	// The namespaces which the operator listens to.
	// If no namespaces are provided, all namespaces will be listened to.
	// +optional
	Namespaces *[]string `json:"watchNamespaces,omitempty"`

	// The remote cache sync period of the operator.
	// This applies to the remote clusters where EdgeClusters are defined.
	// +optional
	RemoteSyncPeriod *metav1.Duration `json:"remoteSyncPeriod,omitempty"`
}

func init() {
	SchemeBuilder.Register(&OperatorConfig{})
}
