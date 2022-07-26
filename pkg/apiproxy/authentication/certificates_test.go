package authentication

import (
	"crypto/x509"
	"log"
	"testing"

	randstr "github.com/thanhpk/randstr"
)

func TestCertificateGeneration(t *testing.T) {
	key, err := GenerateKey()

	if err != nil {
		t.Fatal("couldn't generate key: ", err)
	}

	cert_string, err := GenerateCA(key)

	if err != nil {
		t.Fatal("couldn't generate certificate: ", err)
	}

	log.Print("hello")
	log.Print(*cert_string)

	_, err = x509.ParseCertificate([]byte(*cert_string))

	if err != nil {
		log.Fatal("failed to parse certificate: ", err)
	}
}

func TestTokenSigningRandom(t *testing.T) {
	key, err := GenerateKey()

	if err != nil {
		t.Fatal("couldn't generate key: ", err)
	}

	token := randstr.Hex(32)
	token_signature, err := SignToken(token, key)

	if err != nil {
		t.Fatal(err)
	}

	if token_signature == nil || len(*token_signature) == 0 {
		t.Fatal("signature is empty")
	}

	if VerifyTokenSignature(&key.PublicKey, token, *token_signature) {
		t.Fatal("signatures doesn't match")
	}
}
