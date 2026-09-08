package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"construct/oracle/internal/database"
	"construct/oracle/internal/models"
)

// Blog handlers. Two audiences:
//   - Public (no auth): /api/blog/posts, /api/blog/posts/{slug} — only
//     published rows, curated JSON (tags as array, reading time). The
//     lisaos.dev website fetches these at build time.
//   - Admin (staff session): /api/admin/blog/posts* — full CRUD over all
//     statuses, audit-logged on mutation.

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugSanitizer.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func readingMinutes(content string) int {
	words := len(strings.Fields(content))
	m := (words + 199) / 200
	if m < 1 {
		return 1
	}
	return m
}

func splitTags(tags string) []string {
	out := []string{}
	for t := range strings.SplitSeq(tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// excerptFor returns the post's excerpt, or a derived one from the first
// ~160 characters of content when none was set.
func excerptFor(p *models.Post) string {
	if strings.TrimSpace(p.Excerpt) != "" {
		return p.Excerpt
	}
	plain := strings.Join(strings.Fields(p.Content), " ")
	if len(plain) > 160 {
		return strings.TrimSpace(plain[:160]) + "…"
	}
	return plain
}

// publicView shapes a Post for the public website. content is included
// only on the detail endpoint to keep list payloads small.
func publicView(p *models.Post, withContent bool) map[string]any {
	v := map[string]any{
		"slug":            p.Slug,
		"title":           p.Title,
		"excerpt":         excerptFor(p),
		"cover_image":     p.CoverImage,
		"author":          p.Author,
		"tags":            splitTags(p.Tags),
		"reading_minutes": readingMinutes(p.Content),
		"published_at":    p.PublishedAt,
	}
	if withContent {
		v["content"] = p.Content
	}
	return v
}

// ─── Public ───────────────────────────────────────────────────────────────

// BlogListPublic — GET /api/blog/posts (no auth). Published posts only,
// newest first.
func BlogListPublic(w http.ResponseWriter, r *http.Request) {
	page := clampInt(atoiDefault(r.URL.Query().Get("page"), 1), 1, 1_000_000)
	pageSize := clampInt(atoiDefault(r.URL.Query().Get("pageSize"), 24), 1, 100)
	tag := strings.TrimSpace(r.URL.Query().Get("tag"))

	tx := database.Get(database.DBOracle).Model(&models.Post{}).Where("status = ?", "published")
	if tag != "" {
		tx = tx.Where("tags LIKE ?", "%"+tag+"%")
	}

	var total int64
	tx.Count(&total)

	var rows []models.Post
	if err := tx.Order("published_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "could not load posts"})
		return
	}

	posts := make([]map[string]any, 0, len(rows))
	for i := range rows {
		posts = append(posts, publicView(&rows[i], false))
	}
	WriteJSON(w, 200, map[string]any{"posts": posts, "total": total, "page": page, "pageSize": pageSize})
}

// BlogGetPublic — GET /api/blog/posts/{slug} (no auth). Published only.
func BlogGetPublic(w http.ResponseWriter, r *http.Request) {
	var p models.Post
	err := database.Get(database.DBOracle).
		Where("slug = ? AND status = ?", r.PathValue("slug"), "published").First(&p).Error
	if err != nil {
		WriteJSON(w, 404, map[string]any{"error": "not found"})
		return
	}
	WriteJSON(w, 200, publicView(&p, true))
}

// ─── Admin ──────────────────────────────────────────────────────────────

// BlogAdminList — GET /api/admin/blog/posts. All statuses, newest edit first.
func BlogAdminList(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	var rows []models.Post
	database.Get(database.DBOracle).Order("updated_at DESC").Find(&rows)
	WriteJSON(w, 200, map[string]any{"posts": rows})
}

// BlogAdminGet — GET /api/admin/blog/posts/{id}.
func BlogAdminGet(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	p, ok := loadPost(w, r)
	if !ok {
		return
	}
	WriteJSON(w, 200, p)
}

// BlogAdminCreate — POST /api/admin/blog/posts.
func BlogAdminCreate(w http.ResponseWriter, r *http.Request) {
	session := requireAdmin(w, r)
	if session == nil {
		return
	}
	body, err := parseBody(r)
	if err != nil {
		WriteJSON(w, 400, map[string]any{"error": "Invalid request body"})
		return
	}

	title := strings.TrimSpace(stringFromBody(body, "title"))
	if title == "" {
		WriteJSON(w, 400, map[string]any{"error": "title is required"})
		return
	}
	slug := slugify(stringFromBody(body, "slug"))
	if slug == "" {
		slug = slugify(title)
	}

	db := database.Get(database.DBOracle)
	var existing int64
	db.Model(&models.Post{}).Where("slug = ?", slug).Count(&existing)
	if existing > 0 {
		WriteJSON(w, 409, map[string]any{"error": "a post with this slug already exists"})
		return
	}

	author := strings.TrimSpace(stringFromBody(body, "author"))
	if author == "" {
		author = "Construct"
	}
	p := models.Post{
		Slug:       slug,
		Title:      title,
		Excerpt:    stringFromBody(body, "excerpt"),
		Content:    stringFromBody(body, "content"),
		CoverImage: stringFromBody(body, "cover_image"),
		Tags:       stringFromBody(body, "tags"),
		Author:     author,
		Status:     "draft",
	}
	if err := db.Create(&p).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "could not create post"})
		return
	}
	logAudit(r, session, "blog_post.create", "blog_post", strconv.FormatUint(uint64(p.ID), 10), map[string]any{"slug": p.Slug})
	WriteJSON(w, 201, p)
}

