// internal/feeds/store.go
package feeds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/NullMeDev/Infopulse-Node/internal/logger"
	"github.com/NullMeDev/Infopulse-Node/internal/models"
)

// Store manages intelligence data persistence
type Store struct {
	mu              sync.RWMutex
	items           map[string]*models.Intelligence // Map of intelligence items by ID
	itemsByCategory map[models.Category][]*models.Intelligence
	itemsByHash     map[string]*models.Intelligence // For deduplication
	logger          *logger.Logger
	filePath        string
	lastUpdate      time.Time
}

// NewStore creates a new intelligence store
func NewStore(filePath string, logger *logger.Logger) (*Store, error) {
	store := &Store{
		items:           make(map[string]*models.Intelligence),
		itemsByCategory: make(map[models.Category][]*models.Intelligence),
		itemsByHash:     make(map[string]*models.Intelligence),
		logger:          logger,
		filePath:        filePath,
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %v", err)
	}

	// Load existing items if file exists
	if err := store.load(); err != nil {
		logger.Warning("Store", fmt.Sprintf("Failed to load store: %v", err))
		// Continue even if loading fails
	}

	return store, nil
}

// AddItem adds a new intelligence item to the store
func (s *Store) AddItem(item *models.Intelligence) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates by hash
	if existing, exists := s.itemsByHash[item.Hash]; exists {
		s.logger.Debug("Store", fmt.Sprintf("Duplicate item found: %s", item.Title))
		return false, nil
	}

	// Add to maps
	s.items[item.ID] = item
	s.itemsByHash[item.Hash] = item
	
	// Add to category map
	s.itemsByCategory[item.Category] = append(s.itemsByCategory[item.Category], item)

	// Update last modified time
	s.lastUpdate = time.Now()

	return true, nil
}

// GetItemsByCategory returns all intelligence items of a specific category
func (s *Store) GetItemsByCategory(category models.Category, limit int) []*models.Intelligence {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get items for the category
	items := s.itemsByCategory[category]
	
	// Apply limit if needed
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	
	return items
}

// GetLatestItems returns the most recent intelligence items across all categories
func (s *Store) GetLatestItems(limit int) []*models.Intelligence {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a slice with all items
	var allItems []*models.Intelligence
	for _, item := range s.items {
		allItems = append(allItems, item)
	}

	// Sort by published date (newest first)
	// Note: In a real implementation, we would use proper sorting here
	// For simplicity, we're assuming the items are already sorted
	
	// Apply limit if needed
	if limit > 0 && len(allItems) > limit {
		return allItems[:limit]
	}
	
	return allItems
}

// Save persists the intelligence store to disk
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a slice with all items
	var allItems []*models.Intelligence
	for _, item := range s.items {
		allItems = append(allItems, item)
	}

	// Open file for writing
	file, err := os.Create(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to create store file: %v", err)
	}
	defer file.Close()

	// Write data
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(allItems); err != nil {
		return fmt.Errorf("failed to encode store: %v", err)
	}

	s.logger.Info("Store", fmt.Sprintf("Saved %d items to %s", len(allItems), s.filePath))
	return nil
}

// load reads the intelligence store from disk
func (s *Store) load() error {
	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		s.logger.Info("Store", "Store file does not exist, starting fresh")
		return nil
	}

	// Open file for reading
	file, err := os.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to open store file: %v", err)
	}
	defer file.Close()

	// Read data
	var allItems []*models.Intelligence
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&allItems); err != nil {
		return fmt.Errorf("failed to decode store: %v", err)
	}

	// Load data into maps
	for _, item := range allItems {
		s.items[item.ID] = item
		s.itemsByHash[item.Hash] = item
		s.itemsByCategory[item.Category] = append(s.itemsByCategory[item.Category], item)
	}

	s.logger.Info("Store", fmt.Sprintf("Loaded %d items from %s", len(allItems), s.filePath))
	return nil
}

// GetItem returns an intelligence item by ID
func (s *Store) GetItem(id string) (*models.Intelligence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[id]
	return item, exists
}

// GetTotalCount returns the total number of intelligence items
func (s *Store) GetTotalCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Cleanup removes old intelligence items
func (s *Store) Cleanup(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var removedCount int

	// Find items to remove
	var toRemove []string
	for id, item := range s.items {
		if item.Published.Before(cutoff) {
			toRemove = append(toRemove, id)
		}
	}

	// Remove items
	for _, id := range toRemove {
		item := s.items[id]
		
		// Remove from maps
		delete(s.items, id)
		delete(s.itemsByHash, item.Hash)
		
		// Remove from category map
		var newItems []*models.Intelligence
		for _, i := range s.itemsByCategory[item.Category] {
			if i.ID != id {
				newItems = append(newItems, i)
			}
		}
		s.itemsByCategory[item.Category] = newItems
		
		removedCount++
	}

	// Update last modified time if items were removed
	if removedCount > 0 {
		s.lastUpdate = time.Now()
	}

	s.logger.Info("Store", fmt.Sprintf("Cleaned up %d items older than %v", removedCount, maxAge))
	return removedCount
}
