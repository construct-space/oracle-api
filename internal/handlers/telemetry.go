package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	goauth "github.com/construct-space/go-auth"
)

// telemetry.go — proxy into telemetry-api's /api/admin/* routes, signed
// with X-Internal-Secret. Shape mirrors delivery.go + domains.go so
// oracle-web can consume them with the same pattern.

func telemetryRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := Cfg.TelemetryURL
	if baseURL == "" {
		baseURL = "https://telemetry.lisaos.dev"
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

func proxyTelemetry(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" {
		path = path + "?" + r.URL.RawQuery
	}
	resp, err := telemetryRequest(method, path, nil)
	if err != nil {
		WriteJSON(w, 502, map[string]any{"error": "telemetry unreachable: " + err.Error()})
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

func TelemetryListUsage(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/usage")
}

func TelemetryListDevices(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/devices")
}

func TelemetryListSpaceUsage(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/space-usage")
}

func TelemetryListModelUsage(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/model-usage")
}

func TelemetryListPerf(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/perf")
}

func TelemetryListErrors(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/errors")
}

func TelemetryListTools(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/tools")
}

func TelemetrySummary(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/summary")
}

func TelemetryTopUsers(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/top/users")
}

func TelemetryTopModels(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/top/models")
}

func TelemetryTopSpaces(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/top/spaces")
}

func TelemetryTrends(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/trends")
}

func TelemetryGeo(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/geo")
}

func TelemetryGeoUsers(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyTelemetry(w, r, "GET", "/api/admin/geo/users")
}

// ─── Writes (oracle-web global error handler) ─────────────────────────────

// TelemetryRecordError accepts a low-cardinality error class string from
// the oracle-web error handler and ships it as a one-row events batch to
// telemetry-api. Stamped against the staff session's administrator id —
// the user_id column has no FK so cross-namespace IDs are tolerated.
//
// Body: { "error_class": "type_error" }  (max 80 chars; longer is trimmed)
func TelemetryRecordError(w http.ResponseWriter, r *http.Request) {
	s := requireAuth(w, r)
	if s == nil {
		return
	}
	if s.AdministratorID == nil || *s.AdministratorID == 0 {
		WriteJSON(w, 400, map[string]string{"error": "no administrator id on session"})
		return
	}

	var body struct {
		ErrorClass string `json:"error_class"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	cls := strings.TrimSpace(body.ErrorClass)
	if cls == "" {
		WriteJSON(w, 400, map[string]string{"error": "error_class is required"})
		return
	}
	if len(cls) > 80 {
		cls = cls[:80]
	}

	batch := map[string]any{
		"date":        time.Now().UTC().Format("2006-01-02"),
		"app_version": "oracle",
		"usage":       map[string]any{"errors": 1},
		"errors":      []map[string]any{{"error_class": cls, "count": 1}},
	}
	payload, err := json.Marshal(batch)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "marshal: " + err.Error()})
		return
	}

	baseURL := Cfg.TelemetryURL
	if baseURL == "" {
		baseURL = "https://telemetry.lisaos.dev"
	}
	req, err := http.NewRequest("POST", baseURL+"/api/events", bytes.NewReader(payload))
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if Cfg.InternalSecret != "" {
		goauth.WriteInternalSecret(req, Cfg.InternalSecret)
	}
	req.Header.Set("X-Auth-User-Row-ID", strconv.FormatUint(uint64(*s.AdministratorID), 10))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		WriteJSON(w, 502, map[string]string{"error": "telemetry unreachable: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		WriteJSON(w, 502, map[string]string{"error": fmt.Sprintf("telemetry %d: %s", resp.StatusCode, string(preview))})
		return
	}
	WriteJSON(w, 202, map[string]any{"recorded": true})
}
