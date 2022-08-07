package authentication

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	randstr "github.com/thanhpk/randstr"
)

func unpack(src []string, dst ...*string) {
	for ind, val := range dst {
		*val = src[ind]
	}
}

type Authenticator struct {
	store       Store
	key         *ecdsa.PrivateKey
	caSignature string
	ca          *x509.Certificate
}

func NewFromLocalFiles() (*Authenticator, error) {
	certificate, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", AuthenticationPath, CertificateAuthorityFile))

	if err != nil {
		return nil, fmt.Errorf("couldn't read certificate authority: %s", err)
	}

	key, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", AuthenticationPath, PrivateKeyFile))

	if err != nil {
		return nil, fmt.Errorf("couldn't read private key: %s", err)
	}

	return New(certificate, key)
}

func New(certificate []byte, key []byte) (*Authenticator, error) {
	pemBlock, _ := pem.Decode(certificate)

	if pemBlock == nil {
		return nil, errors.New("invalid certificate provided")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)

	if err != nil {
		return nil, err
	}

	pemBlock, _ = pem.Decode(key)

	if pemBlock == nil {
		return nil, errors.New("invalid private key provided")
	}

	pk, err := x509.ParseECPrivateKey(pemBlock.Bytes)

	if err != nil {
		return nil, err
	}

	if !pk.PublicKey.Equal(cert.PublicKey) {
		return nil, errors.New("certificate and private key don't match")
	}

	authorizer := Authenticator{
		store:       NewStore(),
		key:         pk,
		ca:          cert,
		caSignature: hex.EncodeToString(cert.Signature),
	}

	return &authorizer, nil
}

func (auth *Authenticator) CreateToken() (*string, error) {
	token := randstr.Hex(32)
	signature, err := SignToken(token, auth.key)

	if err != nil {
		return nil, err
	}

	builder := strings.Builder{}

	builder.WriteString("v1/")
	builder.WriteString(token)
	builder.WriteString(":")
	builder.WriteString(*signature)
	builder.WriteString(":")
	builder.WriteString(auth.caSignature)

	signedToken := builder.String()
	// auth.store.StoreToken(token)

	return &signedToken, nil
}

func (auth *Authenticator) StoreToken(token string) {
	auth.store.StoreToken(token)
}

func (auth *Authenticator) Authorize(token string) error {
	token = strings.TrimSpace(token)

	if token == "" {
		return errors.New("empty token provided")
	}

	parts := strings.SplitAfterN(token, "/", 2)

	if len(parts) != 2 {
		return errors.New("invalid token format: cannot retrieve version")
	}

	var version string

	unpack(parts, &version, &token)

	if version != "v1" {
		return errors.New("invalid token format: invalid version")
	}

	parts = strings.SplitAfterN(token, ":", 3)

	if len(parts) != 3 {
		return errors.New("invalid token format: unknown format for v1")
	}

	var rawToken, rawTokenSignature, caSignature string

	unpack(parts, &rawToken, &rawTokenSignature, &caSignature)

	if !auth.store.TokenExists(rawToken) {
		return errors.New("token is invalid")
	}

	if caSignature != auth.caSignature {
		return errors.New("certificate hash doesn't match")
	}

	if !VerifyTokenSignature(&auth.key.PublicKey, rawToken, rawTokenSignature) {
		return errors.New("token signature is invalid")
	}

	return nil
}
