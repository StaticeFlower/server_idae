package httpx

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONWritesStatusContentTypeAndBody(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, 201, map[string]string{"status": "created"})

	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got["status"] != "created" {
		t.Fatalf("body = %q, error = %v", rec.Body.String(), err)
	}
}

func TestErrorUsesMessageShape(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, 400, "invalid input")
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["message"] != "invalid input" || len(got) != 1 {
		t.Fatalf("error body = %#v", got)
	}
}

func TestDecodeValidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Napat"}`))
	var got struct {
		Name string `json:"name"`
	}
	if err := Decode(httptest.NewRecorder(), req, &got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Name != "Napat" {
		t.Fatalf("decoded name = %q", got.Name)
	}
}

func TestDecodeRejectsMalformedJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":`))
	var got map[string]any
	if err := Decode(httptest.NewRecorder(), req, &got); err == nil {
		t.Fatal("Decode() accepted malformed JSON")
	}
}

func TestDecodeRejectsOversizedBody(t *testing.T) {
	body := `{"value":"` + strings.Repeat("x", maxBodyBytes) + `"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	var got map[string]any
	if err := Decode(httptest.NewRecorder(), req, &got); err == nil {
		t.Fatal("Decode() accepted a body larger than the limit")
	}
}
