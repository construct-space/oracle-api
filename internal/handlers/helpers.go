package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"construct/oracle/internal/config"
	"construct/oracle/internal/database"
	"construct/oracle/internal/models"
)

var Cfg *config.Config

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func parseBody(r *http.Request) (map[string]any, error) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { return nil, err }
	return body, nil
}

func parseCookie(cookieHeader, name string) string {
	for _, part := range strings.Split(cookieHeader, ";") {
		trimmed := strings.TrimSpace(part)
		if idx := strings.Index(trimmed, "="); idx > 0 && trimmed[:idx] == name {
			return trimmed[idx+1:]
		}
	}
	return ""
}

func getSession(r *http.Request) *models.Session {
	token := parseCookie(r.Header.Get("Cookie"), "session")
	if token == "" { return nil }
	var session models.Session
	if err := database.Get(database.DBOracle).Where("token = ?", token).First(&session).Error; err != nil { return nil }
	if session.IsExpired() { return nil }
	return &session
}

func requireAuth(w http.ResponseWriter, r *http.Request) *models.Session {
	s := getSession(r)
	if s == nil {
		WriteJSON(w, 401, map[string]any{"error": "Unauthorized"})
		return nil
	}
	return s
}

func requireRole(w http.ResponseWriter, r *http.Request, roles ...string) *models.Session {
	s := requireAuth(w, r)
	if s == nil { return nil }
	role := ""
	if s.Role != nil { role = *s.Role }
	for _, allowed := range roles { if role == allowed { return s } }
	WriteJSON(w, 403, map[string]any{"error": "Forbidden"})
	return nil
}

func requireAdmin(w http.ResponseWriter, r *http.Request) *models.Session { return requireRole(w, r, "admin", "super_admin") }
func requireSuperAdmin(w http.ResponseWriter, r *http.Request) *models.Session { return requireRole(w, r, "super_admin") }

func strPtr(s string) *string { if s == "" { return nil }; return &s }
func stringFromBody(body map[string]any, key string) string {
	if v, ok := body[key].(string); ok { return v }
	return ""
}
