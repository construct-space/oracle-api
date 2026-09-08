package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"construct/oracle/internal/database"
	"construct/oracle/internal/models"

	"gorm.io/gorm"
)

func acctDB() *gorm.DB { return database.Get(database.DBAccounts) }

func ListAccountUsers(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	q := r.URL.Query()

	query := acctDB().Model(&models.AccountUser{})
	if search := q.Get("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR first_name LIKE ? OR last_name LIKE ?", like, like, like, like)
	}
	switch q.Get("status") {
	case "active":    query = query.Where("suspended = ?", false)
	case "suspended": query = query.Where("suspended = ?", true)
	}
	switch q.Get("totp") {
	case "enabled":  query = query.Where("totp_enabled = ?", true)
	case "disabled": query = query.Where("totp_enabled = ?", false)
	}
	switch q.Get("developer") {
	case "yes": query = query.Where("developer_status <> ''")
	case "no":  query = query.Where("developer_status = '' OR developer_status IS NULL")
	}

	// CSV export ignores pagination — staff exports typically want the full
	// filtered set. Cap at 10k to protect memory; if we need more, stream.
	if q.Get("csv") == "1" {
		var users []models.AccountUser
		if err := query.Order("created_at DESC").Limit(10000).Find(&users).Error; err != nil {
			WriteJSON(w, 500, map[string]any{"error": "Failed to list users"}); return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="users.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"id", "uuid", "username", "email", "first_name", "last_name", "totp_enabled", "last_login", "created_at"})
		for _, u := range users {
			ll := ""
			if u.LastLogin != nil { ll = u.LastLogin.Format(time.RFC3339) }
			_ = cw.Write([]string{fmt.Sprintf("%d", u.ID), u.UUID, u.Username, u.Email, u.FirstName, u.LastName, fmt.Sprintf("%t", u.TOTPEnabled), ll, u.CreatedAt.Format(time.RFC3339)})
		}
		cw.Flush(); return
	}

	page := parsePositiveInt(q.Get("page"), 1)
	limit := parsePositiveInt(q.Get("limit"), 10)
	if limit > 100 { limit = 100 }

	var total int64
	if err := query.Count(&total).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to count users"}); return
	}

	var users []models.AccountUser
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&users).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to list users"}); return
	}
	WriteJSON(w, 200, map[string]any{
		"data":  users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func parsePositiveInt(s string, def int) int {
	if s == "" { return def }
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 { return def }
	return n
}

func GetAccountUser(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	id := r.PathValue("id")
	var user models.AccountUser
	if err := acctDB().First(&user, id).Error; err != nil { WriteJSON(w, 404, map[string]any{"error": "User not found"}); return }
	var sessions []models.AccountSession
	var passkeys []models.AccountPasskey
	var accessTokens []models.AccountAccessToken
	var refreshTokens []models.AccountRefreshToken
	var preferences []models.AccountPreference
	acctDB().Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&sessions)
	acctDB().Where("user_id = ?", user.ID).Find(&passkeys)
	acctDB().Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&accessTokens)
	acctDB().Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&refreshTokens)
	acctDB().Where("user_id = ?", user.ID).Find(&preferences)
	WriteJSON(w, 200, map[string]any{"data": user, "sessions": sessions, "passkeys": passkeys, "access_tokens": accessTokens, "refresh_tokens": refreshTokens, "preferences": preferences})
}

func ListAccountSessions(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	q := r.URL.Query(); query := acctDB().Model(&models.AccountSession{})
	if userID := q.Get("user_id"); userID != "" { query = query.Where("user_id = ?", userID) }
	if q.Get("active_only") == "1" { query = query.Where("expires_at > ?", time.Now()) }
	var sessions []models.AccountSession
	if err := query.Order("created_at DESC").Limit(200).Find(&sessions).Error; err != nil { WriteJSON(w, 500, map[string]any{"error": "Failed to list sessions"}); return }
	WriteJSON(w, 200, map[string]any{"data": sessions})
}

func ListAccountOAuthClients(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	var clients []models.AccountOAuthClient
	if err := acctDB().Order("created_at DESC").Find(&clients).Error; err != nil { WriteJSON(w, 500, map[string]any{"error": "Failed to list OAuth clients"}); return }
	WriteJSON(w, 200, map[string]any{"data": clients})
}

// GET /api/accounts/oauth-clients/{id}
// Client row + list of users who currently hold a non-revoked, unexpired
// access token via this client, plus active / total token counts. Used by the
// Oracle detail page.
func GetAccountOAuthClient(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	id := r.PathValue("id")
	var client models.AccountOAuthClient
	if err := acctDB().First(&client, id).Error; err != nil {
		WriteJSON(w, 404, map[string]any{"error": "OAuth client not found"})
		return
	}
	type userAuth struct{ UserID uint `json:"user_id"` }
	var authorizations []userAuth
	acctDB().Model(&models.AccountAccessToken{}).
		Select("DISTINCT user_id").
		Where("client_id = ? AND revoked = false AND expires_at > ?", client.ClientID, time.Now()).
		Scan(&authorizations)

	var activeTokens int64
	acctDB().Model(&models.AccountAccessToken{}).
		Where("client_id = ? AND revoked = false AND expires_at > ?", client.ClientID, time.Now()).
		Count(&activeTokens)

	var totalTokens int64
	acctDB().Model(&models.AccountAccessToken{}).
		Where("client_id = ?", client.ClientID).
		Count(&totalTokens)

	WriteJSON(w, 200, map[string]any{
		"data":           client,
		"authorizations": authorizations,
		"active_tokens":  activeTokens,
		"total_tokens":   totalTokens,
	})
}
