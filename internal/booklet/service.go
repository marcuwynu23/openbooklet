package booklet

import (
	"fmt"
	"time"

	"github.com/openbooklet/openbooklet/internal/section"
)

// Service orchestrates booklet operations including section management.
type Service struct {
	repo BookletRepository
}

// NewService creates a new booklet service with the given repository.
func NewService(repo BookletRepository) *Service {
	return &Service{repo: repo}
}

// CreateBooklet creates a new booklet with the given ID and title.
func (s *Service) CreateBooklet(id, title, bookletType, audience, instructions string) (*Booklet, error) {
	if id == "" {
		return nil, fmt.Errorf("booklet ID is required")
	}
	if title == "" {
		return nil, fmt.Errorf("booklet title is required")
	}

	_, err := s.repo.GetBooklet(id)
	if err == nil {
		return nil, fmt.Errorf("booklet %q already exists: %w", id, ErrConflict)
	}

	booklet := &Booklet{
		ID:           id,
		Title:        title,
		Type:         bookletType,
		Version:      "1.0",
		Status:       BookletStatusDraft,
		Audience:     audience,
		Instructions: instructions,
		CreatedAt:    timeNow(),
		UpdatedAt:    timeNow(),
	}

	if err := booklet.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveBooklet(booklet); err != nil {
		return nil, fmt.Errorf("saving booklet: %w", err)
	}

	return booklet, nil
}

// GetBooklet retrieves a booklet by ID.
func (s *Service) GetBooklet(id string) (*Booklet, error) {
	return s.repo.GetBooklet(id)
}

// GetAllBooklets returns all booklets.
func (s *Service) GetAllBooklets() ([]*Booklet, error) {
	return s.repo.GetAllBooklets()
}

// DeleteBooklet removes a booklet by ID.
func (s *Service) DeleteBooklet(id string) error {
	return s.repo.DeleteBooklet(id)
}

// UpdateBooklet updates an existing booklet.
func (s *Service) UpdateBooklet(booklet *Booklet) error {
	if err := booklet.Validate(); err != nil {
		return err
	}
	booklet.UpdatedAt = timeNow()
	return s.repo.SaveBooklet(booklet)
}

// AddSection adds a section to a booklet.
func (s *Service) AddSection(bookletID string, sec *Section) error {
	if err := sec.Validate(); err != nil {
		return err
	}
	return s.repo.SaveSection(bookletID, sec)
}

// GetSection retrieves a section from a booklet.
func (s *Service) GetSection(bookletID, sectionID string) (*Section, error) {
	return s.repo.GetSection(bookletID, sectionID)
}

// DeleteSection removes a section from a booklet.
func (s *Service) DeleteSection(bookletID, sectionID string) error {
	return s.repo.DeleteSection(bookletID, sectionID)
}

// UpdateSection applies a partial update to a section's editable fields.
// Status is left untouched; status changes go through ChangeSectionStatus.
// Changing the level re-validates the tree: a parent that no longer sits
// above the section is detached (the section becomes top-level) rather than
// left in an invalid state.
func (s *Service) UpdateSection(bookletID, sectionID string, title, prompt, content *string, level *int) (*Section, error) {
	sec, err := s.repo.GetSection(bookletID, sectionID)
	if err != nil {
		return nil, err
	}
	if title != nil {
		sec.Title = *title
	}
	if prompt != nil {
		sec.Prompt = *prompt
	}
	if content != nil {
		sec.Content = *content
	}
	if level != nil {
		if *level < 1 || *level > 6 {
			return nil, fmt.Errorf("level must be 1-6, got %d", *level)
		}
		sec.Level = *level
	}
	sec.UpdatedAt = timeNow()
	if err := sec.Validate(); err != nil {
		return nil, err
	}
	if level != nil {
		if err := s.revalidateParent(bookletID, sec); err != nil {
			return nil, err
		}
	}
	if err := s.repo.SaveSection(bookletID, sec); err != nil {
		return nil, fmt.Errorf("saving section: %w", err)
	}
	return sec, nil
}

// revalidateParent detaches a section whose parent no longer sits above it
// after a level change. Validation runs against the booklet tree with the
// change applied.
func (s *Service) revalidateParent(bookletID string, sec *Section) error {
	b, err := s.repo.GetBooklet(bookletID)
	if err != nil {
		return err
	}
	stored := b.FindSectionByID(sec.ID)
	if stored == nil {
		return fmt.Errorf("section %q not found in booklet %q", sec.ID, bookletID)
	}
	stored.Level = sec.Level
	if sec.ParentID != nil {
		if parent := b.FindSectionByID(*sec.ParentID); parent != nil && parent.Level >= sec.Level {
			sec.ParentID = nil
			stored.ParentID = nil
		}
	}
	return b.validateHierarchy()
}

