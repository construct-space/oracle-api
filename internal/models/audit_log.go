package models

import "time"

// AuditLog records a single privileged action taken through Oracle: who did
// what, against which resource, when, and from where. Written by handlers via
// handlers.logAudit on any successful mutation (2xx response). Metadata is a
// free-form JSON blob for anything the specific action wants to record (new
// role, reject reason, etc.) so schema evolution stays loose.
type AuditLog struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	ActorAdministratorID *uint    `gorm:"index" json:"actor_administrator_id,omitempty"`
	ActorEmail         string     `gorm:"size:255" json:"actor_email"`
	Action             string     `gorm:"size:100;index;not null" json:"action"`
	ResourceType       string     `gorm:"size:50;index" json:"resource_type"`
	ResourceID         string     `gorm:"size:100;index" json:"resource_id"`
	Metadata           *string    `gorm:"type:text" json:"metadata,omitempty"`
	IPAddress          string     `gorm:"size:45" json:"ip_address"`
	UserAgent          string     `gorm:"type:text" json:"user_agent"`
	CreatedAt          time.Time  `gorm:"index" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_log" }
