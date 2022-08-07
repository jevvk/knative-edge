package authentication

const (
	ApiProxyServiceName      = "apiproxy-service"
	HTTPPort                 = 8080
	AuthHeader               = "X-K-EDGE-Authorization"
	Namespace                = "knative-edge-system"
	SecretName               = "knative-edge-certificates"
	AuthenticationPath       = "/var/run/secrets/knative.dev/edge/authentication"
	CertificateAuthorityFile = "ca.pem"
	PrivateKeyFile           = "priv.pem"
)