package authentication

import (
	"crypto/tls"
	"encoding/hex"
	"errors"
	"strings"
)

func unpack(src []string, dst ...*string) {
	for ind, val := range dst {
		*val = src[ind]
	}
}

type Authenticator struct {
	caSignature string
}

func New(token string) (*Authenticator, error) {
	if token == "" {
		return nil, errors.New("empty token provided")
	}

	var version string

	parts := strings.SplitAfterN(token, "/", 2)

	if len(parts) != 2 {
		return nil, errors.New("invalid token format: cannot retrieve version")
	}

	unpack(parts, &version, &token)

	if version != "v1" {
		return nil, errors.New("invalid token format: invalid version")
	}

	parts = strings.SplitAfterN(token, ":", 2)

	if len(parts) != 3 {
		return nil, errors.New("invalid token format: unknown format for v1")
	}

	return &Authenticator{caSignature: parts[2]}, nil
}

func (a *Authenticator) Authorize(conn tls.ConnectionState) error {
	for _, cert := range conn.PeerCertificates {
		caSignature := hex.EncodeToString(cert.Signature)

		// TODO: do I need to worry about timing-based attacks? it uses sha256 after all
		if a.caSignature == caSignature {
			return nil
		}
	}

	return errors.New("no matching certificate found")
}
