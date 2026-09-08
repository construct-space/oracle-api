package main

import (
	"log"
	"net/http"
	"time"

	"construct/oracle/internal/config"
	"construct/oracle/internal/database"
	"construct/oracle/internal/handlers"
	"construct/oracle/internal/middleware"
	"construct/oracle/internal/models"
)

func main() {
	cfg := config.Load()
	handlers.Cfg = cfg
	database.Init(cfg)

	if err := database.Get(database.DBOracle).AutoMigrate(
		&models.Administrator{},
		&models.Session{},
		&models.AuditLog{},
		&models.Post{},
	); err != nil {
		log.Fatalf("auto-migrate: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/login", handlers.AuthLogin)
	mux.HandleFunc("GET /api/auth/me", handlers.AuthMe)
	mux.HandleFunc("POST /api/auth/logout", handlers.AuthLogout)

	mux.HandleFunc("GET /api/dashboard/overview", handlers.DashboardOverview)

	mux.HandleFunc("GET /api/administrators", handlers.ListAdministrators)
	mux.HandleFunc("POST /api/administrators", handlers.CreateAdministrator)
	mux.HandleFunc("PATCH /api/administrators/{id}", handlers.UpdateAdministrator)
	mux.HandleFunc("DELETE /api/administrators/{id}", handlers.DeleteAdministrator)
	mux.HandleFunc("POST /api/administrators/{id}/password", handlers.SetAdministratorPassword)

	mux.HandleFunc("GET /api/audit-log", handlers.ListAuditLog)

	// Blog — authored here, served to the public lisaos.dev/blog.
	// Public read (no auth) returns published posts only; admin CRUD is
	// staff-gated and audit-logged.
	mux.HandleFunc("GET /api/blog/posts", handlers.BlogListPublic)
	mux.HandleFunc("GET /api/blog/posts/{slug}", handlers.BlogGetPublic)
	mux.HandleFunc("GET /api/admin/blog/posts", handlers.BlogAdminList)
	mux.HandleFunc("POST /api/admin/blog/posts", handlers.BlogAdminCreate)
	mux.HandleFunc("GET /api/admin/blog/posts/{id}", handlers.BlogAdminGet)
	mux.HandleFunc("PATCH /api/admin/blog/posts/{id}", handlers.BlogAdminUpdate)
	mux.HandleFunc("DELETE /api/admin/blog/posts/{id}", handlers.BlogAdminDelete)
	mux.HandleFunc("POST /api/admin/blog/posts/{id}/publish", handlers.BlogAdminPublish)
	mux.HandleFunc("POST /api/admin/blog/posts/{id}/unpublish", handlers.BlogAdminUnpublish)

	mux.HandleFunc("GET /api/accounts/users", handlers.ListAccountUsers)
	mux.HandleFunc("GET /api/accounts/users/{id}", handlers.GetAccountUser)
	mux.HandleFunc("GET /api/accounts/sessions", handlers.ListAccountSessions)
	mux.HandleFunc("GET /api/accounts/oauth-clients", handlers.ListAccountOAuthClients)
	mux.HandleFunc("GET /api/accounts/oauth-clients/{id}", handlers.GetAccountOAuthClient)

	mux.HandleFunc("POST /api/accounts/admin/oauth-clients", handlers.AccountsAdminCreateOAuthClient)
	mux.HandleFunc("PATCH /api/accounts/admin/oauth-clients/{id}", handlers.AccountsAdminUpdateOAuthClient)
	mux.HandleFunc("DELETE /api/accounts/admin/oauth-clients/{id}", handlers.AccountsAdminDeleteOAuthClient)
	mux.HandleFunc("POST /api/accounts/admin/oauth-clients/{id}/regenerate-secret", handlers.AccountsAdminRegenerateOAuthClientSecret)
	mux.HandleFunc("DELETE /api/accounts/admin/oauth-clients/{client_id}/authorizations/{user_id}", handlers.AccountsAdminRevokeClientAuthorization)

	mux.HandleFunc("POST /api/accounts/admin/users/{id}/suspend", handlers.AccountsAdminSuspendUser)
	mux.HandleFunc("POST /api/accounts/admin/users/{id}/unsuspend", handlers.AccountsAdminUnsuspendUser)
	mux.HandleFunc("POST /api/accounts/admin/users/{id}/force-logout", handlers.AccountsAdminForceLogout)
	mux.HandleFunc("POST /api/accounts/admin/users/{id}/reset-2fa", handlers.AccountsAdminReset2FA)
	mux.HandleFunc("POST /api/accounts/admin/users/{id}/force-password-reset", handlers.AccountsAdminForcePasswordReset)
	mux.HandleFunc("DELETE /api/accounts/admin/sessions/{id}", handlers.AccountsAdminRevokeSession)
	mux.HandleFunc("DELETE /api/accounts/admin/passkeys/{id}", handlers.AccountsAdminRevokePasskey)

	// Developer: publishers + space moderation (proxied to developer service).
	mux.HandleFunc("GET /api/developer/publishers", handlers.DeveloperListPublishers)
	mux.HandleFunc("GET /api/developer/publishers/{id}", handlers.DeveloperGetPublisher)
	mux.HandleFunc("PUT /api/developer/admin/publishers/{id}/verify", handlers.DeveloperVerifyPublisher)
	mux.HandleFunc("PUT /api/developer/admin/publishers/{id}/unverify", handlers.DeveloperUnverifyPublisher)

	mux.HandleFunc("GET /api/developer/spaces/pending", handlers.DeveloperListPendingSpaces)
	mux.HandleFunc("GET /api/developer/spaces/all", handlers.DeveloperListAllSpaces)
	mux.HandleFunc("GET /api/developer/spaces/{id}", handlers.DeveloperGetSpace)
	mux.HandleFunc("PUT /api/developer/admin/spaces/{id}", handlers.DeveloperUpdateSpace)
	mux.HandleFunc("DELETE /api/developer/admin/spaces/{id}", handlers.DeveloperDeleteSpace)
	mux.HandleFunc("POST /api/developer/admin/spaces/{id}/approve", handlers.DeveloperApproveSpace)
	mux.HandleFunc("POST /api/developer/admin/spaces/{id}/reject", handlers.DeveloperRejectSpace)
	mux.HandleFunc("POST /api/developer/admin/spaces/{id}/request-changes", handlers.DeveloperRequestChangesSpace)
	mux.HandleFunc("POST /api/developer/admin/spaces/{id}/unpublish", handlers.DeveloperUnpublishSpace)
	mux.HandleFunc("POST /api/developer/admin/spaces/{id}/toggle-recommended", handlers.DeveloperToggleRecommended)
	mux.HandleFunc("POST /api/developer/admin/spaces/status", handlers.DeveloperBatchSpaceStatus)

	// Graph: canonical runtime registry — the All Spaces page sources from
	// here so it sees every space with runtime presence, not just the subset
	// that opted into developer's marketplace lifecycle.
	mux.HandleFunc("GET /api/graph/admin/spaces", handlers.GraphAdminListSpaces)
	mux.HandleFunc("GET /api/graph/admin/stats", handlers.GraphAdminStats)
	mux.HandleFunc("GET /api/graph/admin/schemas", handlers.GraphAdminListSchemas)
	mux.HandleFunc("GET /api/graph/admin/schemas/{spaceId}/models", handlers.GraphAdminGetModels)
	mux.HandleFunc("DELETE /api/graph/admin/spaces/{spaceId}", handlers.GraphAdminDeleteSpace)
	mux.HandleFunc("DELETE /api/graph/admin/schemas/{name}", handlers.GraphAdminDeleteSchema)
	mux.HandleFunc("GET /api/graph/admin/schemas/{schemaName}/tables/{tableName}/rows", handlers.GraphAdminGetTableRows)

	// Marketplace: catalog curation — categories, tags, collections,
	// editorial overrides (proxied to marketplace-api).
	mux.HandleFunc("GET /api/marketplace/spaces", handlers.MarketplaceListSpaces)
	mux.HandleFunc("PATCH /api/marketplace/spaces/{name}", handlers.MarketplacePatchSpace)
	mux.HandleFunc("GET /api/marketplace/spaces/{name}/editorial", handlers.MarketplaceGetEditorial)
	mux.HandleFunc("PUT /api/marketplace/spaces/{name}/editorial", handlers.MarketplaceSetEditorial)

	mux.HandleFunc("GET /api/marketplace/categories", handlers.MarketplaceListCategories)
	mux.HandleFunc("POST /api/marketplace/categories", handlers.MarketplaceCreateCategory)
	mux.HandleFunc("PUT /api/marketplace/categories/{slug}", handlers.MarketplacePatchCategory)
	mux.HandleFunc("DELETE /api/marketplace/categories/{slug}", handlers.MarketplaceDeleteCategory)

	mux.HandleFunc("GET /api/marketplace/collections", handlers.MarketplaceListCollections)
	mux.HandleFunc("GET /api/marketplace/collections/{id}", handlers.MarketplaceGetCollection)
	mux.HandleFunc("POST /api/marketplace/collections", handlers.MarketplaceCreateCollection)
	mux.HandleFunc("PUT /api/marketplace/collections/{id}", handlers.MarketplacePatchCollection)
	mux.HandleFunc("DELETE /api/marketplace/collections/{id}", handlers.MarketplaceDeleteCollection)
	mux.HandleFunc("POST /api/marketplace/collections/{id}/spaces", handlers.MarketplaceAddSpaceToCollection)
	mux.HandleFunc("DELETE /api/marketplace/collections/{id}/spaces/{name}", handlers.MarketplaceRemoveSpaceFromCollection)
	mux.HandleFunc("PUT /api/marketplace/collections/{id}/order", handlers.MarketplaceReorderCollection)

	// Source: organizations + projects + teams + invites (proxied to source service).
	mux.HandleFunc("GET /api/source/orgs", handlers.SourceListOrgs)
	mux.HandleFunc("GET /api/source/orgs/{id}", handlers.SourceGetOrg)
	mux.HandleFunc("GET /api/source/orgs/{id}/members", handlers.SourceListOrgMembers)
	mux.HandleFunc("GET /api/source/orgs/{id}/projects", handlers.SourceListOrgProjects)
	mux.HandleFunc("GET /api/source/orgs/{id}/teams", handlers.SourceListOrgTeams)
	mux.HandleFunc("GET /api/source/orgs/{id}/invites", handlers.SourceListOrgInvites)
	mux.HandleFunc("PUT /api/source/admin/org-invites/{id}/revoke", handlers.SourceRevokeInvite)

	// Provider catalog — moved out of source-api into provider-api on
	// 2026-05-11. oracle-web's "Provider" space hits these paths;
	// handlers proxy to provider-api over the private net. The legacy
	// /api/source/providers/* routes are aliased to the same handlers
	// for backward compatibility during cutover; remove in a follow-up
	// once we're confident no client still calls them.
	mux.HandleFunc("GET /api/provider/providers", handlers.SourceListProviders)
	mux.HandleFunc("POST /api/provider/providers", handlers.SourceCreateProvider)
	mux.HandleFunc("GET /api/provider/providers/{id}", handlers.SourceGetProvider)
	mux.HandleFunc("PUT /api/provider/providers/{id}", handlers.SourceUpdateProvider)
	mux.HandleFunc("DELETE /api/provider/providers/{id}", handlers.SourceDeleteProvider)
	mux.HandleFunc("GET /api/provider/providers/{id}/models", handlers.SourceListProviderModels)
	mux.HandleFunc("POST /api/provider/providers/{id}/models", handlers.SourceCreateProviderModel)
	mux.HandleFunc("PUT /api/provider/providers/{id}/models/{modelId}", handlers.SourceUpdateProviderModel)
	mux.HandleFunc("DELETE /api/provider/providers/{id}/models/{modelId}", handlers.SourceDeleteProviderModel)
	mux.HandleFunc("POST /api/provider/providers/{id}/models/{modelId}/sync", handlers.SourceSyncProviderModel)
	mux.HandleFunc("POST /api/provider/providers/{id}/models/{modelId}/unlock", handlers.SourceUnlockProviderModel)

	// Legacy aliases — same handlers, /api/source/providers/* paths.
	// Kept so any in-flight oracle-web tabs from before the rename
	// don't 404. Remove once the new bundle has been live for a day.
	mux.HandleFunc("GET /api/source/providers", handlers.SourceListProviders)
	mux.HandleFunc("POST /api/source/providers", handlers.SourceCreateProvider)
	mux.HandleFunc("GET /api/source/providers/{id}", handlers.SourceGetProvider)
	mux.HandleFunc("PUT /api/source/providers/{id}", handlers.SourceUpdateProvider)
	mux.HandleFunc("DELETE /api/source/providers/{id}", handlers.SourceDeleteProvider)
	mux.HandleFunc("GET /api/source/providers/{id}/models", handlers.SourceListProviderModels)
	mux.HandleFunc("POST /api/source/providers/{id}/models", handlers.SourceCreateProviderModel)
	mux.HandleFunc("PUT /api/source/providers/{id}/models/{modelId}", handlers.SourceUpdateProviderModel)
	mux.HandleFunc("DELETE /api/source/providers/{id}/models/{modelId}", handlers.SourceDeleteProviderModel)
	mux.HandleFunc("POST /api/source/providers/{id}/models/{modelId}/sync", handlers.SourceSyncProviderModel)
	mux.HandleFunc("POST /api/source/providers/{id}/models/{modelId}/unlock", handlers.SourceUnlockProviderModel)

	// Construct — platform-managed provider. Oracle proxies to provider-api's
	// /api/admin/construct/* with X-Internal-Secret. Audit on mutations.
	mux.HandleFunc("GET /api/provider/construct/upstreams", handlers.ConstructListUpstreams)
	mux.HandleFunc("POST /api/provider/construct/upstreams", handlers.ConstructCreateUpstream)
	mux.HandleFunc("PUT /api/provider/construct/upstreams/{id}", handlers.ConstructUpdateUpstream)
	mux.HandleFunc("DELETE /api/provider/construct/upstreams/{id}", handlers.ConstructDeleteUpstream)

	mux.HandleFunc("GET /api/provider/construct/picker-entries", handlers.ConstructListPickerEntries)
	mux.HandleFunc("POST /api/provider/construct/picker-entries", handlers.ConstructCreatePickerEntry)
	mux.HandleFunc("PUT /api/provider/construct/picker-entries/{id}", handlers.ConstructUpdatePickerEntry)
	mux.HandleFunc("DELETE /api/provider/construct/picker-entries/{id}", handlers.ConstructDeletePickerEntry)

	mux.HandleFunc("GET /api/provider/construct/routing-targets", handlers.ConstructListRoutingTargets)
	mux.HandleFunc("POST /api/provider/construct/routing-targets", handlers.ConstructCreateRoutingTarget)
	mux.HandleFunc("PUT /api/provider/construct/routing-targets/{id}", handlers.ConstructUpdateRoutingTarget)
	mux.HandleFunc("DELETE /api/provider/construct/routing-targets/{id}", handlers.ConstructDeleteRoutingTarget)

	mux.HandleFunc("GET /api/provider/construct/config", handlers.ConstructGetConfig)
	mux.HandleFunc("PUT /api/provider/construct/config", handlers.ConstructUpdateConfig)

	mux.HandleFunc("GET /api/provider/construct/users/{id}", handlers.ConstructGetUser)
	mux.HandleFunc("POST /api/provider/construct/users/{id}/grant", handlers.ConstructGrantUser)
	mux.HandleFunc("POST /api/provider/construct/users/{id}/block", handlers.ConstructBlockUser)
	mux.HandleFunc("POST /api/provider/construct/users/{id}/unblock", handlers.ConstructUnblockUser)

	// Source-family operator routing (Tank / Trinity / Apoc / Mouse / Oracle / Neo / Morpheus).
	// Edited from oracle-web's provider/source-family page; provider-api stores the rows.
	mux.HandleFunc("GET /api/provider/source-family", handlers.SourceFamilyList)
	mux.HandleFunc("GET /api/provider/source-family/{id}", handlers.SourceFamilyGet)
	mux.HandleFunc("PUT /api/provider/source-family/{id}", handlers.SourceFamilyUpsert)
	mux.HandleFunc("DELETE /api/provider/source-family/{id}", handlers.SourceFamilyDelete)

	// Source: feed items — content on the construct-app home page top strip.
	mux.HandleFunc("GET /api/source/feed-items", handlers.SourceListFeedItems)
	mux.HandleFunc("POST /api/source/feed-items", handlers.SourceCreateFeedItem)
	mux.HandleFunc("PATCH /api/source/feed-items/{id}", handlers.SourceUpdateFeedItem)
	mux.HandleFunc("DELETE /api/source/feed-items/{id}", handlers.SourceDeleteFeedItem)
	mux.HandleFunc("POST /api/source/feed-items/reorder", handlers.SourceReorderFeedItems)

	// Delivery: email stats, domains (DKIM/SPF/DMARC), messages, API keys.
	mux.HandleFunc("GET /api/delivery/stats", handlers.DeliveryStats)
	mux.HandleFunc("GET /api/delivery/messages", handlers.DeliveryListMessages)
	mux.HandleFunc("GET /api/delivery/domains", handlers.DeliveryListDomains)
	mux.HandleFunc("GET /api/delivery/domains/{id}", handlers.DeliveryGetDomain)
	mux.HandleFunc("GET /api/delivery/domains/{id}/dns-records", handlers.DeliveryGetDomainDNS)
	mux.HandleFunc("GET /api/delivery/domains/{id}/messages", handlers.DeliveryGetDomainMessages)
	mux.HandleFunc("POST /api/delivery/admin/domains/{id}/verify", handlers.DeliveryVerifyDomain)
	mux.HandleFunc("GET /api/delivery/keys", handlers.DeliveryListKeys)
	mux.HandleFunc("GET /api/delivery/tenants", handlers.DeliveryListTenants)
	mux.HandleFunc("GET /api/delivery/notifications", handlers.DeliveryListNotifications)
	mux.HandleFunc("GET /api/delivery/notifications/stats", handlers.DeliveryNotificationStats)
	mux.HandleFunc("POST /api/delivery/admin/notifications/test", handlers.DeliverySendTestNotification)

	// Domains: registrar + DNS + redirect ops (proxied to domains-api).
	mux.HandleFunc("GET /api/domains/stats", handlers.DomainsStats)
	mux.HandleFunc("GET /api/domains/domains", handlers.DomainsListDomains)
	mux.HandleFunc("GET /api/domains/domains/{domain}", handlers.DomainsGetDomain)
	mux.HandleFunc("GET /api/domains/redirects", handlers.DomainsListRedirects)
	mux.HandleFunc("GET /api/domains/tenants", handlers.DomainsListTenants)

	// Telemetry: usage, devices, models, perf, errors, tools + summary &
	// top-N rollups (proxied to telemetry-api).
	mux.HandleFunc("GET /api/telemetry/usage", handlers.TelemetryListUsage)
	mux.HandleFunc("GET /api/telemetry/devices", handlers.TelemetryListDevices)
	mux.HandleFunc("GET /api/telemetry/space-usage", handlers.TelemetryListSpaceUsage)
	mux.HandleFunc("GET /api/telemetry/model-usage", handlers.TelemetryListModelUsage)
	mux.HandleFunc("GET /api/telemetry/perf", handlers.TelemetryListPerf)
	mux.HandleFunc("GET /api/telemetry/errors", handlers.TelemetryListErrors)
	mux.HandleFunc("GET /api/telemetry/tools", handlers.TelemetryListTools)
	mux.HandleFunc("GET /api/telemetry/summary", handlers.TelemetrySummary)
	mux.HandleFunc("GET /api/telemetry/top/users", handlers.TelemetryTopUsers)
	mux.HandleFunc("GET /api/telemetry/top/models", handlers.TelemetryTopModels)
	mux.HandleFunc("GET /api/telemetry/top/spaces", handlers.TelemetryTopSpaces)
	mux.HandleFunc("GET /api/telemetry/trends", handlers.TelemetryTrends)
	mux.HandleFunc("GET /api/telemetry/geo", handlers.TelemetryGeo)
	mux.HandleFunc("GET /api/telemetry/geo/users", handlers.TelemetryGeoUsers)
	mux.HandleFunc("POST /api/telemetry/error", handlers.TelemetryRecordError)

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		handlers.WriteJSON(w, 200, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		handlers.WriteJSON(w, 200, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			handlers.WriteJSON(w, 200, map[string]any{"service": "oracle-api", "status": "ok"})
			return
		}
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			handlers.WriteJSON(w, 404, map[string]any{"error": "Not found"})
			return
		}
		handlers.WriteJSON(w, 404, map[string]any{"error": "Not found"})
	})

	var handler http.Handler = mux
	handler = middleware.CORS(cfg)(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.Logger(handler)

	log.Printf("Oracle API running on :%s", cfg.Port)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
