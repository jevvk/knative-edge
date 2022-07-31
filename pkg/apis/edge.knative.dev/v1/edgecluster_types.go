package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EdgeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EdgeClusterSpec `json:"spec"`
}

type EdgeClusterSpec struct {
	// The zone of the EdgeCluster
	// +optional
	Zone *string `json:"zone"`
	// The region where the EdgeCluster is located.
	// +optional
	Region *string `json:"region"`
	// The list of namespaces which are replicated to the EdgeCluster.
	Namespaces []string `json:"namespaces"`
}

type EdgeClusterStatus struct {
	// An EdgeCluster can be either connected or disconnected.
	ConnectionStatus string
	// The authentication token of the EdgeCluster. This token is cannot be used in its raw form, as it doesn't include the signature or certificate authority hash.
	AuthenticationToken string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EdgeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EdgeCluster `json:"items"`
}
