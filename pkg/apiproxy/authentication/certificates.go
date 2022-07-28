package authentication

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"log"
	"math/big"
	"time"
)

// func pemBlockForKey(priv interface{}) *pem.Block {
// 	switch k := priv.(type) {
// 	case *rsa.PrivateKey:
// 		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}

// 	case *ecdsa.PrivateKey:
// 		b, err := x509.MarshalECPrivateKey(k)

// 		if err != nil {
// 			log.Fatal("Unable to marshal ECDSA private key", err)
// 		}

// 		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}

// 	default:
// 		return nil
// 	}
// }

func GenerateKey() (*ecdsa.PrivateKey, error) {
	// priv, err := rsa.GenerateKey(rand.Reader, *rsaBits)
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func EncodeKey(key *ecdsa.PrivateKey) (*string, error) {
	if key == nil {
		return nil, errors.New("no key provided")
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)

	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	cert_string := out.String()

	return &cert_string, nil
}

func GenerateCA(pk *ecdsa.PrivateKey) (*string, error) {
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Knative Edge"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	/*
	   hosts := strings.Split(*host, ",")
	   for _, h := range hosts {
	   	if ip := net.ParseIP(h); ip != nil {
	   		template.IPAddresses = append(template.IPAddresses, ip)
	   	} else {
	   		template.DNSNames = append(template.DNSNames, h)
	   	}
	   }
	   if *isCA {
	   	template.IsCA = true
	   	template.KeyUsage |= x509.KeyUsageCertSign
	   }
	*/

	cert_bytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &pk.PublicKey, pk)

	if err != nil {
		log.Fatal("Failed to create certificate", err)
	}

	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: cert_bytes})

	cert_string := out.String()

	return &cert_string, nil
}

func SignToken(token string, key *ecdsa.PrivateKey) (*string, error) {
	hash := sha256.Sum256([]byte(token))
	signature, err := ecdsa.SignASN1(rand.Reader, key, hash[:])

	if err != nil {
		return nil, errors.New("could not sign token")
	}

	signature_string := hex.EncodeToString(signature)

	return &signature_string, nil
}

func VerifyTokenSignature(key *ecdsa.PublicKey, token string, signature string) bool {
	signatureBytes, err := hex.DecodeString(signature)

	if err != nil {
		return false
	}

	hash := sha256.Sum256([]byte(token))
	return ecdsa.VerifyASN1(key, hash[:], signatureBytes)
}
