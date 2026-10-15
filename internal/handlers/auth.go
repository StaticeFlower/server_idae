package handlers

import (
	"errors"
	"net/http"

	"gorm.io/gorm"

	"ideaspark-api/internal/auth"
	"ideaspark-api/internal/httpx"
	"ideaspark-api/internal/middleware"
	"ideaspark-api/internal/models"
	"ideaspark-api/internal/request"
)

// sessionResponse ตรงกับ type AuthSession ฝั่งหน้าเว็บ: { token, user }
type sessionResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func (h *Handler) respondSession(w http.ResponseWriter, status int, user models.User) {
	token, err := h.Auth.Issue(user.ID)
	if err != nil {
		serverError(w, "issue token", err)
		return
	}
	httpx.JSON(w, status, sessionResponse{Token: token, User: user})
}

// POST /auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in request.Register
	if err := httpx.Decode(w, r, &in); err != nil {
		badBody(w, err)
		return
	}
	if err := in.Validate(); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		serverError(w, "hash password", err)
		return
	}

	user := models.User{Name: in.Name, Email: in.Email, PasswordHash: hash}
	if err := h.DB.WithContext(r.Context()).Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			httpx.Error(w, http.StatusConflict, "อีเมลนี้ถูกใช้งานแล้ว")
			return
		}
		serverError(w, "create user", err)
		return
	}
	h.respondSession(w, http.StatusCreated, user)
}

// POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in request.Login
	if err := httpx.Decode(w, r, &in); err != nil {
		badBody(w, err)
		return
	}
	if err := in.Validate(); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	const invalidCreds = "อีเมลหรือรหัสผ่านไม่ถูกต้อง"

	var user models.User
	err := h.DB.WithContext(r.Context()).Where("email = ?", in.Email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		auth.BurnCompare(in.Password)
		httpx.Error(w, http.StatusUnauthorized, invalidCreds)
		return
	}
	if err != nil {
		serverError(w, "find user", err)
		return
	}
	if !auth.CheckPassword(user.PasswordHash, in.Password) {
		httpx.Error(w, http.StatusUnauthorized, invalidCreds)
		return
	}
	h.respondSession(w, http.StatusOK, user)
}

// GET /auth/me  (ต้องมี Bearer token)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบ")
		return
	}

	var user models.User
	err := h.DB.WithContext(r.Context()).First(&user, "id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		httpx.Error(w, http.StatusUnauthorized, "เซสชันไม่ถูกต้องหรือหมดอายุ")
		return
	}
	if err != nil {
		serverError(w, "find me", err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}
