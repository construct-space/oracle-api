// Package mailer sends transactional email through delivery-api. Mirrors
// the accounts-api mailer: POST /api/emails with Authorization: Bearer cd_live_,
// which scopes the send to oracle's own tenant record on delivery.
package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"construct/oracle/internal/config"
)

type emailRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

func send(cfg *config.Config, to, subject, html, text string) error {
	if cfg.DeliveryURL == "" || cfg.DeliveryAPIKey == "" {
		// Dev / not-configured path: log only. Never silently succeed in a
		// way ops can't trace — the log line is the breadcrumb.
		log.Printf("[oracle-mailer][DEV] would send %q to %s", subject, to)
		return nil
	}

	body, _ := json.Marshal(emailRequest{
		From: cfg.EmailFrom, To: to, Subject: subject, HTML: html, Text: text,
	})
	req, err := http.NewRequest("POST", cfg.DeliveryURL+"/api/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.DeliveryAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("delivery returned %d", resp.StatusCode)
	}
	log.Printf("[oracle-mailer] sent %q to %s", subject, to)
	return nil
}

// SendAdminWelcome delivers the creds to a newly-created administrator.
// `loginURL` is the oracle sign-in page (typically cfg.AppURL). The
// temporary password is sent in plaintext — the UX expects the admin
// to sign in + change it immediately.
func SendAdminWelcome(cfg *config.Config, toEmail, firstName, username, tempPassword, loginURL string) error {
	display := firstName
	if display == "" {
		display = username
	}
	subject := "Your Construct Oracle administrator account"

	text := "Hi " + display + ",\n\n" +
		"A Construct Oracle administrator account has been created for you.\n\n" +
		"  Sign in:   " + loginURL + "\n" +
		"  Username:  " + username + "\n" +
		"  Password:  " + tempPassword + "\n\n" +
		"Please sign in and change your password immediately.\n\n" +
		"— Construct\n"

	html := `<!doctype html><html><body style="font-family:system-ui,sans-serif;max-width:560px;margin:40px auto;padding:24px;color:#1a1a1a">` +
		`<h2 style="margin-bottom:8px">Welcome to Construct Oracle</h2>` +
		`<p style="color:#666;margin-top:0">Hi ` + htmlEscape(display) + `, an administrator account has been created for you.</p>` +
		`<table style="border-collapse:collapse;margin:20px 0;width:100%">` +
		`<tr><td style="padding:8px 12px;background:#f5f5f5;font-weight:600;width:120px">Sign in</td>` +
		`<td style="padding:8px 12px;background:#f5f5f5"><a href="` + htmlEscape(loginURL) + `">` + htmlEscape(loginURL) + `</a></td></tr>` +
		`<tr><td style="padding:8px 12px;font-weight:600">Username</td>` +
		`<td style="padding:8px 12px;font-family:monospace">` + htmlEscape(username) + `</td></tr>` +
		`<tr><td style="padding:8px 12px;background:#f5f5f5;font-weight:600">Password</td>` +
		`<td style="padding:8px 12px;background:#f5f5f5;font-family:monospace">` + htmlEscape(tempPassword) + `</td></tr>` +
		`</table>` +
		`<p style="color:#c00;font-size:13px">Please sign in and change your password immediately.</p>` +
		`<p style="color:#999;font-size:12px;margin-top:32px">If you didn't expect this email, contact whoever added you on the Oracle team.</p>` +
		`</body></html>`

	return send(cfg, toEmail, subject, html, text)
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}
