package handlers

import (
	"io"
	"net/http"

	goauth "github.com/construct-space/go-auth"
)

// domains.go — thin proxy into domains-api's /api/admin/* routes. Shape
// mirrors delivery.go so oracle-web can consume both with the same pattern.

func domainsRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := Cfg.DomainsURL
	if baseURL == "" {
		baseURL = "https://domains-api.lisaos.dev"
	}
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if Cfg.InternalSecret != "" {
		goauth.WriteInternalSecret(req, Cfg.InternalSecret)
	}
	return http.DefaultClient.Do(req)
}

func proxyDomains(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" {
		path = path + "?" + r.URL.RawQuery
	}
	resp, err := domainsRequest(method, path, nil)
	if err != nil {
		WriteJSON(w, 502, map[string]any{"error": "domains unreachable: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// ─── Reads (any authenticated staff) ──────────────────────────────────────

func DomainsStats(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyDomains(w, r, "GET", "/api/admin/stats")
}

func DomainsListDomains(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyDomains(w, r, "GET", "/api/admin/domains")
}

func DomainsGetDomain(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyDomains(w, r, "GET", "/api/admin/domains/"+r.PathValue("domain"))
}

func DomainsListRedirects(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyDomains(w, r, "GET", "/api/admin/redirects")
}

func DomainsListTenants(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyDomains(w, r, "GET", "/api/admin/tenants")
}
