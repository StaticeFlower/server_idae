// Package models คือ GORM models ที่ตรงกับ ERD (ตาราง users และ ideas)
package models

import "time"

type User struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name         string    `gorm:"type:text;not null" json:"name"`
	Email        string    `gorm:"type:text;not null;uniqueIndex:users_email_key" json:"email"`
	PasswordHash string    `gorm:"type:text;not null" json:"-"` // ไม่ส่งออกทาง JSON เด็ดขาด
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
}

type Idea struct {
	ID         int       `gorm:"primaryKey" json:"id"`
	SellerID   string    `gorm:"type:uuid;not null;index" json:"seller_id"`
	Title      string    `gorm:"type:text;not null" json:"title"`
	Summary    string    `gorm:"type:text;not null" json:"summary"`
	FullDetail string    `gorm:"type:text;not null" json:"full_detail"`
	Price      int       `gorm:"type:integer;not null;default:0" json:"price"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`

	Seller User `gorm:"foreignKey:SellerID;constraint:OnDelete:RESTRICT" json:"-"`
}
