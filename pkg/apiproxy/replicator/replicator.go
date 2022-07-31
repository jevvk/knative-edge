package replicator

import (
	"knative.dev/edge/pkg/apiproxy/authentication"
)

type Replicator struct {
	auth authentication.Authenticator
}

func New(authenticator authentication.Authenticator) Replicator {
	return Replicator{
		auth: authenticator,
	}
}
