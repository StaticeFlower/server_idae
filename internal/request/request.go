// Package request คือ request body ของแต่ละ endpoint พร้อมการตรวจสอบข้อมูล
// กฎต้องสอดคล้องกับฟอร์มฝั่งหน้าเว็บ (ฝั่งเซิร์ฟเวอร์ต้องตรวจซ้ำเสมอ)
package request

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	maxEmailLen      = 254
	maxPasswordBytes = 72 // ขีดจำกัดของ bcrypt
	MaxPrice         = 100_000_000
)

var (
	ErrName          = errors.New("ชื่อต้องมีความยาว 2-100 ตัวอักษร")
	ErrEmail         = errors.New("รูปแบบอีเมลไม่ถูกต้อง")
	ErrPasswordEmpty = errors.New("กรุณากรอกรหัสผ่าน")
	ErrPasswordShort = errors.New("รหัสผ่านต้องมีอย่างน้อย 8 ตัวอักษร")
	ErrPasswordLong  = errors.New("รหัสผ่านยาวเกินไป (ไม่เกิน 72 ไบต์)")
	ErrTitle         = errors.New("ชื่อไอเดียต้องมีความยาว 5-120 ตัวอักษร")
	ErrSummary       = errors.New("สรุปสั้น ๆ ต้องมีความยาว 10-300 ตัวอักษร")
	ErrFullDetail    = errors.New("รายละเอียดเต็มต้องมีความยาว 30-20,000 ตัวอักษร")
	ErrPrice         = errors.New("ราคาต้องเป็นจำนวนเต็มตั้งแต่ 0 ถึง 100,000,000")
)

type Register struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate ตรวจสอบและ normalize ข้อมูล (ตัดช่องว่างชื่อ, อีเมลเป็นตัวพิมพ์เล็ก)
func (r *Register) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if !runeLenBetween(r.Name, 2, 100) {
		return ErrName
	}
	email, err := normalizeEmail(r.Email)
	if err != nil {
		return err
	}
	r.Email = email

	if utf8.RuneCountInString(r.Password) < 8 {
		return ErrPasswordShort
	}
	if len(r.Password) > maxPasswordBytes {
		return ErrPasswordLong
	}
	return nil
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (l *Login) Validate() error {
	email, err := normalizeEmail(l.Email)
	if err != nil {
		return err
	}
	l.Email = email

	if l.Password == "" {
		return ErrPasswordEmpty
	}
	if len(l.Password) > maxPasswordBytes {
		return ErrPasswordLong
	}
	return nil
}

type CreateIdea struct {
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	FullDetail string `json:"full_detail"`
	Price      int    `json:"price"`
}

func (c *CreateIdea) Validate() error {
	c.Title = strings.TrimSpace(c.Title)
	c.Summary = strings.TrimSpace(c.Summary)
	c.FullDetail = strings.TrimSpace(c.FullDetail)

	switch {
	case !runeLenBetween(c.Title, 5, 120):
		return ErrTitle
	case !runeLenBetween(c.Summary, 10, 300):
		return ErrSummary
	case !runeLenBetween(c.FullDetail, 30, 20000):
		return ErrFullDetail
	case c.Price < 0 || c.Price > MaxPrice:
		return ErrPrice
	}
	return nil
}

func normalizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || len(s) > maxEmailLen {
		return "", ErrEmail
	}
	addr, err := mail.ParseAddress(s)
	// ต้องเป็นอีเมลล้วน ๆ ไม่ใช่รูปแบบ "ชื่อ <email>"
	if err != nil || addr.Address != s {
		return "", ErrEmail
	}
	at := strings.LastIndex(s, "@")
	if at < 1 || !strings.Contains(s[at+1:], ".") {
		return "", ErrEmail
	}
	return s, nil
}

func runeLenBetween(s string, min, max int) bool {
	n := utf8.RuneCountInString(s)
	return n >= min && n <= max
}
