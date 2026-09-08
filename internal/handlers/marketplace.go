// marketplace.go — thin proxy from oracle-api to marketplace-api.
//
// Mirrors developer.go / accounts_admin.go: oracle-web calls /api/marketplace/*
// on its own origin; nginx forwards to oracle-api; this file forwards each
// request on to marketplace-api with X-Internal-Secret. Marketplace owns
// the data; oracle is just the staff control surface.
package handlers

import (
	"bytes"
	"io"
	"net/http"

	goauth "github.com/construct-space/go-auth"
)

func marketplaceRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := Cfg.MarketplaceURL
	if baseURL == "" {
		baseURL = "https://marketplace-api.lisaos.dev"
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

func proxyMarketplace(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" {
		path = path + "?" + r.URL.RawQuery
	}
	resp, err := marketplaceRequest(method, path, nil)
	if err != nil {
		WriteJSON(w, 502, map[string]any{"error": "marketplace unreachable: " + err.Error()})
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

func proxyMarketplaceBody(w http.ResponseWriter, r *http.Request, method, path string) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		WriteJSON(w, 400, map[string]any{"error": "read body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	resp, err := marketplaceRequest(method, path, bytes.NewReader(reqBody))
	if err != nil {
		WriteJSON(w, 502, map[string]any{"error": "marketplace unreachable: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

// ---------- Spaces ----------

// SECURITY: every handler below proxies to marketplace-api's /api/admin/*
// surface with the internal secret, which marketplace-api treats as full
// staff authority. They MUST gate on an oracle admin session first — reads
// via requireAuth, mutations via requireAdmin — exactly like developer.go
// and every other oracle proxy group. (These were previously ungated, so
// any caller reaching oracle could read/rewrite the public catalog.)

func MarketplaceListSpaces(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyMarketplace(w, r, "GET", "/api/admin/spaces")
}

func MarketplacePatchSpace(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	name := r.PathValue("name")
	proxyMarketplaceBody(w, r, "PATCH", "/api/admin/spaces/"+name)
}

func MarketplaceGetEditorial(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	name := r.PathValue("name")
	proxyMarketplace(w, r, "GET", "/api/admin/spaces/"+name+"/editorial")
}

func MarketplaceSetEditorial(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	name := r.PathValue("name")
	proxyMarketplaceBody(w, r, "PUT", "/api/admin/spaces/"+name+"/editorial")
}

// ---------- Categories ----------

func MarketplaceListCategories(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyMarketplace(w, r, "GET", "/api/admin/categories")
}

func MarketplaceCreateCategory(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	proxyMarketplaceBody(w, r, "POST", "/api/admin/categories")
}

func MarketplacePatchCategory(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	slug := r.PathValue("slug")
	proxyMarketplaceBody(w, r, "PUT", "/api/admin/categories/"+slug)
}

func MarketplaceDeleteCategory(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	slug := r.PathValue("slug")
	proxyMarketplace(w, r, "DELETE", "/api/admin/categories/"+slug)
}

// ---------- Collections ----------

func MarketplaceListCollections(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyMarketplace(w, r, "GET", "/api/admin/collections")
}

func MarketplaceGetCollection(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	id := r.PathValue("id")
	proxyMarketplace(w, r, "GET", "/api/admin/collections/"+id)
}

func MarketplaceCreateCollection(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	proxyMarketplaceBody(w, r, "POST", "/api/admin/collections")
}

func MarketplacePatchCollection(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	id := r.PathValue("id")
	proxyMarketplaceBody(w, r, "PUT", "/api/admin/collections/"+id)
}

func MarketplaceDeleteCollection(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	id := r.PathValue("id")
	proxyMarketplace(w, r, "DELETE", "/api/admin/collections/"+id)
}

func MarketplaceAddSpaceToCollection(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	id := r.PathValue("id")
	proxyMarketplaceBody(w, r, "POST", "/api/admin/collections/"+id+"/spaces")
}

func MarketplaceRemoveSpaceFromCollection(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	id := r.PathValue("id")
	name := r.PathValue("name")
	proxyMarketplace(w, r, "DELETE", "/api/admin/collections/"+id+"/spaces/"+name)
}

func MarketplaceReorderCollection(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	id := r.PathValue("id")
	proxyMarketplaceBody(w, r, "PUT", "/api/admin/collections/"+id+"/order")
}
