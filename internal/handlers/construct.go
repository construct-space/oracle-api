package handlers

// Construct admin proxies. Oracle-web calls these under
// /api/provider/construct/*; we forward to provider-api's
// /api/admin/construct/* with X-Internal-Secret.
//
// Audit wraps mutations through auditedProvider / auditedProviderBody
// (defined in source.go alongside the catalog handlers).

import "net/http"

// ─── Upstream providers ───────────────────────────────────────────────────

func ConstructListUpstreams(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyProvider(w, r, "GET", "/api/admin/construct/upstreams")
}

func ConstructCreateUpstream(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST", "/api/admin/construct/upstreams",
		"construct_upstream.create", "construct_upstream", "")
}

func ConstructUpdateUpstream(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "PUT", "/api/admin/construct/upstreams/"+r.PathValue("id"),
		"construct_upstream.update", "construct_upstream", r.PathValue("id"))
}

func ConstructDeleteUpstream(w http.ResponseWriter, r *http.Request) {
	auditedProvider(w, r, "DELETE", "/api/admin/construct/upstreams/"+r.PathValue("id"),
		"construct_upstream.delete", "construct_upstream", r.PathValue("id"))
}

// ─── Picker entries ───────────────────────────────────────────────────────

func ConstructListPickerEntries(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyProvider(w, r, "GET", "/api/admin/construct/picker-entries")
}

func ConstructCreatePickerEntry(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST", "/api/admin/construct/picker-entries",
		"construct_picker_entry.create", "construct_picker_entry", "")
}

func ConstructUpdatePickerEntry(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "PUT", "/api/admin/construct/picker-entries/"+r.PathValue("id"),
		"construct_picker_entry.update", "construct_picker_entry", r.PathValue("id"))
}

func ConstructDeletePickerEntry(w http.ResponseWriter, r *http.Request) {
	auditedProvider(w, r, "DELETE", "/api/admin/construct/picker-entries/"+r.PathValue("id"),
		"construct_picker_entry.delete", "construct_picker_entry", r.PathValue("id"))
}

// ─── Routing targets ──────────────────────────────────────────────────────

func ConstructListRoutingTargets(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyProvider(w, r, "GET", "/api/admin/construct/routing-targets")
}

func ConstructCreateRoutingTarget(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST", "/api/admin/construct/routing-targets",
		"construct_routing_target.create", "construct_routing_target", "")
}

func ConstructUpdateRoutingTarget(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "PUT", "/api/admin/construct/routing-targets/"+r.PathValue("id"),
		"construct_routing_target.update", "construct_routing_target", r.PathValue("id"))
}

func ConstructDeleteRoutingTarget(w http.ResponseWriter, r *http.Request) {
	auditedProvider(w, r, "DELETE", "/api/admin/construct/routing-targets/"+r.PathValue("id"),
		"construct_routing_target.delete", "construct_routing_target", r.PathValue("id"))
}

// ─── Config ───────────────────────────────────────────────────────────────

func ConstructGetConfig(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyProvider(w, r, "GET", "/api/admin/construct/config")
}

func ConstructUpdateConfig(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "PUT", "/api/admin/construct/config",
		"construct_config.update", "construct_config", "")
}

// ─── Users ────────────────────────────────────────────────────────────────

func ConstructGetUser(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	proxyProvider(w, r, "GET", "/api/admin/construct/users/"+r.PathValue("id"))
}

func ConstructGrantUser(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST", "/api/admin/construct/users/"+r.PathValue("id")+"/grant",
		"construct_user.grant", "construct_user", r.PathValue("id"))
}

func ConstructBlockUser(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST", "/api/admin/construct/users/"+r.PathValue("id")+"/block",
		"construct_user.block", "construct_user", r.PathValue("id"))
}

func ConstructUnblockUser(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "POST", "/api/admin/construct/users/"+r.PathValue("id")+"/unblock",
		"construct_user.unblock", "construct_user", r.PathValue("id"))
}
