package request

import (
	"errors"
	"strings"
	"testing"
)

func TestRegisterValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      Register
		wantErr error
	}{
		{"ok", Register{"นภัส ศรีสุข", "Napat@Example.com ", "password123"}, nil},
		{"name too short", Register{"a", "a@b.co", "password123"}, ErrName},
		{"name blank", Register{"   ", "a@b.co", "password123"}, ErrName},
		{"email invalid", Register{"สมชาย", "not-an-email", "password123"}, ErrEmail},
		{"email display name form", Register{"สมชาย", "Bob <bob@example.com>", "password123"}, ErrEmail},
		{"email no dot in domain", Register{"สมชาย", "bob@localhost", "password123"}, ErrEmail},
		{"password short", Register{"สมชาย", "a@b.co", "short"}, ErrPasswordShort},
		{"password over 72 bytes", Register{"สมชาย", "a@b.co", strings.Repeat("a", 73)}, ErrPasswordLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegisterNormalizes(t *testing.T) {
	r := Register{Name: "  นภัส  ", Email: "  Napat@Example.COM ", Password: "password123"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if r.Name != "นภัส" || r.Email != "napat@example.com" {
		t.Fatalf("not normalized: %+v", r)
	}
}

func TestLoginValidate(t *testing.T) {
	if err := (&Login{Email: "a@b.co", Password: "x"}).Validate(); err != nil {
		t.Fatalf("short password must be allowed at login, got %v", err)
	}
	if err := (&Login{Email: "a@b.co", Password: ""}).Validate(); !errors.Is(err, ErrPasswordEmpty) {
		t.Fatalf("got %v", err)
	}
	if err := (&Login{Email: "bad", Password: "x"}).Validate(); !errors.Is(err, ErrEmail) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateIdeaValidate(t *testing.T) {
	good := func() CreateIdea {
		return CreateIdea{
			Title:      "แอปแบ่งบิลกลุ่มเพื่อน",
			Summary:    "แชทบอทช่วยหารค่าอาหารแบบเรียลไทม์",
			FullDetail: strings.Repeat("รายละเอียด", 10),
			Price:      0,
		}
	}
	tests := []struct {
		name    string
		mutate  func(*CreateIdea)
		wantErr error
	}{
		{"ok", func(*CreateIdea) {}, nil},
		{"ok with price", func(c *CreateIdea) { c.Price = 1500 }, nil},
		{"title too short", func(c *CreateIdea) { c.Title = "abc" }, ErrTitle},
		{"title too long", func(c *CreateIdea) { c.Title = strings.Repeat("ก", 121) }, ErrTitle},
		{"summary too short", func(c *CreateIdea) { c.Summary = "สั้น" }, ErrSummary},
		{"summary too long", func(c *CreateIdea) { c.Summary = strings.Repeat("ก", 301) }, ErrSummary},
		{"detail too short", func(c *CreateIdea) { c.FullDetail = "สั้นไป" }, ErrFullDetail},
		{"negative price", func(c *CreateIdea) { c.Price = -1 }, ErrPrice},
		{"price too large", func(c *CreateIdea) { c.Price = MaxPrice + 1 }, ErrPrice},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := good()
			tt.mutate(&c)
			if err := c.Validate(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// ความยาวนับเป็นจำนวนตัวอักษร (rune) ไม่ใช่ไบต์ — ภาษาไทย 1 ตัวอักษร = 3 ไบต์
func TestLengthCountsRunesNotBytes(t *testing.T) {
	c := CreateIdea{
		Title:      strings.Repeat("ก", 120), // 120 ตัวอักษร = 360 ไบต์ ต้องผ่าน
		Summary:    strings.Repeat("ก", 300),
		FullDetail: strings.Repeat("ก", 30),
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Thai text at the limit must be accepted, got %v", err)
	}
}
