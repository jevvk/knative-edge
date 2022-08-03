package authentication

import (
	"testing"
)

func TestStore(t *testing.T) {
	store := NewStore()

	if store.TokenExists("shouldn't exist") {
		t.Fatal("nothing should be in the store")
	}

	store.StoreToken("foo")
	store.StoreToken("foo2")

	if !store.TokenExists("foo") {
		t.Fatal("'foo' should be in the store")
	}

	if !store.TokenExists("foo2") {
		t.Fatal("'foo2' should be in the store")
	}

	if !store.RemoveToken("foo") {
		t.Fatal("store.RemoveToken should return true if the token was removed")
	}

	if store.RemoveToken("foo3") {
		t.Fatal("store.RemoveToken should return false if the token didn't exist")
	}

	if store.TokenExists("foo") {
		t.Fatal("'foo' shouldn't be in the store")
	}

	if !store.TokenExists("foo2") {
		t.Fatal("'foo2' should be in the store")
	}
}
