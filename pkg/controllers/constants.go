package controllers

const (
	SystemNamespace = "knative-edge-system"
)

const (
	EnvironmentLabel = "edge.jevv.dev/environment"
	ManagedLabel     = "edge.jevv.dev/managed"
	EdgeLocalLabel   = "edge.jevv.dev/edge-local"
	ManagedByLabel   = "app.kubernetes.io/managed-by"
	CreatedByLabel   = "app.kubernetes.io/created-by"

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
)
