package section

import (
	"errors"
	"fmt"
	"time"
)

// SectionStatus represents the lifecycle status of a section.
type SectionStatus string

const (
	SectionStatusDraft     SectionStatus = "draft"
	SectionStatusGenerated SectionStatus = "generated"
	SectionStatusEdited    SectionStatus = "edited"
	SectionStatusReviewed  SectionStatus = "reviewed"
	SectionStatusApproved  SectionStatus = "approved"
)

// IsValid returns true if the status is a recognized SectionStatus.
func (s SectionStatus) IsValid() bool {
	switch s {
	case SectionStatusDraft, SectionStatusGenerated, SectionStatusEdited, SectionStatusReviewed, SectionStatusApproved:
		return true
	}
	return false
}

// CanTransitionTo returns true if the current status can transition to the target status.
// Section lifecycle: Draft → Generated → Edited → Reviewed → Approved
func (s SectionStatus) CanTransitionTo(target SectionStatus) bool {
	validTransitions := map[SectionStatus][]SectionStatus{
		SectionStatusDraft:     {SectionStatusGenerated, SectionStatusEdited},
		SectionStatusGenerated: {SectionStatusEdited, SectionStatusReviewed},
		SectionStatusEdited:    {SectionStatusReviewed, SectionStatusGenerated},
		SectionStatusReviewed:  {SectionStatusApproved, SectionStatusDraft},
		SectionStatusApproved:  {SectionStatusEdited},
	}
	for _, t := range validTransitions[s] {
		if t == target {
			return true
		}
	}
	return false
}

// GenerationMetadata stores AI generation details for reproducibility and auditability.
type GenerationMetadata struct {
	Provider     string
	Model        string
	Prompt       string
	ContextRefs  []string
	Temperature  *float64
	MaxTokens    *int
	InputTokens  int
	OutputTokens int
	Duration     time.Duration
	CreatedAt    time.Time
}

// ContentVersion represents a version snapshot of section content for history tracking.
type ContentVersion struct {
	Version     int
	Content     string
	Status      SectionStatus
	Description string
	CreatedAt   time.Time
}

// Section represents a single editable cell in a booklet.
type Section struct {
	ID           string
	ParentID     *string
	Title        string
	Level        int
	Prompt       string
	Content      string
	Dependencies []string
	ContextRefs  []string
	Status       SectionStatus
	Generation   *GenerationMetadata
	History      []ContentVersion
	// Position is the zero-based document order. The parser assigns it;
	// repositories persist it and return sections ordered by it.
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate checks the section for structural validity.
func (s *Section) Validate() error {
	if s.ID == "" {
		return errors.New("section ID is required")
	}
	if s.Title == "" {
		return fmt.Errorf("section %q title is required", s.ID)
	}
	if s.Level < 1 || s.Level > 6 {
		return fmt.Errorf("section %q level must be 1-6, got %d", s.ID, s.Level)
	}
	if !s.Status.IsValid() {
		return fmt.Errorf("section %q has invalid status: %s", s.ID, s.Status)
	}
	return nil
}

// AddHistoryRecord appends a new version to the section's history.
func (s *Section) AddHistoryRecord(content string, status SectionStatus, description string) {
	version := ContentVersion{
		Version:     len(s.History) + 1,
		Content:     content,
		Status:      status,
		Description: description,
		CreatedAt:   time.Now(),
	}
	s.History = append(s.History, version)
}

// GetLatestHistory returns the most recent content version, or nil if no history exists.
func (s *Section) GetLatestHistory() *ContentVersion {
	if len(s.History) == 0 {
		return nil
	}
	return &s.History[len(s.History)-1]
}

// RestoreVersion restores the section content and status from a historical version.
func (s *Section) RestoreVersion(version ContentVersion) {
	s.Content = version.Content
	s.Status = version.Status
	s.UpdatedAt = time.Now()
}
