package config

import (
	"bufio"
	"os"
	"strings"
)

func init() {
	loadEnvFile(".env")
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			k := strings.TrimSpace(line[:idx])
			v := strings.TrimSpace(line[idx+1:])
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}

type Config struct {
	Port           string
	AppURL         string
	AllowedOrigins []string
	DBDriver       string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPass         string
	DBSSL          string
	DBNameOracle   string
	DBNameAccounts string
	AccountsURL    string
	DeveloperURL   string
	GraphURL       string
	SourceURL      string
	ProviderURL    string
	DeliveryURL    string
	DomainsURL     string
	TelemetryURL   string
	MarketplaceURL string
	InternalSecret string
	// Transactional email (new-administrator welcome) goes through
	// delivery-api's per-tenant /api/emails endpoint with a cd_live_
	// bearer key — same pattern as accounts-api.
	DeliveryAPIKey string
	EmailFrom      string
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func Load() *Config {
	return &Config{
		Port:           env("PORT", "4200"),
		AppURL:         env("APP_URL", "http://localhost:4200"),
		AllowedOrigins: append([]string{env("APP_URL", "http://localhost:4200"), "http://localhost:5173", "http://localhost:5174"}, splitCSV(env("ALLOWED_ORIGINS", ""))...),
		DBDriver:       env("DB_DRIVER", "mysql"),
		DBHost:         env("DB_HOST", "localhost"),
		DBPort:         env("DB_PORT", "3306"),
		DBUser:         env("DB_USER", "root"),
		DBPass:         env("DB_PASS", ""),
		DBSSL:          env("DB_SSL", "false"),
		DBNameOracle:   env("DB_NAME_ORACLE", "oracle"),
		DBNameAccounts: env("DB_NAME_ACCOUNTS", "accounts"),
		AccountsURL:    env("ACCOUNTS_URL", "http://srv-captain--accounts"),
		DeveloperURL:   env("DEVELOPER_URL", "http://srv-captain--developer"),
		GraphURL:       env("GRAPH_URL", "http://srv-captain--graph"),
		SourceURL:      env("SOURCE_URL", "http://srv-captain--source"),
		ProviderURL:    env("PROVIDER_URL", "http://srv-captain--provider-api"),
		DeliveryURL:    env("DELIVERY_URL", "http://srv-captain--delivery"),
		DomainsURL:     env("DOMAINS_URL", "http://srv-captain--domains-api"),
		TelemetryURL:   env("TELEMETRY_URL", "http://srv-captain--telemetry-api"),
		MarketplaceURL: env("MARKETPLACE_URL", "http://srv-captain--marketplace-api"),
		InternalSecret: env("INTERNAL_SHARED_SECRET", ""),
		DeliveryAPIKey: env("DELIVERY_API_KEY", ""),
		EmailFrom:      env("EMAIL_FROM", "Construct Oracle <noreply@lisaos.dev>"),
	}
}
