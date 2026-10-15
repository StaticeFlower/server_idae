// Package middleware รวม middleware: logging, recover, CORS, ตรวจ token
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"ideaspark-api/internal/httpx"
)

type Middleware func(http.Handler) http.Handler

// Chain ครอบ handler ด้วย middleware โดยตัวแรกในลิสต์อยู่ชั้นนอกสุด
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// ---- Logging ----

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path, // ไม่ log query string เผื่อมีข้อมูลที่ไม่ควรเก็บ
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// ---- Recover ----

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				slog.Error("panic", "error", rec, "stack", string(debug.Stack()))
				httpx.Error(w, http.StatusInternalServerError, "เกิดข้อผิดพลาดภายในระบบ")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ---- CORS ----

// CORS อนุญาตเฉพาะ origin ที่ระบุ (ไม่ใช้ wildcard) และตอบ preflight เอง
func CORS(allowedOrigins []string) Middleware {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" {
				w.Header().Add("Vary", "Origin")
				if _, ok := allowed[origin]; ok {
					h := w.Header()
					h.Set("Access-Control-Allow-Origin", origin)
					h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
					h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					h.Set("Access-Control-Max-Age", "600")
				}
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ---- Auth ----

// TokenParser ตรวจ token แล้วคืน user id (ใช้ interface เพื่อไม่ผูกกับ package auth)
type TokenParser interface {
	Parse(token string) (userID string, err error)
}

type userIDKey struct{}

func RequireAuth(p TokenParser) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
			token = strings.TrimSpace(token)
			if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
				httpx.Error(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบ")
				return
			}
			userID, err := p.Parse(token)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "เซสชันไม่ถูกต้องหรือหมดอายุ")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID ดึง user id ที่ RequireAuth ใส่ไว้ใน context
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey{}).(string)
	return id, ok && id != ""
}
