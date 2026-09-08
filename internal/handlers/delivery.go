package handlers

import (
	"io"
	"net/http"

	goauth "github.com/construct-space/go-auth"
)

func deliveryRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := Cfg.DeliveryURL
	if baseURL == "" { baseURL = "https://delivery.lisaos.dev" }
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	goauth.WriteInternalSecret(req, Cfg.InternalSecret)
	return http.DefaultClient.Do(req)
}

func proxyDelivery(w http.ResponseWriter, r *http.Request, method, path string) {
	if r.URL.RawQuery != "" { path = path + "?" + r.URL.RawQuery }
	resp, err := deliveryRequest(method, path, nil)
	if err != nil { WriteJSON(w, 502, map[string]any{"error": "delivery unreachable: " + err.Error()}); return }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" { w.Header().Set("Content-Type", ct) } else { w.Header().Set("Content-Type", "application/json") }
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// auditedDelivery wraps proxyDelivery with a statusRecorder so we only write an
// audit row on 2xx. Same shape as auditedSource / auditedDeveloper.
func auditedDelivery(w http.ResponseWriter, r *http.Request, method, path, action, resourceType, resourceID string) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	proxyDelivery(rec, r, method, path)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, action, resourceType, resourceID, nil)
	}
}

// ─── Reads (any authenticated staff) ──────────────────────────────────────

func DeliveryStats(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/stats")
}

func DeliveryListMessages(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/messages")
}

func DeliveryListDomains(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/domains")
}

func DeliveryGetDomain(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/domains/"+r.PathValue("id"))
}

func DeliveryGetDomainDNS(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/domains/"+r.PathValue("id")+"/dns-records")
}

func DeliveryGetDomainMessages(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/domains/"+r.PathValue("id")+"/messages")
}

func DeliveryListKeys(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/keys")
}

// Per-tenant footprint. Answers the "who's actually using delivery" question
// on the oracle Tenants page.
func DeliveryListTenants(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/tenants")
}

// ─── Writes (admin only, audited) ─────────────────────────────────────────

func DeliveryVerifyDomain(w http.ResponseWriter, r *http.Request) {
	auditedDelivery(w, r, "POST", "/api/admin/domains/"+r.PathValue("id")+"/verify",
		"domain.verify", "sending_domain", r.PathValue("id"))
}

// ─── Notifications (read) ─────────────────────────────────────────────────

func DeliveryListNotifications(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/notifications")
}

func DeliveryNotificationStats(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyDelivery(w, r, "GET", "/api/admin/notifications/stats")
}

// ─── Notifications (write, audited) ───────────────────────────────────────

// DeliverySendTestNotification forwards the JSON body to delivery's admin
// test endpoint. Audited so we have a paper trail of staff-emitted events
// (these can reach end users' devices, so they're not no-op).
func DeliverySendTestNotification(w http.ResponseWriter, r *http.Request) {
	session := requireAdmin(w, r)
	if session == nil { return }
	rec := recordStatus(w)
	resp, err := deliveryRequest("POST", "/api/admin/notifications/test", r.Body)
	if err != nil {
		WriteJSON(rec, 502, map[string]any{"error": "delivery unreachable: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		rec.Header().Set("Content-Type", ct)
	} else {
		rec.Header().Set("Content-Type", "application/json")
	}
	rec.WriteHeader(resp.StatusCode)
	_, _ = rec.Write(body)
	if rec.status >= 200 && rec.status < 300 {
		logAudit(r, session, "notification.test_send", "notification", "", nil)
	}
}
