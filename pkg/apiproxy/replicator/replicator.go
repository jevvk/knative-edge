package replicator

import (
	"net/http"

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

func (r Replicator) NewHandler() http.Handler {

}

func (r Replicator) HandleNewConnection(req http.Request) {

}

func (r Replicator) HandleChanges() {

}
