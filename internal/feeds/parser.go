// internal/feeds/parser.go
package feeds

import (
	"context"
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

// Parser handles fetching and parsing feeds
type Parser struct {
	client  *http.Client
	logger  *logger.Logger
	timeout time.Duration
}

// NewParser creates a new feed parser
func NewParser(timeout int, logger *logger.Logger) *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		logger:  logger,
		timeout: time.Duration(timeout) * time.Second,
	}
}

// ParseFeed fetches and parses a feed from the given source
func (p *Parser) ParseFeed(source models.FeedSource) ([]*models.Intelligence, error) {
	p.logger.Info("Parser", fmt.Sprintf("Fetching feed: %s (%s)", source.Name, source.URL))

	switch source.FetchMethod {
	case "rss":
		return p.parseRSSFeed(source)
	default:
		return nil, fmt.Errorf("unsupported feed method: %s", source.FetchMethod)
	}
}

// parseRSSFeed fetches and parses an RSS feed
func (p *Parser) parseRSSFeed(source models.FeedSource) ([]*models.Intelligence, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	
	req.Header.Set("User-Agent", "Infopulse-Node/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch feed, status code: %d", resp.StatusCode)
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %v", err)
	}

	var items []*models.Intelligence
	for _, item := range feed.Items {
		if item.Title == "" || item.Link == "" {
			continue
		}

		intel := &models.Intelligence{
			ID:        generateID(source.ID, item.GUID),
			SourceID:  source.ID,
			Category:  getDefaultCategory(source.Categories),
			Title:     item.Title,
			URL:       item.Link,
			Summary:   getSummary(item),
			Hash:      generateHash(item.Title, item.Link, item.Description),
		}

		if item.PublishedParsed != nil {
			intel.Published = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			intel.Published = *item.UpdatedParsed
		} else {
			intel.Published = time.Now()
		}

		intel.Retrieved = time.Now()

		if containsCVE(item.Title) || containsCVE(item.Description) {
			intel.Severity = estimateSeverity(item.Title, item.Description)
		}

		items = append(items, intel)
	}

	p.logger.Info("Parser", fmt.Sprintf("Fetched %d items from %s", len(items), source.Name))
	return items, nil
}

// Helper to generate a unique ID for an intelligence item
func generateID(sourceID, guid string) string {
	if guid == "" {
		return fmt.Sprintf("%s-%d", sourceID, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%x", sourceID, md5.Sum([]byte(guid)))
}

// Helper to get default category for an intelligence item
func getDefaultCategory(categories []models.Category) models.Category {
	if len(categories) > 0 {
		return categories[0]
	}
	return models.CategoryInfosecNews
}

// Helper to generate a hash for deduplication
func generateHash(title, link, description string) string {
	h := md5.New()
	io.WriteString(h, title)
	io.WriteString(h, link)
	io.WriteString(h, description)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Helper to extract summary from item
func getSummary(item *gofeed.Item) string {
	if item.Description != "" {
		return stripHTMLAndTruncate(item.Description, 500)
	}
	
	if item.Content != "" {
		return stripHTMLAndTruncate(item.Content, 500)
	}
	
	return item.Title
}

// Helper to strip HTML and truncate text
func stripHTMLAndTruncate(input string, maxLength int) string {
	text := strings.ReplaceAll(input, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	
	for strings.Contains(text, "<") && strings.Contains(text, ">") {
		start := strings.Index(text, "<")
		end := strings.Index(text[start:], ">") + start
		if end > start {
			text = text[:start] + text[end+1:]
		} else {
			break
		}
	}
	
	if len(text) > maxLength {
		return text[:maxLength] + "..."
	}
	
	return text
}

// Helper to check if text contains CVE references
func containsCVE(text string) bool {
	return strings.Contains(strings.ToUpper(text), "CVE-")
}

// Helper to estimate severity from CVE information
func estimateSeverity(title, description string) string {
	text := strings.ToLower(title + " " + description)
	
	if strings.Contains(text, "critical") {
		return "CRITICAL"
	}
	if strings.Contains(text, "high") {
		return "HIGH"
	}
	if strings.Contains(text, "medium") {
		return "MEDIUM"
	}
	if strings.Contains(text, "low") {
		return "LOW"
	}
	
	return "MEDIUM"
}
