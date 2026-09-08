package models

import "time"

// Post is a blog article authored in oracle and served to the public
// website at lisaos.dev/blog. Content is Markdown — the website
// renders it at build time. Only rows with Status=="published" are
// exposed by the public list/detail endpoints; drafts are admin-only.
//
// Tags is stored as a comma-separated string for DB-portability (oracle
// runs on either MySQL or Postgres, so no pg array types). The public
// API splits it into a JSON array; the admin API returns the raw string
// so the editor can round-trip it.
type Post struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Slug        string     `gorm:"size:160;uniqueIndex;not null" json:"slug"`
	Title       string     `gorm:"size:200;not null" json:"title"`
	Excerpt     string     `gorm:"size:500" json:"excerpt"`
	Content     string     `gorm:"type:text" json:"content"`
	CoverImage  string     `gorm:"size:500" json:"cover_image"`
	Tags        string     `gorm:"size:300" json:"tags"`
	Author      string     `gorm:"size:120" json:"author"`
	Status      string     `gorm:"size:20;default:draft;not null;index" json:"status"`
	PublishedAt *time.Time `gorm:"index" json:"published_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

func (Post) TableName() string { return "posts" }
