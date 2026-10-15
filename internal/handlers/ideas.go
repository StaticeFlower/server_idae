package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ideaspark-api/internal/httpx"
	"ideaspark-api/internal/middleware"
	"ideaspark-api/internal/models"
	"ideaspark-api/internal/request"
)

const (
	defaultLimit   = 100
	maxLimit       = 100
	maxSearchRunes = 100
)

// DTO ตรงกับ type IdeaWithSeller ฝั่งหน้าเว็บ (ฟิลด์ของ idea + seller {id, name})
type sellerDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ideaDTO struct {
	ID         int       `json:"id"`
	SellerID   string    `json:"seller_id"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	FullDetail string    `json:"full_detail"`
	Price      int       `json:"price"`
	CreatedAt  time.Time `json:"created_at"`
	Seller     sellerDTO `json:"seller"`
}

func toIdeaDTO(i models.Idea, seller sellerDTO) ideaDTO {
	return ideaDTO{
		ID:         i.ID,
		SellerID:   i.SellerID,
		Title:      i.Title,
		Summary:    i.Summary,
		FullDetail: i.FullDetail,
		Price:      i.Price,
		CreatedAt:  i.CreatedAt,
		Seller:     seller,
	}
}

// preloadSeller โหลดเฉพาะ id และ name ของผู้ขาย (ไม่ดึง email / password_hash)
func preloadSeller(db *gorm.DB) *gorm.DB {
	return db.Select("id", "name")
}

func orderClause(sort string) (string, bool) {
	switch sort {
	case "", "newest":
		return "created_at DESC, id DESC", true
	case "oldest":
		return "created_at ASC, id ASC", true
	case "price_asc":
		return "price ASC, created_at DESC, id DESC", true
	case "price_desc":
		return "price DESC, created_at DESC, id DESC", true
	}
	return "", false
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func intParam(s string, def, min, max int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// GET /ideas?q=&sort=&limit=&offset=
func (h *Handler) ListIdeas(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	order, ok := orderClause(query.Get("sort"))
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "ค่า sort ไม่ถูกต้อง")
		return
	}
	limit := intParam(query.Get("limit"), defaultLimit, 1, maxLimit)
	offset := intParam(query.Get("offset"), 0, 0, 1_000_000)

	q := strings.TrimSpace(query.Get("q"))
	if utf8.RuneCountInString(q) > maxSearchRunes {
		q = string([]rune(q)[:maxSearchRunes])
	}

	tx := h.DB.WithContext(r.Context()).Model(&models.Idea{})
	if q != "" {
		like := "%" + likeEscaper.Replace(q) + "%"
		tx = tx.Where("title ILIKE ? OR summary ILIKE ?", like, like)
	}

	var ideas []models.Idea
	err := tx.Preload("Seller", preloadSeller).
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&ideas).Error
	if err != nil {
		serverError(w, "list ideas", err)
		return
	}

	// ต้องเป็น [] ไม่ใช่ null เมื่อไม่มีข้อมูล
	out := make([]ideaDTO, 0, len(ideas))
	for _, i := range ideas {
		out = append(out, toIdeaDTO(i, sellerDTO{ID: i.Seller.ID, Name: i.Seller.Name}))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// GET /ideas/{id}
func (h *Handler) GetIdea(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		httpx.Error(w, http.StatusNotFound, "ไม่พบไอเดียนี้")
		return
	}

	var idea models.Idea
	err = h.DB.WithContext(r.Context()).
		Preload("Seller", preloadSeller).
		First(&idea, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		httpx.Error(w, http.StatusNotFound, "ไม่พบไอเดียนี้")
		return
	}
	if err != nil {
		serverError(w, "get idea", err)
		return
	}
	httpx.JSON(w, http.StatusOK, toIdeaDTO(idea, sellerDTO{ID: idea.Seller.ID, Name: idea.Seller.Name}))
}

// POST /ideas  (ต้องมี Bearer token) — seller_id มาจาก token เท่านั้น ไม่รับจาก body
func (h *Handler) CreateIdea(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบ")
		return
	}

	var in request.CreateIdea
	if err := httpx.Decode(w, r, &in); err != nil {
		badBody(w, err)
		return
	}
	if err := in.Validate(); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	db := h.DB.WithContext(r.Context())

	// ยืนยันว่าผู้ใช้ยังมีอยู่ และเอาชื่อมาใส่ในผลลัพธ์
	var seller models.User
	err := db.Select("id", "name").First(&seller, "id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		httpx.Error(w, http.StatusUnauthorized, "เซสชันไม่ถูกต้องหรือหมดอายุ")
		return
	}
	if err != nil {
		serverError(w, "find seller", err)
		return
	}

	idea := models.Idea{
		SellerID:   seller.ID,
		Title:      in.Title,
		Summary:    in.Summary,
		FullDetail: in.FullDetail,
		Price:      in.Price,
	}
	// Omit associations: กัน GORM พยายามบันทึกข้อมูลผู้ขายซ้ำไปด้วย
	if err := db.Omit(clause.Associations).Create(&idea).Error; err != nil {
		serverError(w, "create idea", err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toIdeaDTO(idea, sellerDTO{ID: seller.ID, Name: seller.Name}))
}
