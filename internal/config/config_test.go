package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoadRequiresDatabaseURLAndSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	if _, err := Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL is missing")
	}

	t.Setenv("DATABASE_URL", "postgresql://x")
	t.Setenv("JWT_SECRET", "too-short")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when JWT_SECRET is too short")
	}
}

func TestLoadDefaultsAndParsing(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://x")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("PORT", "")
	t.Setenv("CORS_ORIGINS", " http://localhost:3000/ , https://app.example.com ,")
	t.Setenv("AUTO_MIGRATE", "TRUE")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q", cfg.Port)
	}
	want := []string{"http://localhost:3000", "https://app.example.com"}
	if !reflect.DeepEqual(cfg.CORSOrigins, want) {
		t.Fatalf("CORSOrigins = %v, want %v", cfg.CORSOrigins, want)
	}
	if !cfg.AutoMigrate {
		t.Fatal("AutoMigrate should be true")
	}
}
