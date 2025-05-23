// internal/feeds/store.go
package feeds

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/NullMeDev/Infopulse-Node/internal/logger"
	"github.com/NullMeDev/Infopulse-Node/internal/models"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Store handles persistence of intelligence data
type Store struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewStore creates a new store instance
func NewStore(dbPath string, logger *logger.Logger) (*Store, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Set connection parameters
	db.SetMaxOpenConns(1) // SQLite supports only one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	store := &Store{
		db:     db,
		logger: logger,
	}

	// Initialize database
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// initialize creates tables if they don't exist
func (s *Store) initialize() error {
	// Create intelligence table
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS intelligence (
		id TEXT PRIMARY KEY,
		source_id TEXT NOT NULL,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		url TEXT NOT NULL,
		summary TEXT,
		published TIMESTAMP NOT NULL,
		retrieved TIMESTAMP NOT NULL,
		hash TEXT NOT NULL,
		severity TEXT
	)`)
	if err != nil {
		return fmt.Errorf("failed to create intelligence table: %v", err)
	}

	// Create indices
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_intelligence_hash ON intelligence(hash)`)
	if err != nil {
		return fmt.Errorf("failed to create hash index: %v", err)
	}

	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_intelligence_category ON intelligence(category)`)
	if err != nil {
		return fmt.Errorf("failed to create category index: %v", err)
	}

	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_intelligence_published ON intelligence(published)`)
	if err != nil {
		return fmt.Errorf("failed to create published index: %v", err)
	}

	s.logger.Info("Store", "Database initialized")
	return nil
}

// SaveIntelligence saves intelligence items to the database
func (s *Store) SaveIntelligence(items []*models.Intelligence) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.Prepare(`
	INSERT OR IGNORE INTO intelligence 
	(id, source_id, category, title, url, summary, published, retrieved, hash, severity)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	// Insert items
	count := 0
	for _, item := range items {
		_, err := stmt.Exec(
			item.ID,
			item.SourceID,
			item.Category,
			item.Title,
			item.URL,
			item.Summary,
			item.Published,
			item.Retrieved,
			item.Hash,
			item.Severity,
		)
		if err != nil {
			s.logger.Error("Store", fmt.Sprintf("Failed to insert item: %v", err))
			continue
		}
		count++
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %v", err)
	}

	s.logger.Info("Store", fmt.Sprintf("Inserted %d intelligence items", count))
	return count, nil
}

// GetIntelligenceByID retrieves an intelligence item by ID
func (s *Store) GetIntelligenceByID(id string) (*models.Intelligence, error) {
	row := s.db.QueryRow(`
	SELECT id, source_id, category, title, url, summary, published, retrieved, hash, severity
	FROM intelligence
	WHERE id = ?`, id)

	item := &models.Intelligence{}
	err := row.Scan(
		&item.ID,
		&item.SourceID,
		&item.Category,
		&item.Title,
		&item.URL,
		&item.Summary,
		&item.Published,
		&item.Retrieved,
		&item.Hash,
		&item.Severity,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No item found
		}
		return nil, fmt.Errorf("failed to query intelligence: %v", err)
	}

	return item, nil
}

// GetLatestIntelligence retrieves the latest intelligence items
func (s *Store) GetLatestIntelligence(category models.Category, limit int) ([]*models.Intelligence, error) {
	var rows *sql.Rows
	var err error

	if category == "" {
		// Query all categories
		rows, err = s.db.Query(`
		SELECT id, source_id, category, title, url, summary, published, retrieved, hash, severity
		FROM intelligence
		ORDER BY published DESC
		LIMIT ?`, limit)
	} else {
		// Query specific category
		rows, err = s.db.Query(`
		SELECT id, source_id, category, title, url, summary, published, retrieved, hash, severity
		FROM intelligence
		WHERE category = ?
		ORDER BY published DESC
		LIMIT ?`, category, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query intelligence: %v", err)
	}
	defer rows.Close()

	var items []*models.Intelligence
	for rows.Next() {
		item := &models.Intelligence{}
		err := rows.Scan(
			&item.ID,
			&item.SourceID,
			&item.Category,
			&item.Title,
			&item.URL,
			&item.Summary,
			&item.Published,
			&item.Retrieved,
			&item.Hash,
			&item.Severity,
		)
		if err != nil {
			s.logger.Error("Store", fmt.Sprintf("Failed to scan row: %v", err))
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// GetTotalCount gets the total count of intelligence items
func (s *Store) GetTotalCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM intelligence").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %v", err)
	}
	return count, nil
}

// GetCategoryCount gets the count of intelligence items by category
func (s *Store) GetCategoryCount(category models.Category) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM intelligence WHERE category = ?", category).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %v", err)
	}
	return count, nil
}
