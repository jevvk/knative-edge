package authentication

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
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

	skipGeneration := false
	resetSecret := false

	if err == nil {
		privSecret, privExists := secret.Data[PrivateKeyFile]
		caSecret, caExists := secret.Data[CertificateAuthorityFile]

		if !privExists || !caExists {
			resetSecret = true
		} else if len(privSecret) == 0 || len(caSecret) == 0 {
			resetSecret = true
		} else {
			skipGeneration = true
		}
	} else if !apierrs.IsNotFound(err) {
		panic(fmt.Errorf("couldn't retrieve private key: %s", err))
	}

	if !skipGeneration {
		log.Printf("Couldn't find secrets/%s in namespaces/%s. A new private key and certificate will be generated.", secretName, Namespace)

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

		secret.Name = secretName
		secret.Namespace = Namespace
		secret.Data = make(map[string][]byte)
		secret.Data[PrivateKeyFile] = []byte(*privateKeyPem)
		secret.Data[CertificateAuthorityFile] = []byte(*caPem)

		if resetSecret {
			_, err = secrets.Update(ctx, secret, metaV1.UpdateOptions{})
		} else {
			_, err = secrets.Create(ctx, secret, metaV1.CreateOptions{})
		}

		if err != nil {
			panic(fmt.Errorf("couldn't update secret: %s", err))
		}

		log.Printf("New secret created.")
	} else {
		log.Printf("Already found secrets/%s in namespaces/%s.", secretName, Namespace)
	}
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

	parts := strings.Split(token, ":")

	if len(parts) != 3 {
		return errors.New("invalid token format")
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
