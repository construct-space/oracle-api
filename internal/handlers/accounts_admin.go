package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	goauth "github.com/construct-space/go-auth"
)

func accountsRequest(method, path string, body any) (*http.Response, error) {
	baseURL := Cfg.AccountsURL
	if baseURL == "" { baseURL = "https://accounts.lisaos.dev" }
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil { return nil, fmt.Errorf("marshal body: %w", err) }
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	goauth.WriteInternalSecret(req, Cfg.InternalSecret)
	return http.DefaultClient.Do(req)
}

func proxyAccountsAdmin(w http.ResponseWriter, method, path string) {
	resp, err := accountsRequest(method, path, nil)
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "accounts unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// proxyAccountsAdminBody forwards the caller's JSON body to the accounts
// admin endpoint. Used for create/update where the payload matters.
func proxyAccountsAdminBody(w http.ResponseWriter, r *http.Request, method, path string) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil { WriteJSON(w, 400, map[string]any{"error": "read body: " + err.Error()}); return }
	defer r.Body.Close()

	baseURL := Cfg.AccountsURL
	if baseURL == "" { baseURL = "https://accounts.lisaos.dev" }
	req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(reqBody))
	if err != nil { WriteJSON(w, 500, map[string]any{"error": err.Error()}); return }
	req.Header.Set("Content-Type", "application/json")
	goauth.WriteInternalSecret(req, Cfg.InternalSecret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "accounts unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

// auditedProxy wraps proxyAccountsAdmin with audit recording — runs the proxy
// through a statusRecorder, and on a 2xx response writes one audit row. Using
// this pattern instead of logging before the call means a downstream 500
// doesn't leave a stray "suspend succeeded" entry.
func auditedProxy(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxyAccountsAdmin(rec, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

func auditedProxyBody(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxyAccountsAdminBody(rec, r, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

func AccountsAdminSuspendUser(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "POST", "/api/admin/users/"+r.PathValue("id")+"/suspend", "user.suspend", "user", r.PathValue("id"))
}
func AccountsAdminUnsuspendUser(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "POST", "/api/admin/users/"+r.PathValue("id")+"/unsuspend", "user.unsuspend", "user", r.PathValue("id"))
}
func AccountsAdminForceLogout(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "POST", "/api/admin/users/"+r.PathValue("id")+"/force-logout", "user.force_logout", "user", r.PathValue("id"))
}
func AccountsAdminReset2FA(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "POST", "/api/admin/users/"+r.PathValue("id")+"/reset-2fa", "user.reset_2fa", "user", r.PathValue("id"))
}
func AccountsAdminForcePasswordReset(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "POST", "/api/admin/users/"+r.PathValue("id")+"/force-password-reset", "user.force_password_reset", "user", r.PathValue("id"))
}
func AccountsAdminRevokeSession(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "DELETE", "/api/admin/sessions/"+r.PathValue("id"), "session.revoke", "session", r.PathValue("id"))
}
func AccountsAdminRevokePasskey(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "DELETE", "/api/admin/passkeys/"+r.PathValue("id"), "passkey.revoke", "passkey", r.PathValue("id"))
}

// OAuth clients — CRUD + secret rotation + authorization revoke.
func AccountsAdminCreateOAuthClient(w http.ResponseWriter, r *http.Request) {
	auditedProxyBody(w, r, "POST", "/api/admin/oauth-clients", "oauth_client.create", "oauth_client", "")
}
func AccountsAdminUpdateOAuthClient(w http.ResponseWriter, r *http.Request) {
	auditedProxyBody(w, r, "PATCH", "/api/admin/oauth-clients/"+r.PathValue("id"), "oauth_client.update", "oauth_client", r.PathValue("id"))
}
func AccountsAdminDeleteOAuthClient(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "DELETE", "/api/admin/oauth-clients/"+r.PathValue("id"), "oauth_client.delete", "oauth_client", r.PathValue("id"))
}
func AccountsAdminRegenerateOAuthClientSecret(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "POST", "/api/admin/oauth-clients/"+r.PathValue("id")+"/regenerate-secret", "oauth_client.regenerate_secret", "oauth_client", r.PathValue("id"))
}
func AccountsAdminRevokeClientAuthorization(w http.ResponseWriter, r *http.Request) {
	auditedProxy(w, r, "DELETE", "/api/admin/oauth-clients/"+r.PathValue("client_id")+"/authorizations/"+r.PathValue("user_id"),
		"oauth_client.revoke_authorization", "oauth_client", r.PathValue("client_id"))
}
