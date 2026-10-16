package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ideaspark-api/internal/auth"
	"ideaspark-api/internal/models"
)

func TestOrderClause(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"", "created_at DESC, id DESC", true},
		{"newest", "created_at DESC, id DESC", true},
		{"oldest", "created_at ASC, id ASC", true},
		{"price_asc", "price ASC, created_at DESC, id DESC", true},
		{"price_desc", "price DESC, created_at DESC, id DESC", true},
		{"random", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := orderClause(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("orderClause(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestIntParam(t *testing.T) {
	tests := []struct {
		input         string
		def, min, max int
		want          int
	}{
		{"", 100, 1, 100, 100},
		{"bad", 100, 1, 100, 100},
		{"0", 100, 1, 100, 1},
		{"50", 100, 1, 100, 50},
		{"101", 100, 1, 100, 100},
		{"-3", 0, 0, 1_000_000, 0},
		{"2000000", 0, 0, 1_000_000, 1_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := intParam(tt.input, tt.def, tt.min, tt.max); got != tt.want {
				t.Fatalf("intParam() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestToIdeaDTO(t *testing.T) {
	created := time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC)
	in := models.Idea{ID: 3, SellerID: "seller-1", Title: "Title", Summary: "Summary", FullDetail: "Detail", Price: 250, CreatedAt: created}
	seller := sellerDTO{ID: "seller-1", Name: "Seller"}
	got := toIdeaDTO(in, seller)
	if got.ID != in.ID || got.SellerID != in.SellerID || got.Title != in.Title || got.Summary != in.Summary || got.FullDetail != in.FullDetail || got.Price != in.Price || !got.CreatedAt.Equal(created) || got.Seller != seller {
		t.Fatalf("toIdeaDTO() = %+v", got)
	}
}

func TestListIdeasRejectsUnsupportedSortWithoutDatabase(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/ideas?sort=unsupported", nil)
	rec := httptest.NewRecorder()
	h.ListIdeas(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestRoutesRejectInvalidRequestsWithoutDatabase(t *testing.T) {
	h := &Handler{Auth: auth.NewManager(strings.Repeat("s", 32), time.Hour)}
	routes := h.Routes(nil)
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantMsg    string
	}{
		{"register malformed JSON", http.MethodPost, "/auth/register", `{"name":`, http.StatusBadRequest, "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง"},
		{"register invalid fields", http.MethodPost, "/auth/register", `{"name":"x","email":"bad","password":"short"}`, http.StatusBadRequest, "ชื่อต้องมีความยาว 2-100 ตัวอักษร"},
		{"me without token", http.MethodGet, "/auth/me", "", http.StatusUnauthorized, "กรุณาเข้าสู่ระบบ"},
		{"create idea without token", http.MethodPost, "/ideas", `{}`, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบ"},
		{"idea with invalid id", http.MethodGet, "/ideas/not-an-id", "", http.StatusNotFound, "ไม่พบไอเดียนี้"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			routes.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var got map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("invalid JSON error response: %v; body = %s", err, rec.Body.String())
			}
			if got["message"] != tt.wantMsg {
				t.Fatalf("message = %q, want %q", got["message"], tt.wantMsg)
			}
		})
	}
}
