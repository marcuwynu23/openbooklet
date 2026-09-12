package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/openbooklet/openbooklet/internal/booklet"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository implements booklet.BookletRepository on any GORM backend.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a repository over an open, migrated database.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// NewSQLiteBookletRepository creates a repository over an open database.
// Deprecated: prefer NewRepository; the backend is dialect-agnostic.
func NewSQLiteBookletRepository(db *gorm.DB) *Repository {
	return NewRepository(db)
}

var _ booklet.BookletRepository = (*Repository)(nil)

// SaveBooklet stores or updates a booklet.
func (r *Repository) SaveBooklet(b *booklet.Booklet) error {
	if b == nil {
		return fmt.Errorf("booklet cannot be nil")
	}

	createdAt := formatTime(b.CreatedAt)
	updatedAt := formatTime(b.UpdatedAt)
	if createdAt == "" {
		createdAt = formatTime(time.Now())
		b.CreatedAt = time.Now()
	}
	if updatedAt == "" {
		updatedAt = formatTime(time.Now())
		b.UpdatedAt = time.Now()
	}

	if err := r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(bookletToModel(b, createdAt, updatedAt)).Error; err != nil {
		return fmt.Errorf("saving booklet: %w", err)
	}

	// Replace dependent rows, mirroring prior replace semantics.
	if err := r.db.Where("booklet_id = ?", b.ID).Delete(&sectionModel{}).Error; err != nil {
		return fmt.Errorf("deleting old sections: %w", err)
	}
	for i := range b.Sections {
		// Persist document order: positions always mirror slice order.
		b.Sections[i].Position = i
		if err := r.saveSection(b.ID, &b.Sections[i]); err != nil {
			return err
		}
	}

	if err := r.db.Where("booklet_id = ?", b.ID).Delete(&referenceModel{}).Error; err != nil {
		return fmt.Errorf("deleting old booklet_references: %w", err)
	}
	for i := range b.References {
		ref := b.References[i]
		m := &referenceModel{
			ID:          ref.ID,
			BookletID:   b.ID,
			DependsOn:   joinStrings(ref.DependsOn),
			Description: ref.Description,
		}
		if err := r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(m).Error; err != nil {
			return fmt.Errorf("saving reference: %w", err)
		}
	}

	return nil
}

// saveSection stores a single section.
func (r *Repository) saveSection(bookletID string, sec *booklet.Section) error {
	createdAt := formatTime(sec.CreatedAt)
	updatedAt := formatTime(sec.UpdatedAt)
	if err := r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(sectionToModel(bookletID, sec, createdAt, updatedAt)).Error; err != nil {
		return fmt.Errorf("saving section: %w", err)
	}
	return nil
}

// GetBooklet retrieves a booklet by ID.
func (r *Repository) GetBooklet(id string) (*booklet.Booklet, error) {
	var m bookletModel
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: booklet %q", booklet.ErrNotFound, id)
		}
		return nil, fmt.Errorf("getting booklet: %w", err)
	}

	b := bookletToDomain(&m)

	sections, err := r.getSections(id)
	if err != nil {
		return nil, fmt.Errorf("loading sections: %w", err)
	}
	b.Sections = sections

	refs, err := r.getBookletReferences(id)
	if err != nil {
		return nil, fmt.Errorf("loading booklet_references: %w", err)
	}
	b.References = refs

	return b, nil
}

// getSections loads all sections for a booklet.
func (r *Repository) getSections(bookletID string) ([]booklet.Section, error) {
	var models []sectionModel
	if err := r.db.Where("booklet_id = ?", bookletID).Order("position, id").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("querying sections: %w", err)
	}

	var sections []booklet.Section
	for i := range models {
		sec := sectionToDomain(&models[i])
		sec.History = r.loadHistory(bookletID, sec.ID)
		sections = append(sections, *sec)
	}

	return sections, nil
}

// loadHistory loads content versions for a section.
func (r *Repository) loadHistory(bookletID, sectionID string) []booklet.ContentVersion {
	var models []contentVersionModel
	if err := r.db.Where("booklet_id = ? AND section_id = ?", bookletID, sectionID).Order("version").Find(&models).Error; err != nil {
		return nil
	}

	var history []booklet.ContentVersion
	for i := range models {
		history = append(history, historyToDomain(&models[i]))
	}

	return history
}

// GetAllBooklets returns all stored booklets.
func (r *Repository) GetAllBooklets() ([]*booklet.Booklet, error) {
	var models []bookletModel
	if err := r.db.Order("updated_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("querying booklets: %w", err)
	}

	var booklets []*booklet.Booklet
	for i := range models {
		booklets = append(booklets, bookletToDomain(&models[i]))
	}

	return booklets, nil
}

// DeleteBooklet removes a booklet by ID.
func (r *Repository) DeleteBooklet(id string) error {
	res := r.db.Where("id = ?", id).Delete(&bookletModel{})
	if res.Error != nil {
		return fmt.Errorf("deleting booklet: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: booklet %q", booklet.ErrNotFound, id)
	}
	return nil
}

// SaveSection stores or updates a section within a booklet.
func (r *Repository) SaveSection(bookletID string, sec *booklet.Section) error {
	return r.saveSection(bookletID, sec)
}

// GetSection retrieves a section by booklet ID and section ID.
func (r *Repository) GetSection(bookletID, sectionID string) (*booklet.Section, error) {
	var m sectionModel
	if err := r.db.Where("id = ? AND booklet_id = ?", sectionID, bookletID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: section %q in booklet %q", booklet.ErrNotFound, sectionID, bookletID)
		}
		return nil, fmt.Errorf("getting section: %w", err)
	}
	sec := sectionToDomain(&m)
	sec.History = r.loadHistory(bookletID, sec.ID)

	return sec, nil
}

// DeleteSection removes a section from a booklet.
func (r *Repository) DeleteSection(bookletID, sectionID string) error {
	res := r.db.Where("id = ? AND booklet_id = ?", sectionID, bookletID).Delete(&sectionModel{})
	if res.Error != nil {
		return fmt.Errorf("deleting section: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: section %q in booklet %q", booklet.ErrNotFound, sectionID, bookletID)
	}
	return nil
}

// getBookletReferences loads booklet_references for a booklet.
func (r *Repository) getBookletReferences(bookletID string) ([]booklet.Reference, error) {
	var models []referenceModel
	if err := r.db.Where("booklet_id = ?", bookletID).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("querying booklet_references: %w", err)
	}

	var refs []booklet.Reference
	for i := range models {
		refs = append(refs, referenceToDomain(&models[i]))
	}

	return refs, nil
}

// intPtrOrNil returns a pointer to n if n > 0, otherwise nil.
func intPtrOrNil(n int64) *int {
	if n <= 0 {
		return nil
	}
	p := int(n)
	return &p
}

// parseJSONArray parses a comma-separated quoted string into a string slice.
func parseJSONArray(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	var current string
	inQuote := false
	for _, r := range s {
		if r == '"' {
			inQuote = !inQuote
		} else if r == ',' && !inQuote {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// joinStrings joins a string slice into a comma-separated quoted string.
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ","
		}
		result += `"` + s + `"`
	}
	return result
}

// boolToInt maps a bool to 0/1 for integer-backed storage.
func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// formatTime converts time.Time to a string.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// parseTime converts a string to time.Time.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
