package storage

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Driver selects the SQL backend. SQLite is the local-first default;
// Postgres and MySQL support shared or server deployments.
type Driver string

const (
	DriverSQLite   Driver = "sqlite"
	DriverPostgres Driver = "postgres"
	DriverMySQL    Driver = "mysql"
)

// ParseDriver resolves a driver name, defaulting blank to SQLite.
func ParseDriver(name string) (Driver, error) {
	switch Driver(name) {
	case "", DriverSQLite:
		return DriverSQLite, nil
	case DriverPostgres:
		return DriverPostgres, nil
	case DriverMySQL:
		return DriverMySQL, nil
	default:
		return "", fmt.Errorf("unknown database driver %q: want sqlite, postgres, or mysql", name)
	}
}

// Open connects to the given backend. The DSN is driver-specific: a file
// path (or ":memory:") for SQLite, a URL for Postgres, and a DSN string
// for MySQL.
func Open(driver Driver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case DriverPostgres:
		dialector = postgres.Open(dsn)
	case DriverMySQL:
		dialector = mysql.Open(dsn)
	default:
		dialector = sqlite.Open(dsn)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("opening %s database: %w", driver, err)
	}
	return db, nil
}

// DriverName reports the dialect of an open database.
func DriverName(db *gorm.DB) string {
	if db == nil {
		return ""
	}
	return string(db.Dialector.Name())
}