// BlogAdminUpdate — PATCH /api/admin/blog/posts/{id}. Only fields present
// in the body are touched.
func BlogAdminUpdate(w http.ResponseWriter, r *http.Request) {
	session := requireAdmin(w, r)
	if session == nil {
		return
	}
	p, ok := loadPost(w, r)
	if !ok {
		return
	}
	body, err := parseBody(r)
	if err != nil {
		WriteJSON(w, 400, map[string]any{"error": "Invalid request body"})
		return
	}

	db := database.Get(database.DBOracle)

	// Only update the columns present in the body. This deliberately
	// never touches status/published_at, so a content edit can't clobber
	// a concurrent (un)publish.
	updates := map[string]any{}
	if v, ok := body["slug"]; ok {
		newSlug := slugify(toString(v))
		if newSlug != "" && newSlug != p.Slug {
			var clash int64
			db.Model(&models.Post{}).Where("slug = ? AND id <> ?", newSlug, p.ID).Count(&clash)
			if clash > 0 {
				WriteJSON(w, 409, map[string]any{"error": "a post with this slug already exists"})
				return
			}
			updates["slug"] = newSlug
		}
	}
	for _, key := range []string{"title", "excerpt", "content", "cover_image", "tags", "author"} {
		if v, ok := body[key]; ok {
			updates[key] = toString(v)
		}
	}

	if len(updates) > 0 {
		if err := db.Model(p).Updates(updates).Error; err != nil {
			WriteJSON(w, 500, map[string]any{"error": "could not update post"})
			return
		}
		db.First(p, p.ID) // reload so the response reflects persisted state
	}
	logAudit(r, session, "blog_post.update", "blog_post", strconv.FormatUint(uint64(p.ID), 10), map[string]any{"slug": p.Slug})
	WriteJSON(w, 200, p)
}

// BlogAdminDelete — DELETE /api/admin/blog/posts/{id}.
func BlogAdminDelete(w http.ResponseWriter, r *http.Request) {
	session := requireSuperAdmin(w, r)
	if session == nil {
		return
	}
	p, ok := loadPost(w, r)
	if !ok {
		return
	}
	if err := database.Get(database.DBOracle).Delete(p).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "could not delete post"})
		return
	}
	logAudit(r, session, "blog_post.delete", "blog_post", strconv.FormatUint(uint64(p.ID), 10), map[string]any{"slug": p.Slug})
	WriteJSON(w, 200, map[string]any{"ok": true})
}

// BlogAdminPublish — POST /api/admin/blog/posts/{id}/publish.
func BlogAdminPublish(w http.ResponseWriter, r *http.Request) {
	session := requireAdmin(w, r)
	if session == nil {
		return
	}
	p, ok := loadPost(w, r)
	if !ok {
		return
	}
	p.Status = "published"
	if p.PublishedAt == nil {
		now := time.Now()
		p.PublishedAt = &now
	}
	if err := database.Get(database.DBOracle).Save(p).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "could not publish post"})
		return
	}
	logAudit(r, session, "blog_post.publish", "blog_post", strconv.FormatUint(uint64(p.ID), 10), map[string]any{"slug": p.Slug})
	WriteJSON(w, 200, p)
}

// BlogAdminUnpublish — POST /api/admin/blog/posts/{id}/unpublish. Reverts to
// draft so it disappears from the public site on the next build.
func BlogAdminUnpublish(w http.ResponseWriter, r *http.Request) {
	session := requireAdmin(w, r)
	if session == nil {
		return
	}
	p, ok := loadPost(w, r)
	if !ok {
		return
	}
	p.Status = "draft"
	if err := database.Get(database.DBOracle).Save(p).Error; err != nil {
		WriteJSON(w, 500, map[string]any{"error": "could not unpublish post"})
		return
	}
	logAudit(r, session, "blog_post.unpublish", "blog_post", strconv.FormatUint(uint64(p.ID), 10), map[string]any{"slug": p.Slug})
	WriteJSON(w, 200, p)
}

// ─── helpers ──────────────────────────────────────────────────────────────

func loadPost(w http.ResponseWriter, r *http.Request) (*models.Post, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]any{"error": "invalid id"})
		return nil, false
	}
	var p models.Post
	if err := database.Get(database.DBOracle).First(&p, id).Error; err != nil {
		WriteJSON(w, 404, map[string]any{"error": "not found"})
		return nil, false
	}
	return &p, true
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
