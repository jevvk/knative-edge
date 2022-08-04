package authentication

import (
	"context"
	"fmt"
	"log"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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
