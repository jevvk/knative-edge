package authentication

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	randstr "github.com/thanhpk/randstr"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

func Initialize(ctx context.Context, secretName string) {
	config, err := rest.InClusterConfig()

	if err != nil {
		panic(fmt.Errorf("couldn't retrieve kubernetes client: %s", err))
	}

	kubeClient := kubernetes.NewForConfigOrDie(config)
	secrets := kubeClient.CoreV1().Secrets(Namespace)

	secret, err := secrets.Get(ctx, secretName, metaV1.GetOptions{})

	if apierrs.IsNotFound(err) {
		privateKey, err := GenerateKey()

		if err != nil {
			panic(fmt.Errorf("couldn't generate private key: %s", err))
		}

		privateKeyPem, err := EncodeKey(privateKey)

		if err != nil {
			panic(fmt.Errorf("couldn't encode private key: %s", err))
		}

		caPem, err := GenerateCA(privateKey)

		if err != nil {
			panic(fmt.Errorf("couldn't generate certificate authority: %s", err))
		}

		secret.Data[PrivateKeyFile] = []byte(*privateKeyPem)
		secret.Data[CertificateAuthorityFile] = []byte(*caPem)

		_, err = secrets.Update(ctx, secret, metaV1.UpdateOptions{})

		if err != nil {
			panic(fmt.Errorf("couldn't update secret: %s", err))
		}
	} else if err != nil {
		panic(fmt.Errorf("couldn't retrieve private key: %s", err))
	}
}

func New(certificate string, key string) (*Authenticator, error) {
	pemBlock, _ := pem.Decode([]byte(certificate))

	if pemBlock == nil {
		return nil, errors.New("invalid certificate provided")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)

	if err != nil {
		return nil, err
	}

	pemBlock, _ = pem.Decode([]byte(key))

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

func (auth Authenticator) CreateToken() (*string, error) {
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
	builder.WriteString(auth.caSignature)

	token = builder.String()

	return &token, nil
}

func (auth Authenticator) StoreToken(token string) {
	auth.store.StoreToken(token)
}

func (auth Authenticator) Authorize(token string) error {
	token = strings.TrimSpace(token)

	if token == "" {
		return errors.New("empty token provided")
	}

	parts := strings.Split(token, ":")

	if len(parts) != 3 {
		return errors.New("invalid token format")
	}

	var rawToken, rawTokenSignature, caSignature string

	unpack(parts, &rawToken, &rawTokenSignature, &caSignature)

	if caSignature != auth.caSignature {
		return errors.New("certificate hash doesn't match")
	}

	if !VerifyTokenSignature(&auth.key.PublicKey, rawToken, rawTokenSignature) {
		return errors.New("token signature is invalid")
	}

	if !auth.store.TokenExists(token) {
		return errors.New("token is invalid")
	}

	return nil
}
