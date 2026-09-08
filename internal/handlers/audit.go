package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"construct/oracle/internal/database"
	"construct/oracle/internal/models"
)

// statusRecorder wraps http.ResponseWriter so a proxy handler can record the
// downstream status code without the caller losing access to it. Used by
// audit-wrapped proxy handlers: write through as normal, then inspect
// rec.status before deciding whether to write an audit row.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// recordStatus wraps w and returns a *statusRecorder with an initial "assume
// 200" status — http.ResponseWriter's zero-value is 200 if Write is called
// without an explicit WriteHeader, so start there.
func recordStatus(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, status: 200}
}

// logAudit writes one audit row. Safe to call from any handler — silent on DB
// failure so an audit outage never blocks the actual operation.
// metadata can be nil; if non-nil it's serialized to JSON.
func logAudit(r *http.Request, session *models.Session, action, resourceType, resourceID string, metadata map[string]any) {
	entry := models.AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    clientIP(r),
		UserAgent:    r.UserAgent(),
		CreatedAt:    time.Now(),
	}
	if session != nil {
		entry.ActorAdministratorID = session.AdministratorID
		if session.Email != nil {
			entry.ActorEmail = *session.Email
		}
	}
	if len(metadata) > 0 {
		if b, err := json.Marshal(metadata); err == nil {
			s := string(b)
			entry.Metadata = &s
		}
	}
	// Fire-and-forget: auditing is important but must never wedge a request.
	e := entry
	safeAsync("oracle.audit", func() {
		_ = database.Get(database.DBOracle).Create(&e).Error
	})
}

// clientIP returns the best-effort client IP. Prefer X-Forwarded-For (first
// entry) since Oracle sits behind an nginx gateway; fall back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	// RemoteAddr is host:port — strip port.
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > 0 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// ListAuditLog — GET /api/audit-log
// Filters: actor_id, action, resource_type, resource_id, page, limit.
// Requires any authenticated admin (viewing the log is a safe read).
func ListAuditLog(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	q := r.URL.Query()
	page := parseIntOr(q.Get("page"), 1)
	limit := parseIntOr(q.Get("limit"), 20)
	if limit > 200 {
		limit = 200
	}

	db := database.Get(database.DBOracle).Model(&models.AuditLog{})
	if v := q.Get("actor_id"); v != "" {
		db = db.Where("actor_administrator_id = ?", v)
	}
	if v := q.Get("action"); v != "" {
		db = db.Where("action = ?", v)
	}
	if v := q.Get("resource_type"); v != "" {
		db = db.Where("resource_type = ?", v)
	}
	if v := q.Get("resource_id"); v != "" {
		db = db.Where("resource_id = ?", v)
	}

	var total int64
	db.Count(&total)

	var entries []models.AuditLog
	if err := db.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&entries).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "failed to list audit log"})
		return
	}
	WriteJSON(w, 200, map[string]any{
		"entries": entries,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

func parseIntOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	return n
}
