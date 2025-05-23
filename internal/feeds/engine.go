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

// Engine coordinates feed operations
type Engine struct {
	config     *config.Config
	parser     *Parser
	store      *Store
	logger     *logger.Logger
	mu         sync.Mutex
	isRunning  bool
	stopChan   chan struct{}
	waitGroup  sync.WaitGroup
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

	return &Engine{
		config:    cfg,
		parser:    parser,
		store:     store,
		logger:    logger,
		stopChan:  make(chan struct{}),
	}, nil
}

// Start begins feed processing
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return fmt.Errorf("engine is already running")
	}

	e.isRunning = true
	e.stopChan = make(chan struct{})

	// Start a goroutine to manage feed updates
	e.waitGroup.Add(1)
	go e.run()

	e.logger.Info("Engine", "Feed engine started")
	return nil
}

// Stop stops feed processing
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return nil
	}

	// Signal all workers to stop
	close(e.stopChan)

	// Wait for all workers to finish
	e.waitGroup.Wait()

	// Save store before exiting
	if err := e.store.Save(); err != nil {
		e.logger.Error("Engine", fmt.Sprintf("Failed to save store: %v", err))
	}

	e.isRunning = false
	e.logger.Info("Engine", "Feed engine stopped")
	return nil
}

// run is the main engine loop
func (e *Engine) run() {
	defer e.waitGroup.Done()

	// First run immediately
	e.fetchAllFeeds()

	// Set up ticker for periodic updates
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.fetchAllFeeds()
		case <-e.stopChan:
			return
		}
	}
}

// fetchAllFeeds processes all configured feed sources
func (e *Engine) fetchAllFeeds() {
	e.logger.Info("Engine", "Starting feed update cycle")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, e.config.MaxConcurrentFetches)

	for _, source := range e.config.FeedSources {
		if !source.Enabled {
			continue
		}

		// Check if it's time to update this feed
		// TODO: Implement proper update frequency tracking

		// Acquire semaphore slot
		semaphore <- struct{}{}
		wg.Add(1)

		// Start worker goroutine
		go func(src models.FeedSource) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore slot

			// Fetch and process feed
			e.processFeed(src)
		}(source)
	}

	// Wait for all workers to finish
	wg.Wait()
	
	// Save store after updates
	if err := e.store.Save(); err != nil {
		e.logger.Error("Engine", fmt.Sprintf("Failed to save store: %v", err))
	}

	// Cleanup old items (keep items for 30 days)
	e.store.Cleanup(30 * 24 * time.Hour)
	
	e.logger.Info("Engine", "Finished feed update cycle")
}

// processFeed fetches and processes a single feed
func (e *Engine) processFeed(source models.FeedSource) {
	e.logger.Info("Engine", fmt.Sprintf("Processing feed: %s", source.Name))

	// Parse feed
	items, err := e.parser.ParseFeed(source)
	if err != nil {
		e.logger.Error("Engine", fmt.Sprintf("Failed to parse feed %s: %v", source.Name, err))
		return
	}

	// Process and store items
	var newItems int
	for _, item := range items {
		// Add to store (only counts as new if not a duplicate)
		isNew, err := e.store.AddItem(item)
		if err != nil {
			e.logger.Error("Engine", fmt.Sprintf("Failed to add item %s: %v", item.Title, err))
			continue
		}

		if isNew {
			newItems++
			e.logger.Info("Engine", fmt.Sprintf("New item: %s", item.Title))
			
			// TODO: Trigger notifications for new items
		}
	}

	e.logger.Info("Engine", fmt.Sprintf("Processed feed %s: %d items, %d new", 
		source.Name, len(items), newItems))
}

// GetLatestIntel returns the latest intelligence items
func (e *Engine) GetLatestIntel(category models.Category, limit int) []*models.Intelligence {
	if category != "" {
		return e.store.GetItemsByCategory(category, limit)
	}
	return e.store.GetLatestItems(limit)
}

// GetTotalCount returns the total number of intelligence items
func (e *Engine) GetTotalCount() int {
	return e.store.GetTotalCount()
}
