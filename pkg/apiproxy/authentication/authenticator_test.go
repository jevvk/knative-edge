package authentication

import (
	"fmt"
	"strings"
	"testing"
)

func TestAuthorizer(t *testing.T) {
	key, err := GenerateKey()

	if err != nil {
		t.Fatal(err)
	}

	keyString, err := EncodeKey(key)

	if err != nil {
		t.Fatal(err)
	}

	certString, err := GenerateCA(key)

	if err != nil {
		t.Fatal(err)
	}

	auth, err := New(*certString, *keyString)

	if err != nil {
		t.Fatal(err)
	}

	token, err := auth.CreateToken()

	if err != nil {
		t.Fatal(err)
	}

	if auth.Authorize(*token) == nil {
		t.Fatal("unsaved token shouldn't be authorized")
	}

	auth.StoreToken(*token)

	if err = auth.Authorize(*token); err != nil {
		t.Fatalf("token should be authorized: %s", err)
	}

	token2 := "hello world"
	if auth.Authorize(token2) == nil {
		t.Fatal("invalid token shouldn't be authorized")
	}

	token3 := "t:t:t"
	if auth.Authorize(token3) == nil {
		t.Fatal("invalid token shouldn't be authorized")
	}

	fmt.Print(*token)

	tokenParts := strings.Split(*token, ":")

	token4 := fmt.Sprintf("%s:%s:%s", tokenParts[0], tokenParts[1], "foo")
	if auth.Authorize(token4) == nil {
		t.Fatal("invalid token shouldn't be authorized")
	}

	token5 := fmt.Sprintf("%s:%s:%s", tokenParts[0], "foo", tokenParts[1])
	if auth.Authorize(token5) == nil {
		t.Fatal("invalid token shouldn't be authorized")
	}

	token6 := fmt.Sprintf("%s:%s:%s", "foo", tokenParts[1], tokenParts[1])
	if auth.Authorize(token6) == nil {
		t.Fatal("invalid token shouldn't be authorized")
	}

	// testing different certificate

	key2, err := GenerateKey()

	if err != nil {
		t.Fatal(err)
	}

	keyString2, err := EncodeKey(key2)

	if err != nil {
		t.Fatal(err)
	}

	certString2, err := GenerateCA(key2)

	if err != nil {
		t.Fatal(err)
	}

	auth2, err := New(*certString2, *keyString2)

	if err != nil {
		t.Fatal(err)
	}

	token7, err := auth2.CreateToken()

	if err != nil {
		t.Fatal(err)
	}

	if auth.Authorize(*token7) == nil {
		t.Fatal("token from different certificate shouldn't work")
	}

	auth2.StoreToken(*token7)

	if auth.Authorize(*token7) == nil {
		t.Fatal("token from different certificate shouldn't work")
	}

	auth.StoreToken(*token7)

	if auth.Authorize(*token7) == nil {
		t.Fatal("token from different certificate shouldn't work")
	}

}

func TestAuthenticatorConstructor(t *testing.T) {
	key, err := GenerateKey()

	if err != nil {
		t.Fatal(err)
	}

	keyString, err := EncodeKey(key)

	if err != nil {
		t.Fatal(err)
	}

	certString, err := GenerateCA(key)

	if err != nil {
		t.Fatal(err)
	}

	_, err = New(*certString, *keyString)

	if err != nil {
		t.Fatal(err)
	}

	// now different private key

	key2, err := GenerateKey()

	if err != nil {
		t.Fatal(err)
	}

	keyString2, err := EncodeKey(key2)

	if err != nil {
		t.Fatal(err)
	}

	_, err = New(*certString, *keyString2)

	if err == nil {
		t.Fatal("certificate and private key should match")
	}
}
