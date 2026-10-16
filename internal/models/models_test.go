package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUserJSONOmitsPasswordHash(t *testing.T) {
	user := User{ID: "user-1", Name: "Napat", Email: "n@example.com", PasswordHash: "secret-hash"}
	b, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret-hash") || strings.Contains(string(b), "password_hash") {
		t.Fatalf("serialized user exposes password hash: %s", b)
	}
}

func TestIdeaJSONOmitsSellerAssociation(t *testing.T) {
	idea := Idea{ID: 7, SellerID: "user-1", Seller: User{ID: "user-1", Name: "Napat", Email: "n@example.com", PasswordHash: "secret-hash"}}
	b, err := json.Marshal(idea)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret-hash") || strings.Contains(string(b), `"seller"`) || strings.Contains(string(b), "n@example.com") {
		t.Fatalf("serialized idea exposes seller association: %s", b)
	}
}
