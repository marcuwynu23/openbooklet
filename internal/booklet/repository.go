package booklet

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("resource not found")

// ErrConflict is returned when an operation would cause a conflict.
var ErrConflict = errors.New("resource conflict")

// BookletRepository defines the data access contract for booklet operations.
type BookletRepository interface {
	SaveBooklet(booklet *Booklet) error
	GetBooklet(id string) (*Booklet, error)
	GetAllBooklets() ([]*Booklet, error)
	DeleteBooklet(id string) error
	SaveSection(bookletID string, section *Section) error
	GetSection(bookletID, sectionID string) (*Section, error)
	DeleteSection(bookletID, sectionID string) error
}

// InMemoryBookletRepository is a phase-1 in-memory implementation of BookletRepository.
type InMemoryBookletRepository struct {
	booklets map[string]*Booklet
}

// NewInMemoryBookletRepository creates a new in-memory booklet repository.
func NewInMemoryBookletRepository() *InMemoryBookletRepository {
	return &InMemoryBookletRepository{
		booklets: make(map[string]*Booklet),
	}
}

// SaveBooklet stores or updates a booklet in the repository.
func (r *InMemoryBookletRepository) SaveBooklet(booklet *Booklet) error {
	if booklet == nil {
		return fmt.Errorf("booklet cannot be nil")
	}
	r.booklets[booklet.ID] = booklet
	return nil
}

// GetBooklet retrieves a booklet by ID. Returns ErrNotFound if not found.
func (r *InMemoryBookletRepository) GetBooklet(id string) (*Booklet, error) {
	b, exists := r.booklets[id]
	if !exists {
		return nil, fmt.Errorf("%w: booklet %q", ErrNotFound, id)
	}
	return b, nil
}

// GetAllBooklets returns all stored booklets.
func (r *InMemoryBookletRepository) GetAllBooklets() ([]*Booklet, error) {
	var result []*Booklet
	for _, b := range r.booklets {
		result = append(result, b)
	}
	return result, nil
}

// DeleteBooklet removes a booklet by ID. Returns ErrNotFound if not found.
func (r *InMemoryBookletRepository) DeleteBooklet(id string) error {
	if _, exists := r.booklets[id]; !exists {
		return fmt.Errorf("%w: booklet %q", ErrNotFound, id)
	}
	delete(r.booklets, id)
	return nil
}

// SaveSection stores or updates a section within a booklet.
func (r *InMemoryBookletRepository) SaveSection(bookletID string, section *Section) error {
	b, exists := r.booklets[bookletID]
	if !exists {
		return fmt.Errorf("booklet %q: %w", bookletID, ErrNotFound)
	}
	for i := range b.Sections {
		if b.Sections[i].ID == section.ID {
			b.Sections[i] = *section
			return nil
		}
	}
	b.Sections = append(b.Sections, *section)
	return nil
}

// GetSection retrieves a section by booklet ID and section ID.
func (r *InMemoryBookletRepository) GetSection(bookletID, sectionID string) (*Section, error) {
	b, exists := r.booklets[bookletID]
	if !exists {
		return nil, fmt.Errorf("%w: booklet %q", ErrNotFound, bookletID)
	}
	for i := range b.Sections {
		if b.Sections[i].ID == sectionID {
			return &b.Sections[i], nil
		}
	}
	return nil, fmt.Errorf("%w: section %q in booklet %q", ErrNotFound, sectionID, bookletID)
}

// DeleteSection removes a section from a booklet.
func (r *InMemoryBookletRepository) DeleteSection(bookletID, sectionID string) error {
	b, exists := r.booklets[bookletID]
	if !exists {
		return fmt.Errorf("booklet %q: %w", bookletID, ErrNotFound)
	}
	for i := range b.Sections {
		if b.Sections[i].ID == sectionID {
			b.Sections = append(b.Sections[:i], b.Sections[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%w: section %q in booklet %q", ErrNotFound, sectionID, bookletID)
}
