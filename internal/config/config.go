// Package config อ่านค่าตั้งค่าจาก environment variables
package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	CORSOrigins []string
	AutoMigrate bool
}

const minJWTSecretLen = 32

func Load() (*Config, error) {
	cfg := &Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTL:      7 * 24 * time.Hour,
		CORSOrigins: splitCSV(getenv("CORS_ORIGINS", "http://localhost:3000")),
		AutoMigrate: strings.EqualFold(os.Getenv("AUTO_MIGRATE"), "true"),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < minJWTSecretLen {
		return nil, errors.New("JWT_SECRET is required and must be at least 32 characters")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, strings.TrimRight(p, "/"))
		}
	}
	return out
}
