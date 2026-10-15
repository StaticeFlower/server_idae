package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestCORSAllowedOrigin(t *testing.T) {
	h := CORS([]string{"http://localhost:3000/"})(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/ideas", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Allow-Origin = %q", got)
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	h := CORS([]string{"http://localhost:3000"})(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/ideas", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("must not allow origin, got %q", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	h := CORS([]string{"http://localhost:3000"})(next)

	req := httptest.NewRequest(http.MethodOptions, "/ideas", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if called {
		t.Fatal("preflight must not reach the next handler")
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatal("Authorization header must be allowed")
	}
}

type fakeParser struct{}

func (fakeParser) Parse(token string) (string, error) {
	if token == "good" {
		return "user-1", nil
	}
	return "", errors.New("bad token")
}

func TestRequireAuth(t *testing.T) {
	var gotUser string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _ = UserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := RequireAuth(fakeParser{})(next)

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic good", http.StatusUnauthorized},
		{"empty token", "Bearer ", http.StatusUnauthorized},
		{"bad token", "Bearer nope", http.StatusUnauthorized},
		{"good token", "Bearer good", http.StatusOK},
		{"scheme is case-insensitive", "bearer good", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser = ""
			req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && gotUser != "user-1" {
				t.Fatalf("user id in context = %q", gotUser)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	h := Recover(boom)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"message"`) {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestChainOrder(t *testing.T) {
	var order []string
	mk := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	h := Chain(okHandler, mk("first"), mk("second"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if strings.Join(order, ",") != "first,second" {
		t.Fatalf("order = %v", order)
	}
}
