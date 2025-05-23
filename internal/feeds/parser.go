// internal/feeds/parser.go
package feeds

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NullMeDev/Infopulse-Node/internal/logger"
	"github.com/NullMeDev/Infopulse-Node/internal/models"
	"github.com/mmcdole/gofeed"
)

// Parser handles parsing feed content
type Parser struct {
	feedParser *gofeed.Parser
	client     *http.Client
	logger     *logger.Logger
}

// NewParser creates a new feed parser
func NewParser(timeoutSeconds int, logger *logger.Logger) *Parser {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	return &Parser{
		feedParser: gofeed.NewParser(),
		client:     client,
		logger:     logger,
	}
}

// ParseFeed fetches and parses a feed source
func (p *Parser) ParseFeed(source models.FeedSource) ([]*models.Intelligence, error) {
	p.logger.Info("Parser", fmt.Sprintf("Fetching feed: %s (%s)", source.Name, source.URL))

	var items []*models.Intelligence

	// Handle different fetch methods
	switch strings.ToLower(source.FetchMethod) {
	case "rss":
		parsedItems, err := p.parseRSS(source)
		if err != nil {
			return nil, err
		}
		items = parsedItems
	// Add other fetch methods here as needed
	default:
		return nil, fmt.Errorf("unsupported fetch method: %s", source.FetchMethod)
	}

	p.logger.Info("Parser", fmt.Sprintf("Parsed %d items from %s", len(items), source.Name))
	return items, nil
}

// parseRSS fetches and parses an RSS feed
func (p *Parser) parseRSS(source models.FeedSource) ([]*models.Intelligence, error) {
	// Fetch the feed content
	resp, err := p.client.Get(source.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch feed: HTTP %d", resp.StatusCode)
	}

	// Parse the feed
	feed, err := p.feedParser.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %v", err)
	}

	// Process items
	var items []*models.Intelligence
	now := time.Now().UTC()

	for _, feedItem := range feed.Items {
		// Create intelligence item
		item := &models.Intelligence{
			SourceID:  source.ID,
			Title:     feedItem.Title,
			URL:       feedItem.Link,
			Retrieved: now,
			Category:  source.Categories[0], // Default to first category
		}

		// Set summary
		if feedItem.Description != "" {
			item.Summary = cleanSummary(feedItem.Description)
		} else if feedItem.Content != "" {
			item.Summary = cleanSummary(feedItem.Content)
		}

		// Set published date
		if feedItem.PublishedParsed != nil {
			item.Published = *feedItem.PublishedParsed
		} else if feedItem.UpdatedParsed != nil {
			item.Published = *feedItem.UpdatedParsed
		} else {
			item.Published = now
		}

		// Generate ID and hash
		item.ID = generateID(item)
		item.Hash = generateHash(item)

		// Set severity if present
		item.Severity = extractSeverity(feedItem)

		items = append(items, item)
	}

	return items, nil
}

// cleanSummary cleans HTML and truncates the summary
func cleanSummary(text string) string {
	// TODO: Implement better HTML cleaning
	// For now, do a simple truncation
	if len(text) > 500 {
		return text[:497] + "..."
	}
	return text
}

// generateID creates a unique ID for an item
func generateID(item *models.Intelligence) string {
	// Use URL as basis for ID if available
	if item.URL != "" {
		hash := md5.Sum([]byte(item.URL))
		return fmt.Sprintf("%x", hash)[:12]
	}

	// Fallback to title and timestamp
	hash := md5.Sum([]byte(item.Title + item.Published.String()))
	return fmt.Sprintf("%x", hash)[:12]
}

// generateHash creates a hash for deduplication
func generateHash(item *models.Intelligence) string {
	hash := md5.Sum([]byte(item.Title + item.URL))
	return fmt.Sprintf("%x", hash)
}

// extractSeverity extracts severity information from feed item
func extractSeverity(feedItem *gofeed.Item) string {
	// Check for CVE severity in title
	title := strings.ToUpper(feedItem.Title)
	
	if strings.Contains(title, "CRITICAL") {
		return "CRITICAL"
	} else if strings.Contains(title, "HIGH") {
		return "HIGH"
	} else if strings.Contains(title, "MEDIUM") {
		return "MEDIUM"
	} else if strings.Contains(title, "LOW") {
		return "LOW"
	}
	
	// TODO: Implement better CVE severity extraction
	
	return ""
}
