package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // SQLite driver
)

// Storage holds the database connection and provides access to repositories.
type Storage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite-backed storage.
// The dataSourceName can be a file path or ":memory:" for an in-memory database.
func NewSQLiteStorage(dataSourceName string) (*Storage, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	s := &Storage{db: db}
	if err := s.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

// DB returns the underlying database connection.
func (s *Storage) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// runMigrations creates all required tables.
func (s *Storage) runMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS booklets (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			type TEXT,
			version TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			audience TEXT,
			instructions TEXT,
			template TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sections (
			id TEXT NOT NULL,
			booklet_id TEXT NOT NULL,
			parent_id TEXT,
			title TEXT NOT NULL,
			level INTEGER NOT NULL,
			prompt TEXT,
			content TEXT,
			dependencies TEXT,
			context_refs TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			provider TEXT,
			model TEXT,
			prompt_snapshot TEXT,
			temperature REAL,
			max_tokens INTEGER,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			duration_ms INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (id, booklet_id)
		)`,
		`CREATE TABLE IF NOT EXISTS content_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			section_id TEXT NOT NULL,
			booklet_id TEXT NOT NULL,
			version INTEGER NOT NULL,
			content TEXT NOT NULL,
			status TEXT NOT NULL,
			description TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS booklet_references (
			id TEXT PRIMARY KEY,
			booklet_id TEXT NOT NULL,
			depends_on TEXT,
			description TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_booklet_references_id ON booklet_references(id)`,
		`CREATE INDEX IF NOT EXISTS idx_booklet_references_booklet_id ON booklet_references(booklet_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sections_booklet_id ON sections(booklet_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_versions_section_id ON content_versions(section_id, booklet_id)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
