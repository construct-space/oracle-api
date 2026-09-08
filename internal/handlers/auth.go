package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"construct/oracle/internal/database"
	"construct/oracle/internal/models"
)

func AuthLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	email := r.FormValue("email")
	password := r.FormValue("password")
	if email == "" || password == "" {
		if body, err := parseBody(r); err == nil {
			email, _ = body["email"].(string)
			password, _ = body["password"].(string)
		}
	}
	if email == "" || password == "" {
		WriteJSON(w, 400, map[string]any{"error": "Email and password are required"})
		return
	}
	var admin models.Administrator
	if err := database.Get(database.DBOracle).Where("email = ?", email).First(&admin).Error; err != nil || !admin.CheckPassword(password) {
		WriteJSON(w, 401, map[string]any{"error": "Invalid email or password"})
		return
	}
	if !admin.Active {
		WriteJSON(w, 403, map[string]any{"error": "Your administrator account is disabled"})
		return
	}
	now := time.Now()
	database.Get(database.DBOracle).Model(&admin).Update("last_login", &now)
	name := strings.TrimSpace(admin.FirstName + " " + admin.LastName)
	if name == "" { name = admin.Email }
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	session := models.Session{Token: models.GenerateSessionToken(), Email: strPtr(admin.Email), Name: strPtr(name), Role: strPtr(admin.Role), AdministratorID: &admin.ID, UserAgent: strPtr(r.UserAgent()), IPAddress: strPtr(r.RemoteAddr), ExpiresAt: &expiresAt, CreatedAt: &now}
	database.Get(database.DBOracle).Create(&session)
	database.Get(database.DBOracle).Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	secure := ""
	if !strings.HasPrefix(Cfg.AppURL, "http://localhost") && !strings.HasPrefix(Cfg.AppURL, "http://127.0.0.1") { secure = " Secure;" }
	cookie := fmt.Sprintf("session=%s; Path=/; HttpOnly;%s SameSite=Lax; Max-Age=%d", session.Token, secure, 30*24*60*60)
	w.Header().Set("Set-Cookie", cookie)
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, 200, map[string]any{"ok": true, "message": "Login successful"})
}

func AuthMe(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		WriteJSON(w, 401, map[string]any{"authenticated": false})
		return
	}
	userData := map[string]any{"id": session.ID}
	if session.Email != nil { userData["email"] = *session.Email }
	if session.Name != nil { userData["name"] = *session.Name }
	if session.Role != nil { userData["role"] = *session.Role }
	if session.AdministratorID != nil { userData["administrator_id"] = *session.AdministratorID }
	WriteJSON(w, 200, map[string]any{"authenticated": true, "user": userData})
}

func AuthLogout(w http.ResponseWriter, r *http.Request) {
	if session := getSession(r); session != nil { database.Get(database.DBOracle).Delete(session) }
	secure := ""
	if !strings.HasPrefix(Cfg.AppURL, "http://localhost") && !strings.HasPrefix(Cfg.AppURL, "http://127.0.0.1") { secure = " Secure;" }
	w.Header().Set("Set-Cookie", fmt.Sprintf("session=; Path=/; HttpOnly;%s SameSite=Lax; Max-Age=0", secure))
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, 200, map[string]any{"ok": true})
}
