package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Session struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Token           string     `gorm:"size:128;uniqueIndex;not null" json:"-"`
	Email           *string    `gorm:"size:200" json:"email,omitempty"`
	Name            *string    `gorm:"size:200" json:"name,omitempty"`
	Role            *string    `gorm:"size:20" json:"role,omitempty"`
	AdministratorID *uint      `gorm:"index" json:"administrator_id,omitempty"`
	UserAgent       *string    `gorm:"type:text" json:"user_agent,omitempty"`
	IPAddress       *string    `gorm:"size:255" json:"ip_address,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
}

func (Session) TableName() string { return "admin_sessions" }

func (s *Session) IsExpired() bool {
	return s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt)
}

func GenerateSessionToken() string {
	b := make([]byte, 48)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
