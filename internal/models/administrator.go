package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Administrator struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Email        string     `gorm:"size:200;uniqueIndex;not null" json:"email"`
	FirstName    string     `gorm:"size:100;not null" json:"first_name"`
	LastName     string     `gorm:"size:100;not null" json:"last_name"`
	Username     string     `gorm:"size:100;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Role         string     `gorm:"size:20;default:admin;not null" json:"role"`
	Active       bool       `gorm:"default:true;not null" json:"active"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
}

func (Administrator) TableName() string { return "administrators" }

func (a *Administrator) HashPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.PasswordHash = string(hash)
	return nil
}

func (a *Administrator) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) == nil
}
