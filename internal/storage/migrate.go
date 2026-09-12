package storage

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Migration is one versioned schema or data change. Migrations run once, in
// version order, tracked in the schema_migrations table. DDL/DML lives here
// in versioned steps; day-to-day data access goes through the GORM models.
type Migration struct {
	Version int
	Name    string
	Up      func(db *gorm.DB) error
}

// schemaMigration tracks applied migration versions.
type schemaMigration struct {
	Version   int    `gorm:"column:version;primaryKey"`
	Name      string `gorm:"column:name"`
	AppliedAt string `gorm:"column:applied_at"`
}

// TableName pins the migration tracking table.
func (schemaMigration) TableName() string { return "schema_migrations" }

// migrationList is the ordered migration history. Append-only: never edit a
// landed migration, add a new version instead. Migrations never auto-diff
// existing tables: v1 creates tables only when missing, and later versions
// add columns explicitly, so legacy databases are altered surgically.
func migrationList() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "initial_schema",
			Up: func(db *gorm.DB) error {
				for _, dst := range []any{
					&bookletModel{},
					&sectionModel{},
					&contentVersionModel{},
					&referenceModel{},
				} {
					if db.Migrator().HasTable(dst) {
						continue
					}
					if err := db.Migrator().CreateTable(dst); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			Version: 2,
			Name:    "add_booklet_header_footer",
			Up: func(db *gorm.DB) error {
				for field, column := range map[string]string{
					"Header":     "header",
					"Footer":     "footer",
					"ShowFooter": "show_footer",
				} {
					if !db.Migrator().HasColumn(&bookletModel{}, column) {
						if err := db.Migrator().AddColumn(&bookletModel{}, field); err != nil {
							return err
						}
					}
				}
				for _, stmt := range []string{
					`UPDATE booklets SET header = '' WHERE header IS NULL`,
					`UPDATE booklets SET footer = '' WHERE footer IS NULL`,
					`UPDATE booklets SET show_footer = 0 WHERE show_footer IS NULL`,
				} {
					if err := db.Exec(stmt).Error; err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			Version: 3,
			Name:    "add_generation_context_refs",
			Up: func(db *gorm.DB) error {
				if !db.Migrator().HasColumn(&sectionModel{}, "generation_context_refs") {
					if err := db.Migrator().AddColumn(&sectionModel{}, "GenerationCtxRefs"); err != nil {
						return err
					}
				}
				return db.Exec(`UPDATE sections SET generation_context_refs = '' WHERE generation_context_refs IS NULL`).Error
			},
		},
		{
			Version: 4,
			Name:    "add_section_position",
			Up: func(db *gorm.DB) error {
				if !db.Migrator().HasColumn(&sectionModel{}, "position") {
					if err := db.Migrator().AddColumn(&sectionModel{}, "Position"); err != nil {
						return err
					}
				}
				// Preserve insertion order as document order for existing rows.
				// Only SQLite can hold legacy data; fresh tables never contain NULLs.
				if string(db.Dialector.Name()) != string(DriverSQLite) {
					return nil
				}
				return db.Exec(`UPDATE sections SET position = (
					SELECT COUNT(*) FROM sections AS s2
					WHERE s2.booklet_id = sections.booklet_id AND s2.rowid <= sections.rowid
				) - 1 WHERE position IS NULL`).Error
			},
		},
	}
}

// Migrate brings the database to the latest version.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("creating migration table: %w", err)
	}
	for _, m := range migrationList() {
		var count int64
		if err := db.Model(&schemaMigration{}).Where("version = ?", m.Version).Count(&count).Error; err != nil {
			return fmt.Errorf("checking migration %d: %w", m.Version, err)
		}
		if count > 0 {
			continue
		}
		if err := m.Up(db); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Name, err)
		}
		record := schemaMigration{Version: m.Version, Name: m.Name, AppliedAt: time.Now().UTC().Format(time.RFC3339)}
		if err := db.Create(&record).Error; err != nil {
			return fmt.Errorf("recording migration %d: %w", m.Version, err)
		}
	}
	return nil
}

// appliedVersions lists recorded migration versions (used by tests).
func appliedVersions(db *gorm.DB) ([]int, error) {
	var versions []int
	if err := db.Model(&schemaMigration{}).Order("version").Pluck("version", &versions).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return versions, nil
}
