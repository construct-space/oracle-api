package handlers

import (
	"bytes"
	"io"
	"net/http"

	goauth "github.com/construct-space/go-auth"
)

func sourceRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := Cfg.SourceURL
	if baseURL == "" { baseURL = "https://source.lisaos.dev" }
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	goauth.WriteInternalSecret(req, Cfg.InternalSecret)
	return http.DefaultClient.Do(req)
}

func proxySource(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" { path = path + "?" + r.URL.RawQuery }
	resp, err := sourceRequest(method, path, nil)
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "source unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// proxySourceWithBody forwards the caller's request body (needed for POST/
// PUT). Otherwise identical to proxySource.
func proxySourceWithBody(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" { path = path + "?" + r.URL.RawQuery }
	payload, _ := io.ReadAll(r.Body)
	resp, err := sourceRequest(method, path, bytes.NewReader(payload))
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "source unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// ─── Organizations ────────────────────────────────────────────────────────

func SourceListOrgs(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxySource(w, r, "GET", "/api/admin/orgs")
}

func SourceGetOrg(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxySource(w, r, "GET", "/api/admin/orgs/"+r.PathValue("id"))
}

func SourceListOrgMembers(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxySource(w, r, "GET", "/api/admin/orgs/"+r.PathValue("id")+"/members")
}

func SourceListOrgProjects(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxySource(w, r, "GET", "/api/admin/orgs/"+r.PathValue("id")+"/projects")
}

func SourceListOrgTeams(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxySource(w, r, "GET", "/api/admin/orgs/"+r.PathValue("id")+"/teams")
}

func SourceListOrgInvites(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxySource(w, r, "GET", "/api/admin/orgs/"+r.PathValue("id")+"/invites")
}

// auditedSource mirrors auditedProxy — runs the proxy through a statusRecorder
// and only writes the audit row on a 2xx response.
func auditedSource(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxySource(rec, r, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

func auditedSourceBody(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxySourceWithBody(rec, r, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

// ─── Provider-api proxy ───────────────────────────────────────────────────
// Provider catalog moved out of source-api into provider-api on 2026-05-11.
// Same audit shape as the source family — only the upstream URL changes.

func providerRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := Cfg.ProviderURL
	if baseURL == "" { baseURL = "http://srv-captain--provider-api" }
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	goauth.WriteInternalSecret(req, Cfg.InternalSecret)
	return http.DefaultClient.Do(req)
}

func proxyProvider(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" { path = path + "?" + r.URL.RawQuery }
	resp, err := providerRequest(method, path, nil)
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "provider unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func proxyProviderWithBody(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" { path = path + "?" + r.URL.RawQuery }
	payload, _ := io.ReadAll(r.Body)
	resp, err := providerRequest(method, path, bytes.NewReader(payload))
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "provider unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func auditedProvider(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxyProvider(rec, r, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

func auditedProviderBody(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxyProviderWithBody(rec, r, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

func SourceRevokeInvite(w http.ResponseWriter, r *http.Request) {
	auditedSource(w, r, "PUT", "/api/admin/org-invites/"+r.PathValue("id")+"/revoke", "invite.revoke", "org_invite", r.PathValue("id"))
}

// ─── Provider catalog ─────────────────────────────────────────────────────
// Catalog moved to provider-api on 2026-05-11; oracle-web's existing
// /api/source/providers* paths still hit oracle-api, oracle-api just
// proxies to provider-api now. Same audit semantics as before.

func SourceListProviders(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyProvider(w, r, "GET", "/api/admin/providers")
}

func SourceGetProvider(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyProvider(w, r, "GET", "/api/admin/providers/"+r.PathValue("id"))
}

func SourceCreateProvider(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST", "/api/admin/providers", "provider.create", "provider", "")
}

func SourceUpdateProvider(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "PUT", "/api/admin/providers/"+r.PathValue("id"), "provider.update", "provider", r.PathValue("id"))
}

func SourceDeleteProvider(w http.ResponseWriter, r *http.Request) {
	auditedProvider(w, r, "DELETE", "/api/admin/providers/"+r.PathValue("id"), "provider.delete", "provider", r.PathValue("id"))
}

func SourceListProviderModels(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyProvider(w, r, "GET", "/api/admin/providers/"+r.PathValue("id")+"/models")
}

func SourceCreateProviderModel(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST", "/api/admin/providers/"+r.PathValue("id")+"/models", "provider_model.create", "provider_model", r.PathValue("id"))
}

func SourceUpdateProviderModel(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "PUT", "/api/admin/providers/"+r.PathValue("id")+"/models/"+r.PathValue("modelId"), "provider_model.update", "provider_model", r.PathValue("modelId"))
}

func SourceDeleteProviderModel(w http.ResponseWriter, r *http.Request) {
	auditedProvider(w, r, "DELETE", "/api/admin/providers/"+r.PathValue("id")+"/models/"+r.PathValue("modelId"), "provider_model.delete", "provider_model", r.PathValue("modelId"))
}

// Per-model sync from models.dev. Skips rows the admin has edited.
func SourceSyncProviderModel(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST",
		"/api/admin/providers/"+r.PathValue("id")+"/models/"+r.PathValue("modelId")+"/sync",
		"provider_model.sync", "provider_model", r.PathValue("modelId"))
}

// Clears the "locked" flag on a model row so the next sync can
// overwrite it.
func SourceUnlockProviderModel(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST",
		"/api/admin/providers/"+r.PathValue("id")+"/models/"+r.PathValue("modelId")+"/unlock",
		"provider_model.unlock", "provider_model", r.PathValue("modelId"))
}

// ─── Feed (homepage top-strip admin) ──────────────────────────────────────
// Proxies to source's /api/admin/feed-items. Read is authed-staff-only;
// mutations audit-log like the other write paths.

func SourceListFeedItems(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxySource(w, r, "GET", "/api/admin/feed-items")
}

func SourceCreateFeedItem(w http.ResponseWriter, r *http.Request) {
	auditedSourceBody(w, r, "POST", "/api/admin/feed-items", "feed_item.create", "feed_item", "")
}

func SourceUpdateFeedItem(w http.ResponseWriter, r *http.Request) {
	auditedSourceBody(w, r, "PATCH", "/api/admin/feed-items/"+r.PathValue("id"), "feed_item.update", "feed_item", r.PathValue("id"))
}

func SourceDeleteFeedItem(w http.ResponseWriter, r *http.Request) {
	auditedSource(w, r, "DELETE", "/api/admin/feed-items/"+r.PathValue("id"), "feed_item.delete", "feed_item", r.PathValue("id"))
}

func SourceReorderFeedItems(w http.ResponseWriter, r *http.Request) {
	auditedSourceBody(w, r, "POST", "/api/admin/feed-items/reorder", "feed_item.reorder", "feed_item", "")
}
