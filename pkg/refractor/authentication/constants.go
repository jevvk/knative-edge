package authentication

const (
	ApiProxyServiceName      = "refractor-service"
	HTTPPort                 = 8080
	AuthHeader               = "X-K-EDGE-Authorization"
	Namespace                = "knative-edge-system"
	SecretName               = "knative-edge-certificates"
	AuthenticationPath       = "/var/run/secrets/jevv.dev/edge/authentication"
	CertificateAuthorityFile = "ca.pem"
	PrivateKeyFile           = "priv.pem"
)
