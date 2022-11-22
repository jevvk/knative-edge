package controllers

const (
	SystemNamespace          = "knative-edge-system"
	ControllerServiceAccount = "knative-edge-controller"
)

const (
	EnvironmentLabel = "edge.jevv.dev/environment"
	ManagedLabel     = "edge.jevv.dev/managed"
	EdgeLocalLabel   = "edge.jevv.dev/local"
	EdgeTagLabel     = "edge.jevv.dev/tag"
	ManagedByLabel   = "edge.jevv.dev/managed-by"
	CreatedByLabel   = "edge.jevv.dev/created-by"
	EdgeOffloadLabel = "edge.jevv.dev/edge-offload"
	EdgeProxyLabel   = "edge.jevv.dev/edge-proxy"

	KServiceLabel    = "serving.knative.dev/service"
	KServiceUIDLabel = "serving.knative.dev/serviceUID"

	AppLabel        = "app"
	ServiceLabel    = "service"
	ControllerLabel = "controller"
)

const (
	ManagedByLabelValue = "knative-edge"
	CreatedByLabelValue = "controller-manager"
)

const (
	EdgeOffloadOptionsAnnotation = "edge.jevv.dev/edge-offload-options"
	EdgeOffloadLastRunAnnotation = "edge.jevv.dev/edge-offload-last-run"
	EdgeProxyTrafficAnnotation   = "edge.jevv.dev/edge-proxy-traffic"
	ObservedGenerationAnnotation = "edge.jevv.dev/observed-generation"
	ProxyImageAnnotation         = "edge.jevv.dev/proxy-image"
	ControllerImageAnnotation    = "edge.jevv.dev/controller-image"

	LastGenerationAnnotation       = "edge.jevv.dev/last-observed-generation"
	LastRemoteGenerationAnnotation = "edge.jevv.dev/last-observed-remote-generation"
	RemoteUrlAnnotation            = "edge.jevv.dev/remote-url"
	RemoteHostAnnotation           = "edge.jevv.dev/remote-host"

	KnativeNoGCAnnotation = "serving.knative.dev/no-gc"
)

const (
	PrometheusUrlEnv      = "PROMETHEUS_URL"
	PrometheusUserEnv     = "PROMETHEUS_BASIC_USER"
	PrometheusPasswordEnv = "PROMETHEUS_BASIC_PASSWORD"
)
