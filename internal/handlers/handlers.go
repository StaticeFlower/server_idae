// Package handlers คือ HTTP handlers และ routes ของ API
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"

	"ideaspark-api/internal/auth"
	"ideaspark-api/internal/httpx"
	"ideaspark-api/internal/middleware"
)

type Handler struct {
	DB   *gorm.DB
	Auth *auth.Manager
}

// Routes ประกอบ router พร้อม middleware ทั้งหมด
func (h *Handler) Routes(corsOrigins []string) http.Handler {
	mux := http.NewServeMux()
	requireAuth := middleware.RequireAuth(h.Auth)

	mux.HandleFunc("GET /healthz", h.Health)

	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.Handle("GET /auth/me", requireAuth(http.HandlerFunc(h.Me)))

	mux.HandleFunc("GET /ideas", h.ListIdeas)
	mux.HandleFunc("GET /ideas/{id}", h.GetIdea)
	mux.Handle("POST /ideas", requireAuth(http.HandlerFunc(h.CreateIdea)))

	return middleware.Chain(mux,
		middleware.Logging,
		middleware.Recover,
		middleware.CORS(corsOrigins),
	)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := h.DB.DB()
	if err == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		err = sqlDB.PingContext(ctx)
	}
	if err != nil {
		slog.Error("health check failed", "error", err)
		httpx.Error(w, http.StatusServiceUnavailable, "ฐานข้อมูลไม่พร้อมใช้งาน")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- helpers ----

func serverError(w http.ResponseWriter, op string, err error) {
	slog.Error("request failed", "op", op, "error", err)
	httpx.Error(w, http.StatusInternalServerError, "เกิดข้อผิดพลาดภายในระบบ")
}

func badBody(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		httpx.Error(w, http.StatusRequestEntityTooLarge, "ข้อมูลที่ส่งมามีขนาดใหญ่เกินไป")
		return
	}
	httpx.Error(w, http.StatusBadRequest, "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง")
}
