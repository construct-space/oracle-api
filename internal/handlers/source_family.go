package handlers

// Source-family routing proxy. Oracle's provider/source-family page
// hits these endpoints; we forward to provider-api with the internal
// secret. Mutations are audited like the other admin actions.

import "net/http"

func SourceFamilyList(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyProvider(w, r, "GET", "/api/admin/source-family")
}

func SourceFamilyGet(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil {
		return
	}
	proxyProvider(w, r, "GET", "/api/admin/source-family/"+r.PathValue("id"))
}

func SourceFamilyUpsert(w http.ResponseWriter, r *http.Request) {
	auditedProviderBody(w, r, "PUT", "/api/admin/source-family/"+r.PathValue("id"),
		"source_family.upsert", "source_family_operator", r.PathValue("id"))
}

func SourceFamilyDelete(w http.ResponseWriter, r *http.Request) {
	auditedProvider(w, r, "DELETE", "/api/admin/source-family/"+r.PathValue("id"),
		"source_family.delete", "source_family_operator", r.PathValue("id"))
}
