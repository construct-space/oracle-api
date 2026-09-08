package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"construct/oracle/internal/database"
	"construct/oracle/internal/mailer"
	"construct/oracle/internal/models"
)

func ListAdministrators(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	var admins []models.Administrator
	if err := database.Get(database.DBOracle).Order("created_at DESC").Find(&admins).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to list administrators"})
		return
	}
	WriteJSON(w, 200, map[string]any{"data": admins})
}

func CreateAdministrator(w http.ResponseWriter, r *http.Request) {
	session := requireSuperAdmin(w, r)
	if session == nil {
		return
	}
	body, err := parseBody(r)
	if err != nil {
		WriteJSON(w, 400, map[string]any{"error": "Invalid request body"})
		return
	}
	now := time.Now()
	admin := models.Administrator{Email: stringFromBody(body, "email"), FirstName: stringFromBody(body, "first_name"), LastName: stringFromBody(body, "last_name"), Username: stringFromBody(body, "username"), Role: stringFromBody(body, "role"), Active: true, CreatedAt: &now}
	if admin.Role == "" {
		admin.Role = "admin"
	}
	password := stringFromBody(body, "password")
	if password == "" {
		WriteJSON(w, 400, map[string]any{"error": "Password is required"})
		return
	}
	if err := admin.HashPassword(password); err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to hash password"})
		return
	}
	if err := database.Get(database.DBOracle).Create(&admin).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to create administrator"})
		return
	}
	logAudit(r, session, "administrator.create", "administrator", fmt.Sprint(admin.ID), map[string]any{"email": admin.Email, "role": admin.Role})

	// Welcome email carries the plaintext password the super-admin chose.
	// Fire-and-forget + log — the admin account has been created regardless;
	// we don't want to 500 the create response because mail is down.
	to, first, user, pw, appURL := admin.Email, admin.FirstName, admin.Username, password, Cfg.AppURL
	safeAsync("mailer.admin_welcome", func() {
		if err := mailer.SendAdminWelcome(Cfg, to, first, user, pw, appURL); err != nil {
			log.Printf("[administrators] welcome email to %s failed: %v", to, err)
		}
	})

	WriteJSON(w, 201, map[string]any{"data": admin, "email_sent": true})
}

// PATCH /api/administrators/{id}
// Body: any subset of {email, first_name, last_name, username, role, active}
// Super-admin only. A super-admin cannot demote or disable themselves — that's
// a lockout waiting to happen; password-reset / delete have to go through
// another super-admin.
func UpdateAdministrator(w http.ResponseWriter, r *http.Request) {
	session := requireSuperAdmin(w, r)
	if session == nil {
		return
	}
	id := r.PathValue("id")

	var admin models.Administrator
	if err := database.Get(database.DBOracle).First(&admin, id).Error; err != nil {
		WriteJSON(w, 404, map[string]any{"error": "Administrator not found"})
		return
	}

	var body struct {
		Email     *string `json:"email"`
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Username  *string `json:"username"`
		Role      *string `json:"role"`
		Active    *bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteJSON(w, 400, map[string]any{"error": "Invalid request body"})
		return
	}

	isSelf := session.AdministratorID != nil && *session.AdministratorID == admin.ID
	updates := map[string]any{}
	if body.Email != nil {
		v := strings.TrimSpace(*body.Email)
		if v == "" {
			WriteJSON(w, 400, map[string]any{"error": "email cannot be empty"})
			return
		}
		updates["email"] = v
	}
	if body.FirstName != nil {
		updates["first_name"] = *body.FirstName
	}
	if body.LastName != nil {
		updates["last_name"] = *body.LastName
	}
	if body.Username != nil {
		updates["username"] = *body.Username
	}
	if body.Role != nil {
		if isSelf && *body.Role != admin.Role {
			WriteJSON(w, 400, map[string]any{"error": "Cannot change your own role"})
			return
		}
		updates["role"] = *body.Role
	}
	if body.Active != nil {
		if isSelf && !*body.Active {
			WriteJSON(w, 400, map[string]any{"error": "Cannot disable your own account"})
			return
		}
		updates["active"] = *body.Active
	}

	if len(updates) == 0 {
		WriteJSON(w, 400, map[string]any{"error": "no updatable fields provided"})
		return
	}
	if err := database.Get(database.DBOracle).Model(&admin).Updates(updates).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to update administrator"})
		return
	}
	database.Get(database.DBOracle).First(&admin, id)
	logAudit(r, session, "administrator.update", "administrator", id, updates)
	WriteJSON(w, 200, map[string]any{"data": admin})
}

// DELETE /api/administrators/{id}
// Super-admin only. Self-deletion is blocked so a lone super-admin can't
// strand the Oracle.
func DeleteAdministrator(w http.ResponseWriter, r *http.Request) {
	session := requireSuperAdmin(w, r)
	if session == nil {
		return
	}
	id := r.PathValue("id")

	var admin models.Administrator
	if err := database.Get(database.DBOracle).First(&admin, id).Error; err != nil {
		WriteJSON(w, 404, map[string]any{"error": "Administrator not found"})
		return
	}
	if session.AdministratorID != nil && *session.AdministratorID == admin.ID {
		WriteJSON(w, 400, map[string]any{"error": "Cannot delete your own account"})
		return
	}
	// Nuke any active sessions attached to this administrator so deletion
	// fully invalidates access, not just the row.
	database.Get(database.DBOracle).Where("administrator_id = ?", admin.ID).Delete(&models.Session{})
	if err := database.Get(database.DBOracle).Delete(&admin).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to delete administrator"})
		return
	}
	logAudit(r, session, "administrator.delete", "administrator", id, map[string]any{"email": admin.Email})
	WriteJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/administrators/{id}/password
// Body: {"password": "..."}
// Super-admin only. Sets a new password hash for any administrator (including
// themselves). All their existing sessions are revoked so the new password
// takes effect immediately everywhere.
func SetAdministratorPassword(w http.ResponseWriter, r *http.Request) {
	session := requireSuperAdmin(w, r)
	if session == nil {
		return
	}
	id := r.PathValue("id")

	var admin models.Administrator
	if err := database.Get(database.DBOracle).First(&admin, id).Error; err != nil {
		WriteJSON(w, 404, map[string]any{"error": "Administrator not found"})
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteJSON(w, 400, map[string]any{"error": "Invalid request body"})
		return
	}
	if len(body.Password) < 8 {
		WriteJSON(w, 400, map[string]any{"error": "Password must be at least 8 characters"})
		return
	}
	if err := admin.HashPassword(body.Password); err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to hash password"})
		return
	}
	if err := database.Get(database.DBOracle).Model(&admin).Update("password_hash", admin.PasswordHash).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "Failed to update password"})
		return
	}
	database.Get(database.DBOracle).Where("administrator_id = ?", admin.ID).Delete(&models.Session{})
	logAudit(r, session, "administrator.set_password", "administrator", id, nil)
	WriteJSON(w, 200, map[string]any{"ok": true})
}
