package authentication

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"strings"

	randstr "github.com/thanhpk/randstr"
)

func unpack(src []string, dst ...*string) {
	for ind, val := range dst {
		*val = src[ind]
	}
}

type Authorizer struct {
	store        Store
	key          *ecdsa.PrivateKey
	ca_signature string
	ca           *x509.Certificate
}

func NewAuthorizer(certificate string, key string) (*Authorizer, error) {
	cert, err := x509.ParseCertificate([]byte(certificate))

	if err != nil {
		return nil, errors.New("couldn't parse provided certificate")
	}

	pk, err := x509.ParseECPrivateKey([]byte(key))

	if err != nil {
		return nil, errors.New("couldn't parse private key")
	}

	authorizer := Authorizer{
		store:        NewStore(),
		key:          pk,
		ca:           cert,
		ca_signature: hex.EncodeToString(cert.Signature),
	}

	return &authorizer, nil
}

func (auth Authorizer) CreateToken() (*string, error) {
	token := randstr.Hex(32)
	signed_token, err := SignToken(token, auth.key)

	if err != nil {
		return nil, err
	}

	builder := strings.Builder{}

	builder.WriteString(token)
	builder.WriteString(":")
	builder.WriteString(*signed_token)
	builder.WriteString(":")
	builder.WriteString(auth.ca_signature)

	token = builder.String()

	return &token, nil
}

func (auth Authorizer) StoreToken(token string) {
	auth.store.StoreToken(token)
}

func (auth Authorizer) Authorize(token string) error {
	token = strings.TrimSpace(token)

	if token == "" {
		return errors.New("empty token provided")
	}

	parts := strings.Split(token, ":")

	if len(parts) != 3 {
		return errors.New("invalid token format")
	}

	var client_token, client_token_signature, ca_hash string

	unpack(parts, &client_token, &client_token_signature, &ca_hash)

	if ca_hash != auth.ca_signature {
		return errors.New("certificate hash doesn't match")
	}

	actual_token_signature, err := SignToken(client_token, auth.key)

	if err != nil {
		return errors.New("couldn't sign the provided token using the certificate")
	}

	if client_token_signature != *actual_token_signature {
		return errors.New("token signature don't match")
	}

	if !auth.store.TokenExists(client_token) {
		return errors.New("token is invalid")
	}

	return nil
}
