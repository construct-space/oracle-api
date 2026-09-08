package handlers

import (
	"bytes"
	"io"
	"net/http"

	goauth "github.com/construct-space/go-auth"
)

// developerRequest builds a request to the developer service with the shared
// internal secret. Mirrors accountsRequest in accounts_admin.go.
func developerRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := Cfg.DeveloperURL
	if baseURL == "" { baseURL = "https://developer.lisaos.dev" }
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	goauth.WriteInternalSecret(req, Cfg.InternalSecret)
	return http.DefaultClient.Do(req)
}

func proxyDeveloper(w http.ResponseWriter, r *http.Request, method, path string) {
	// Forward the query string (publishers list uses q, kind, verified, page, limit).
	if r.URL.RawQuery != "" { path = path + "?" + r.URL.RawQuery }
	resp, err := developerRequest(method, path, nil)
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "developer unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func proxyDeveloperBody(w http.ResponseWriter, r *http.Request, method, path string) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil { WriteJSON(w, 400, map[string]any{"error": "read body: " + err.Error()}); return }
	defer r.Body.Close()
	resp, err := developerRequest(method, path, bytes.NewReader(reqBody))
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "developer unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

// ─── Publisher handlers (read) ────────────────────────────────────────────

func DeveloperListPublishers(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDeveloper(w, r, "GET", "/api/admin/publishers")
}

func DeveloperGetPublisher(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDeveloper(w, r, "GET", "/api/admin/publishers/"+r.PathValue("id"))
}

// ─── Publisher admin actions ──────────────────────────────────────────────

// auditedDeveloper runs proxyDeveloper through a statusRecorder so we only
// write the audit row on 2xx. Mirrors auditedProxy in accounts_admin.go.
func auditedDeveloper(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxyDeveloper(rec, r, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

func auditedDeveloperBody(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxyDeveloperBody(rec, r, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

func DeveloperVerifyPublisher(w http.ResponseWriter, r *http.Request) {
	auditedDeveloper(w, r, "PUT", "/api/admin/publishers/"+r.PathValue("id")+"/verify", "publisher.verify", "publisher", r.PathValue("id"))
}

func DeveloperUnverifyPublisher(w http.ResponseWriter, r *http.Request) {
	auditedDeveloper(w, r, "PUT", "/api/admin/publishers/"+r.PathValue("id")+"/unverify", "publisher.unverify", "publisher", r.PathValue("id"))
}

// ─── Space moderation (read) ──────────────────────────────────────────────

func DeveloperListPendingSpaces(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDeveloper(w, r, "GET", "/api/admin/spaces/pending-review")
}

func DeveloperListAllSpaces(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDeveloper(w, r, "GET", "/api/admin/spaces/all")
}

func DeveloperGetSpace(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDeveloper(w, r, "GET", "/api/admin/spaces/"+r.PathValue("id"))
}

// ─── Space moderation (write) ─────────────────────────────────────────────

func DeveloperUpdateSpace(w http.ResponseWriter, r *http.Request) {
	auditedDeveloperBody(w, r, "PUT", "/api/admin/spaces/"+r.PathValue("id"), "space.update", "space", r.PathValue("id"))
}

func DeveloperDeleteSpace(w http.ResponseWriter, r *http.Request) {
	auditedDeveloper(w, r, "DELETE", "/api/admin/spaces/"+r.PathValue("id"), "space.delete", "space", r.PathValue("id"))
}

func DeveloperApproveSpace(w http.ResponseWriter, r *http.Request) {
	auditedDeveloper(w, r, "POST", "/api/admin/spaces/"+r.PathValue("id")+"/approve", "space.approve", "space", r.PathValue("id"))
}

func DeveloperRejectSpace(w http.ResponseWriter, r *http.Request) {
	auditedDeveloperBody(w, r, "POST", "/api/admin/spaces/"+r.PathValue("id")+"/reject", "space.reject", "space", r.PathValue("id"))
}

func DeveloperRequestChangesSpace(w http.ResponseWriter, r *http.Request) {
	auditedDeveloperBody(w, r, "POST", "/api/admin/spaces/"+r.PathValue("id")+"/request-changes", "space.request_changes", "space", r.PathValue("id"))
}

func DeveloperUnpublishSpace(w http.ResponseWriter, r *http.Request) {
	auditedDeveloperBody(w, r, "POST", "/api/admin/spaces/"+r.PathValue("id")+"/unpublish", "space.unpublish", "space", r.PathValue("id"))
}

func DeveloperToggleRecommended(w http.ResponseWriter, r *http.Request) {
	auditedDeveloper(w, r, "POST", "/api/admin/spaces/"+r.PathValue("id")+"/toggle-recommended", "space.toggle_recommended", "space", r.PathValue("id"))
}
