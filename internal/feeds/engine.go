// internal/feeds/engine.go
package feeds

import (
	"fmt"
	"sync"
	"time"

	"github.com/NullMeDev/Infopulse-Node/internal/config"
	"github.com/NullMeDev/Infopulse-Node/internal/logger"
	"github.com/NullMeDev/Infopulse-Node/internal/models"
)

// Engine manages feed fetching and processing
type Engine struct {
	config   *config.Config
	parser   *Parser
	store    *Store
	logger   *logger.Logger
	sources  []models.FeedSource
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewEngine creates a new feed engine
func NewEngine(cfg *config.Config, logger *logger.Logger) (*Engine, error) {
	// Create parser
	parser := NewParser(cfg.FetchTimeoutSeconds, logger)

	// Create store
	store, err := NewStore(cfg.DBFilePath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %v", err)
	}

	// Create engine
	engine := &Engine{
		config:   cfg,
		parser:   parser,
		store:    store,
		logger:   logger,
		sources:  cfg.FeedSources,
		stopChan: make(chan struct{}),
	}

	return engine, nil
}

// Start starts the feed engine
func (e *Engine) Start() error {
	e.logger.Info("Engine", "Starting feed engine")

	// Start update loop
	e.wg.Add(1)
	go e.updateLoop()

	return nil
}

// Stop stops the feed engine
func (e *Engine) Stop() error {
	e.logger.Info("Engine", "Stopping feed engine")

	// Signal all goroutines to stop
	close(e.stopChan)

	// Wait for all goroutines to finish
	e.wg.Wait()

	// Close store
	return e.store.Close()
}

// updateLoop periodically updates feeds
func (e *Engine) updateLoop() {
	defer e.wg.Done()

	// Immediate first update
	e.updateAllFeeds()

	// Set up ticker for periodic updates
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.updateAllFeeds()
		case <-e.stopChan:
			e.logger.Info("Engine", "Update loop stopped")
			return
		}
	}
}

// updateAllFeeds updates all configured feeds
func (e *Engine) updateAllFeeds() {
	e.logger.Info("Engine", fmt.Sprintf("Updating %d feeds", len(e.sources)))

	// Create worker pool
	type Job struct {
		source models.FeedSource
	}
	type Result struct {
		source models.FeedSource
		items  []*models.Intelligence
		err    error
	}

	jobs := make(chan Job, len(e.sources))
	results := make(chan Result, len(e.sources))
	
	// Create workers
	var workersWg sync.WaitGroup
	workerCount := e.config.MaxConcurrentFetches
	if workerCount <= 0 {
		workerCount = 5 // Default to 5 workers
	}

	for i := 0; i < workerCount; i++ {
		workersWg.Add(1)
		go func() {
			defer workersWg.Done()
			
			for job := range jobs {
				// Fetch and parse feed
				items, err := e.parser.ParseFeed(job.source)
				results <- Result{
					source: job.source,
					items:  items,
					err:    err,
				}
			}
		}()
	}

	// Queue jobs
	for _, source := range e.sources {
		if !source.Enabled {
			continue
		}
		jobs <- Job{source: source}
	}
	close(jobs)

	// Process results in a separate goroutine
	var processWg sync.WaitGroup
	processWg.Add(1)
	go func() {
		defer processWg.Done()
		
		totalItems := 0
		savedItems := 0
		
		for i := 0; i < len(e.sources); i++ {
			result := <-results
			if result.err != nil {
				e.logger.Error("Engine", fmt.Sprintf("Failed to update feed %s: %v", result.source.Name, result.err))
				continue
			}
			
			totalItems += len(result.items)
			count, err := e.store.SaveIntelligence(result.items)
			if err != nil {
				e.logger.Error("Engine", fmt.Sprintf("Failed to save items from %s: %v", result.source.Name, err))
				continue
			}
			
			savedItems += count
			if count > 0 {
				e.logger.Info("Engine", fmt.Sprintf("Saved %d/%d new items from %s", count, len(result.items), result.source.Name))
			}
		}
		
		e.logger.Info("Engine", fmt.Sprintf("Feed update complete. Processed %d items, saved %d new items", totalItems, savedItems))
	}()

	// Wait for workers to finish
	workersWg.Wait()
	close(results)
	
	// Wait for processing to finish
	processWg.Wait()
}

// RefreshFeeds forces a refresh of all feeds
func (e *Engine) RefreshFeeds() {
	go e.updateAllFeeds()
}

// GetLatestIntel returns the latest intelligence items
func (e *Engine) GetLatestIntel(category models.Category, limit int) []*models.Intelligence {
	items, err := e.store.GetLatestIntelligence(category, limit)
	if err != nil {
		e.logger.Error("Engine", fmt.Sprintf("Failed to get latest intelligence: %v", err))
		return []*models.Intelligence{}
	}
	return items
}

// GetIntelByID gets an intelligence item by ID
func (e *Engine) GetIntelByID(id string) *models.Intelligence {
	item, err := e.store.GetIntelligenceByID(id)
	if err != nil {
		e.logger.Error("Engine", fmt.Sprintf("Failed to get intelligence by ID: %v", err))
		return nil
	}
	return item
}

// GetTotalCount gets the total count of intelligence items
func (e *Engine) GetTotalCount() int {
	count, err := e.store.GetTotalCount()
	if err != nil {
		e.logger.Error("Engine", fmt.Sprintf("Failed to get total count: %v", err))
		return 0
	}
	return count
}
