// internal/models/models.go
package models

import (
    "time"
)

// Category represents intelligence categories
type Category string

const (
    CategoryCybersec    Category = "cybersec"
    CategoryAITools     Category = "ai-tools"
    CategoryOpenSource  Category = "opensource"
    CategoryInfosecNews Category = "infosec-news"
)

// FeedSource represents a configured intelligence source
type FeedSource struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    URL         string     `json:"url"`
    Categories  []Category `json:"categories"`
    FetchMethod string     `json:"fetchMethod"` // "rss", "api", etc.
    UpdateFreq  int        `json:"updateFrequencyMinutes"`
    Enabled     bool       `json:"enabled"`
}

// Intelligence represents processed intelligence ready for output
type Intelligence struct {
    ID          string    `json:"id"`
    SourceID    string    `json:"sourceId"`
    Category    Category  `json:"category"`
    Title       string    `json:"title"`
    URL         string    `json:"url"`
    Summary     string    `json:"summary"`
    Published   time.Time `json:"published"`
    Retrieved   time.Time `json:"retrieved"`
    Hash        string    `json:"hash"` // Deduplication hash
    Sentiment   int8      `json:"sentiment"` // -1, 0, 1 for future sentiment analysis
    Entities    []Entity  `json:"entities,omitempty"`
    Severity    string    `json:"severity,omitempty"` // For CVEs and security alerts
}

// Entity represents a named entity extracted from intelligence
type Entity struct {
    Type  string `json:"type"` // organization, person, technology, threat-actor, etc.
    Name  string `json:"name"`
    Count int    `json:"count"` // Number of occurrences
}

// LogEntry represents a local log entry
type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Source    string    `json:"source"`
    Category  Category  `json:"category,omitempty"`
    Title     string    `json:"title,omitempty"`
    URL       string    `json:"url,omitempty"`
    Message   string    `json:"message"`
    Sentiment int8      `json:"sentiment,omitempty"`
}
