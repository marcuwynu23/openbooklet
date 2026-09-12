package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// Storage holds the database connection and provides access to repositories.
type Storage struct {
	db     *gorm.DB
	driver Driver
}

// OpenStorage connects to the given backend and runs migrations.
func OpenStorage(driver Driver, dataSourceName string) (*Storage, error) {
	db, err := Open(driver, dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			sqlDB.Close()
		}
		return nil, err
	}
	return &Storage{db: db, driver: driver}, nil
}

// NewSQLiteStorage creates a SQLite-backed storage.
// The dataSourceName can be a file path or ":memory:" for an in-memory database.
func NewSQLiteStorage(dataSourceName string) (*Storage, error) {
	return OpenStorage(DriverSQLite, dataSourceName)
}

// DB returns the underlying database connection.
func (s *Storage) DB() *gorm.DB {
	return s.db
}

// DriverName reports which backend this storage uses.
func (s *Storage) DriverName() string {
	return string(s.driver)
}

// Close closes the database connection.
func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("resolving connection: %w", err)
	}
	return sqlDB.Close()
}
