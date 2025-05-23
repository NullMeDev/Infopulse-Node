// internal/models/models.go
package models

import (
	"time"
)

// Category represents a topic category for intelligence items
type Category string

const (
	CategoryCybersec    Category = "CYBERSEC"
	CategoryAITools     Category = "AITOOLS"
	CategoryOpenSource  Category = "OPENSOURCE"
	CategoryInfosecNews Category = "INFOSEC_NEWS"
)

// Intelligence represents an intelligence item
type Intelligence struct {
	ID        string    `json:"id"`        // Unique identifier
	SourceID  string    `json:"sourceId"`  // ID of the source feed
	Category  Category  `json:"category"`  // Primary category
	Title     string    `json:"title"`     // Title of the item
	URL       string    `json:"url"`       // URL to the original content
	Summary   string    `json:"summary"`   // Summary or excerpt
	Published time.Time `json:"published"` // Original publication date
	Retrieved time.Time `json:"retrieved"` // When the item was retrieved
	Hash      string    `json:"hash"`      // Hash for deduplication
	Severity  string    `json:"severity"`  // Severity (for CVEs and vulnerabilities)
}

// FeedSource represents a source of intelligence
type FeedSource struct {
	ID         string     `json:"id"`         // Unique identifier
	Name       string     `json:"name"`       // Display name
	URL        string     `json:"url"`        // URL to fetch
	Categories []Category `json:"categories"` // Categories this feed belongs to
	FetchMethod string    `json:"fetchMethod"` // Method used to fetch (rss, api, etc.)
	UpdateFreq int        `json:"updateFreq"`  // Update frequency in minutes
	Enabled    bool       `json:"enabled"`     // Whether this feed is enabled
}
