package models

import "time"

type AccountUser struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	UUID               string     `gorm:"size:36" json:"uuid"`
	FirstName          string     `gorm:"size:255" json:"first_name"`
	LastName           string     `gorm:"size:255" json:"last_name"`
	Username           string     `gorm:"size:255" json:"username"`
	Email              string     `gorm:"size:255" json:"email"`
	Phone              *string    `gorm:"size:255" json:"phone,omitempty"`
	TOTPEnabled        bool       `json:"totp_enabled"`
	Suspended          bool       `gorm:"default:false;index" json:"suspended"`
	MustChangePassword bool       `gorm:"default:false" json:"must_change_password"`
	DeveloperStatus    string     `gorm:"size:50" json:"developer_status"`
	LastLogin          *time.Time `json:"last_login,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
func (AccountUser) TableName() string { return "users" }

type AccountSession struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	UserAgent *string   `gorm:"type:text" json:"user_agent,omitempty"`
	IPAddress *string   `gorm:"size:255" json:"ip_address,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
func (AccountSession) TableName() string { return "sessions" }

type AccountPasskey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `json:"user_id"`
	Name       string     `gorm:"size:255" json:"name"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
func (AccountPasskey) TableName() string { return "passkeys" }

type AccountAccessToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ClientID  string    `gorm:"column:client_id" json:"client_id"`
	UserID    uint      `gorm:"column:user_id" json:"user_id"`
	Scope     string    `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}
func (AccountAccessToken) TableName() string { return "access_tokens" }

type AccountRefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"column:user_id" json:"user_id"`
	ClientID  string    `gorm:"column:client_id" json:"client_id"`
	Scope     string    `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}
func (AccountRefreshToken) TableName() string { return "refresh_tokens" }

type AccountPreference struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	Key       string    `gorm:"size:255" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
func (AccountPreference) TableName() string { return "preferences" }

type AccountOAuthClient struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ClientID    string    `gorm:"column:client_id" json:"client_id"`
	Name        string    `json:"name"`
	RedirectURI string    `gorm:"column:redirect_uri" json:"redirect_uri"`
	Description *string   `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}
func (AccountOAuthClient) TableName() string { return "oauth_clients" }
