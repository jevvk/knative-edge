package controllers

const (
	SystemNamespace          = "knative-edge-system"
	ControllerServiceAccount = "knative-edge-controller"
)

const (
	EnvironmentLabel = "edge-environment"
	ManagedLabel     = "edge-managed"
	EdgeLocalLabel   = "edge-local"
	ManagedByLabel   = "managed-by"
	CreatedByLabel   = "created-by"

	AppLabel        = "app"
	ServiceLabel    = "service"
	ControllerLabel = "controller"
)

const (
	ManagedByLabelValue = "knative-edge"
	CreatedByLabelValue = "controller-manager"
)

const (
	ObserverGenerationAnnotation = "edge.jevv.dev/observed-generation"
	ProxyImageAnnotation         = "edge.jevv.dev/proxy-image"
	ControllerImageAnnotation    = "edge.jevv.dev/controller-image"
)
