package auth

import (
	"strings"
	"testing"
	"time"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == password || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("HashPassword() returned an unexpected hash: %q", hash)
	}
	if !CheckPassword(hash, password) {
		t.Fatal("CheckPassword() rejected the matching password")
	}
	if CheckPassword(hash, "wrong password") {
		t.Fatal("CheckPassword() accepted a different password")
	}
	if CheckPassword("not-a-bcrypt-hash", password) {
		t.Fatal("CheckPassword() accepted a malformed hash")
	}
}

func TestManagerIssueAndParse(t *testing.T) {
	manager := NewManager(strings.Repeat("s", 32), time.Hour)
	token, err := manager.Issue("user-123")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	got, err := manager.Parse(token)
	if err != nil || got != "user-123" {
		t.Fatalf("Parse() = (%q, %v), want user-123", got, err)
	}
}

func TestManagerRejectsInvalidTokens(t *testing.T) {
	manager := NewManager(strings.Repeat("s", 32), time.Hour)
	otherKey := NewManager(strings.Repeat("x", 32), time.Hour)
	valid, err := manager.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}
	expired, err := NewManager(strings.Repeat("s", 32), -time.Hour).Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		input string
		parse func(string) (string, error)
	}{
		{"malformed", "not-a-jwt", manager.Parse},
		{"wrong signing key", valid, otherKey.Parse},
		{"expired", expired, manager.Parse},
		{"empty", "", manager.Parse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if userID, err := tt.parse(tt.input); err == nil || userID != "" {
				t.Fatalf("Parse() = (%q, %v), want an error", userID, err)
			}
		})
	}
}
