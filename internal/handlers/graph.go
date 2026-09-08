package handlers

import (
	"io"
	"net/http"

	goauth "github.com/construct-space/go-auth"
)

// graphRequest builds an X-Internal-Secret request to the graph service.
// Mirror of developerRequest. Graph's admin endpoints accept either the
// internal secret (this path, used by Oracle) or a logged-in admin
// session — Oracle always uses the secret since it has its own admin
// session model and proxies on the user's behalf.
func graphRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := Cfg.GraphURL
	if baseURL == "" {
		baseURL = "https://graph.lisaos.dev"
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

func proxyGraph(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" {
		path = path + "?" + r.URL.RawQuery
	}
	resp, err := graphRequest(method, path, nil)
	if err != nil {
		WriteJSON(w, 502, map[string]any{"error": "graph unreachable: " + err.Error()})
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

// GET /api/graph/admin/spaces — list every space across all orgs from the
// graph (canonical runtime registry). Oracle's All Spaces page joins this
// with developer's batch status endpoint to overlay marketplace lifecycle.
func GraphAdminListSpaces(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyGraph(w, r, "GET", "/api/admin/spaces")
}

// GET /api/graph/admin/stats — graph-wide health: schemas, tables, rows, db size.
func GraphAdminStats(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyGraph(w, r, "GET", "/api/admin/stats")
}

// GET /api/graph/admin/schemas — every provisioned schema (one per
// space+project pair). Oracle's GraphPage table-renders this list.
func GraphAdminListSchemas(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyGraph(w, r, "GET", "/api/admin/schemas")
}

// GET /api/graph/admin/schemas/{spaceId}/models — model definitions for a
// space. Used by drill-into-schema views.
func GraphAdminGetModels(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	spaceID := r.PathValue("spaceId")
	proxyGraph(w, r, "GET", "/api/admin/schemas/"+spaceID+"/models")
}

// DELETE /api/graph/admin/spaces/{spaceId} — drop everything for a space:
// all provisioned schemas + manifest history + the spaces row. Used to
// clean up orphan / not-submitted spaces from the All Spaces view.
func GraphAdminDeleteSpace(w http.ResponseWriter, r *http.Request) {
	// Irreversible cross-tenant data destruction — require an admin role,
	// not merely an authenticated session (matches every other destructive
	// oracle action).
	if requireAdmin(w, r) == nil {
		return
	}
	spaceID := r.PathValue("spaceId")
	proxyGraph(w, r, "DELETE", "/api/admin/spaces/"+spaceID)
}

// DELETE /api/graph/admin/schemas/{name} — drop one provisioned schema by
// its actual schema name (not space id). Used by Oracle's schema-detail
// page when the reviewer wants to drop a single project's data while
// leaving other schemas (other projects) for the same space alone.
func GraphAdminDeleteSchema(w http.ResponseWriter, r *http.Request) {
	// Irreversible schema drop — require an admin role (see DeleteSpace).
	if requireAdmin(w, r) == nil {
		return
	}
	name := r.PathValue("name")
	proxyGraph(w, r, "DELETE", "/api/admin/schemas/"+name)
}

// GET /api/graph/admin/schemas/{schemaName}/tables/{tableName}/rows
// Browse rows from a single table inside a provisioned schema. limit +
// offset query params are forwarded as-is by proxyGraph.
func GraphAdminGetTableRows(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	schemaName := r.PathValue("schemaName")
	tableName := r.PathValue("tableName")
	proxyGraph(w, r, "GET", "/api/admin/schemas/"+schemaName+"/tables/"+tableName+"/rows")
}

// POST /api/developer/admin/spaces/status — batch lookup of marketplace
// status for a list of space names. Routed here so Oracle's frontend hits
// a same-origin endpoint with the existing session cookie. Body is
// forwarded verbatim.
func DeveloperBatchSpaceStatus(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyDeveloperBody(w, r, "POST", "/api/admin/spaces/status")
}
