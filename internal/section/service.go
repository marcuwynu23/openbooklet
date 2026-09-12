package section

import (
	"fmt"
	"time"
)

// Service provides section-specific operations including history management.
type Service struct {
	repo SectionRepository
}

// SectionRepository defines the data access contract for section operations.
type SectionRepository interface {
	SaveSection(bookletID string, section *Section) error
	GetSection(bookletID, sectionID string) (*Section, error)
	DeleteSection(bookletID, sectionID string) error
}

// NewService creates a new section service with the given repository.
func NewService(repo SectionRepository) *Service {
	return &Service{repo: repo}
}

// CreateSection creates a new section within a booklet.
func (s *Service) CreateSection(bookletID string, title string, level int, prompt string) (*Section, error) {
	if title == "" {
		return nil, fmt.Errorf("section title is required")
	}
	if level < 1 || level > 6 {
		return nil, fmt.Errorf("section level must be 1-6, got %d", level)
	}

	section := &Section{
		ID:        fmt.Sprintf("sec-%d", time.Now().UnixNano()),
		Title:     title,
		Level:     level,
		Prompt:    prompt,
		Content:   "",
		Status:    SectionStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := section.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveSection(bookletID, section); err != nil {
		return nil, fmt.Errorf("creating section: %w", err)
	}

	return section, nil
}

// GetSection retrieves a section by booklet ID and section ID.
func (s *Service) GetSection(bookletID, sectionID string) (*Section, error) {
	return s.repo.GetSection(bookletID, sectionID)
}

// UpdateSection updates an existing section.
func (s *Service) UpdateSection(bookletID string, section *Section) error {
	if err := section.Validate(); err != nil {
		return err
	}
	section.UpdatedAt = time.Now()
	return s.repo.SaveSection(bookletID, section)
}

// DeleteSection removes a section from a booklet.
func (s *Service) DeleteSection(bookletID, sectionID string) error {
	return s.repo.DeleteSection(bookletID, sectionID)
}

// GenerateSection creates a section from a prompt and sets its status to Generated.
func (s *Service) GenerateSection(bookletID string, title string, level int, prompt string, content string) (*Section, error) {
	sec, err := s.CreateSection(bookletID, title, level, prompt)
	if err != nil {
		return nil, err
	}
	sec.Content = content
	sec.Status = SectionStatusGenerated
	sec.Generation = &GenerationMetadata{
		CreatedAt: time.Now(),
	}
	sec.UpdatedAt = time.Now()

	if err := s.repo.SaveSection(bookletID, sec); err != nil {
		return nil, err
	}

	return sec, nil
}

// AddHistoryRecord appends a version to the section's history.
func (s *Service) AddHistoryRecord(bookletID, sectionID string, content string, status SectionStatus, description string) error {
	sec, err := s.repo.GetSection(bookletID, sectionID)
	if err != nil {
		return err
	}
	sec.AddHistoryRecord(content, status, description)
	sec.UpdatedAt = time.Now()
	return s.repo.SaveSection(bookletID, sec)
}

// RestoreVersion restores a section to a previous version.
func (s *Service) RestoreVersion(bookletID, sectionID string, version ContentVersion) error {
	sec, err := s.repo.GetSection(bookletID, sectionID)
	if err != nil {
		return err
	}
	sec.RestoreVersion(version)
	return s.repo.SaveSection(bookletID, sec)
}

// GetHistory returns the version history for a section.
func (s *Service) GetHistory(bookletID, sectionID string) ([]ContentVersion, error) {
	sec, err := s.repo.GetSection(bookletID, sectionID)
	if err != nil {
		return nil, err
	}
	return sec.History, nil
}

// TransitionStatus moves a section to a new status if the transition is valid.
func (s *Service) TransitionStatus(bookletID, sectionID string, newStatus SectionStatus) error {
	sec, err := s.repo.GetSection(bookletID, sectionID)
	if err != nil {
		return err
	}
	if !sec.Status.CanTransitionTo(newStatus) {
		return fmt.Errorf("invalid section status transition: %s → %s", sec.Status, newStatus)
	}
	sec.Status = newStatus
	sec.UpdatedAt = time.Now()
	return s.repo.SaveSection(bookletID, sec)
}
