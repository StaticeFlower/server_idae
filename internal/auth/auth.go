// Package auth จัดการรหัสผ่าน (bcrypt) และ JWT
package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 12
	issuer     = "ideaspark-api"
)

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

var (
	dummyOnce sync.Once
	dummyHash string
)

// BurnCompare ใช้เวลาเทียบเท่าการตรวจรหัสผ่านจริง เรียกเมื่อไม่พบอีเมลในระบบ
// เพื่อไม่ให้เวลาตอบสนองเผยว่าอีเมลนั้นมีบัญชีอยู่หรือไม่
func BurnCompare(password string) {
	dummyOnce.Do(func() {
		dummyHash, _ = HashPassword("timing-equalizer-password")
	})
	_ = CheckPassword(dummyHash, password)
}

// Manager ออกและตรวจสอบ JWT (HS256)
type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Issue(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse ตรวจ token แล้วคืน user id (subject)
func (m *Manager) Parse(tokenString string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(issuer),
	)
	if err != nil {
		return "", err
	}
	if !token.Valid || claims.Subject == "" {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}