// ChangeSectionStatus transitions a section to a new status.
func (s *Service) ChangeSectionStatus(bookletID, sectionID string, newStatus section.SectionStatus) error {
	sec, err := s.repo.GetSection(bookletID, sectionID)
	if err != nil {
		return err
	}
	if !sec.Status.CanTransitionTo(newStatus) {
		return fmt.Errorf("invalid status transition: %s → %s", sec.Status, newStatus)
	}
	sec.Status = newStatus
	sec.UpdatedAt = timeNow()
	return s.repo.SaveSection(bookletID, sec)
}

// ReorderSections reorders sections within a booklet.
func (s *Service) ReorderSections(bookletID string, sectionIDs []string) error {
	b, err := s.repo.GetBooklet(bookletID)
	if err != nil {
		return err
	}

	idSet := make(map[string]bool)
	for _, id := range sectionIDs {
		idSet[id] = true
	}

	for _, id := range sectionIDs {
		found := false
		for _, s := range b.Sections {
			if s.ID == id {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("section %q not found in booklet %q", id, bookletID)
		}
	}

	ordered := make([]Section, 0, len(b.Sections))
	seen := make(map[string]bool)
	for _, id := range sectionIDs {
		for i := range b.Sections {
			if b.Sections[i].ID == id && !seen[id] {
				ordered = append(ordered, b.Sections[i])
				seen[id] = true
				break
			}
		}
	}

	for i := range b.Sections {
		if !seen[b.Sections[i].ID] {
			ordered = append(ordered, b.Sections[i])
			seen[b.Sections[i].ID] = true
		}
	}

	b.Sections = ordered
	b.UpdatedAt = timeNow()
	return s.repo.SaveBooklet(b)
}

// SetSectionParent sets or clears the parent of a section.
func (s *Service) SetSectionParent(bookletID, sectionID string, parentID *string) error {
	b, err := s.repo.GetBooklet(bookletID)
	if err != nil {
		return err
	}

	sec := b.FindSectionByID(sectionID)
	if sec == nil {
		return fmt.Errorf("section %q not found in booklet %q", sectionID, bookletID)
	}

	if parentID != nil {
		parent := b.FindSectionByID(*parentID)
		if parent == nil {
			return fmt.Errorf("parent section %q not found in booklet %q", *parentID, bookletID)
		}
		if parent.Level >= sec.Level {
			return fmt.Errorf("cannot set parent %q (level %d) for section %q (level %d)",
				*parentID, parent.Level, sectionID, sec.Level)
		}
	}

	sec.ParentID = parentID
	sec.UpdatedAt = timeNow()
	return s.repo.SaveSection(bookletID, sec)
}

// MoveSection changes the parent and level of a section.
func (s *Service) MoveSection(bookletID, sectionID string, newParentID *string, newLevel int) error {
	if newLevel < 1 || newLevel > 6 {
		return fmt.Errorf("level must be 1-6, got %d", newLevel)
	}

	b, err := s.repo.GetBooklet(bookletID)
	if err != nil {
		return err
	}

	sec := b.FindSectionByID(sectionID)
	if sec == nil {
		return fmt.Errorf("section %q not found in booklet %q", sectionID, bookletID)
	}

	if newParentID != nil {
		parent := b.FindSectionByID(*newParentID)
		if parent == nil {
			return fmt.Errorf("parent section %q not found in booklet %q", *newParentID, bookletID)
		}
		if parent.Level >= newLevel {
			return fmt.Errorf("cannot move section to level %d under parent at level %d", newLevel, parent.Level)
		}
	}

	sec.ParentID = newParentID
	sec.Level = newLevel
	sec.UpdatedAt = timeNow()
	return s.repo.SaveSection(bookletID, sec)
}

// GetSectionTree returns the top-level sections of a booklet.
func (s *Service) GetSectionTree(bookletID string) ([]Section, error) {
	b, err := s.repo.GetBooklet(bookletID)
	if err != nil {
		return nil, err
	}
	return b.TopLevelSections(), nil
}

// CountSections returns the total number of sections in a booklet.
func (s *Service) CountSections(bookletID string) (int, error) {
	b, err := s.repo.GetBooklet(bookletID)
	if err != nil {
		return 0, err
	}
	return b.SectionCount(), nil
}

// timeNow returns the current time, extractable for testing.
func timeNow() time.Time {
	return time.Now()
}
