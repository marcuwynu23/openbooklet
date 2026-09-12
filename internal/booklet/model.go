package booklet

import (
	"errors"
	"fmt"
	"time"

	"github.com/openbooklet/openbooklet/internal/section"
)

// Section is an alias for section.Section to maintain compatibility with the
// booklet domain model. All Section operations are delegated to the section package.
type Section = section.Section

// SectionStatus is an alias for section.SectionStatus.
type SectionStatus = section.SectionStatus

// ContentVersion is an alias for section.ContentVersion.
type ContentVersion = section.ContentVersion

// GenerationMetadata is an alias for section.GenerationMetadata.
type GenerationMetadata = section.GenerationMetadata

// BookletStatus represents the lifecycle status of a booklet.
type BookletStatus string

const (
	BookletStatusDraft     BookletStatus = "draft"
	BookletStatusInReview  BookletStatus = "in_review"
	BookletStatusApproved  BookletStatus = "approved"
	BookletStatusPublished BookletStatus = "published"
	BookletStatusArchived  BookletStatus = "archived"
)

// IsValid returns true if the status is a recognized BookletStatus.
func (s BookletStatus) IsValid() bool {
	switch s {
	case BookletStatusDraft, BookletStatusInReview, BookletStatusApproved, BookletStatusPublished, BookletStatusArchived:
		return true
	}
	return false
}

// CanTransitionTo returns true if the current status can transition to the target status.
// Booklet lifecycle: Draft → InReview → Approved → Published → Archived
func (s BookletStatus) CanTransitionTo(target BookletStatus) bool {
	validTransitions := map[BookletStatus][]BookletStatus{
		BookletStatusDraft:     {BookletStatusInReview},
		BookletStatusInReview:  {BookletStatusApproved, BookletStatusDraft},
		BookletStatusApproved:  {BookletStatusPublished, BookletStatusInReview},
		BookletStatusPublished: {BookletStatusArchived, BookletStatusApproved},
		BookletStatusArchived:  {},
	}
	for _, t := range validTransitions[s] {
		if t == target {
			return true
		}
	}
	return false
}

// Reference defines a dependency or reference between sections.
type Reference struct {
	ID          string
	DependsOn   []string
	Description string
}

// Template defines a reusable booklet template with default sections.
type Template struct {
	Name        string
	Type        string
	Version     string
	Description string
	Sections    []TemplateSection
}

// TemplateSection defines a section within a template.
type TemplateSection struct {
	ID       string
	Title    string
	Level    int
	Required bool
	Prompt   string
	Order    int
}

// Booklet represents the complete document with metadata, instructions, and sections.
type Booklet struct {
	ID           string
	Title        string
	Type         string
	Version      string
	Status       BookletStatus
	Audience     string
	Instructions string
	Template     string
	References   []Reference
	Sections     []Section
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate checks the booklet and all sections for structural validity.
func (b *Booklet) Validate() error {
	if b.ID == "" {
		return errors.New("booklet ID is required")
	}
	if b.Title == "" {
		return fmt.Errorf("booklet title is required")
	}
	if !b.Status.IsValid() {
		return fmt.Errorf("invalid booklet status: %s", b.Status)
	}
	for i := range b.Sections {
		if err := b.Sections[i].Validate(); err != nil {
			return fmt.Errorf("section %q: %w", b.Sections[i].ID, err)
		}
	}
	return b.validateHierarchy()
}

// validateHierarchy checks that parent-child relationships are consistent.
func (b *Booklet) validateHierarchy() error {
	idMap := make(map[string]*Section)
	for i := range b.Sections {
		idMap[b.Sections[i].ID] = &b.Sections[i]
	}
	for i := range b.Sections {
		s := &b.Sections[i]
		if s.ParentID != nil {
			parent, exists := idMap[*s.ParentID]
			if !exists {
				return fmt.Errorf("section %q references unknown parent %q", s.ID, *s.ParentID)
			}
			if parent.Level >= s.Level {
				return fmt.Errorf("section %q level %d must be greater than parent %q level %d",
					s.ID, s.Level, parent.ID, parent.Level)
			}
		}
	}
	return nil
}

// TopLevelSections returns all sections that have no parent.
func (b *Booklet) TopLevelSections() []Section {
	var result []Section
	for i := range b.Sections {
		if b.Sections[i].ParentID == nil {
			result = append(result, b.Sections[i])
		}
	}
	return result
}

// ChildrenOf returns all direct children of the given parent section ID.
func (b *Booklet) ChildrenOf(parentID string) []Section {
	var result []Section
	for i := range b.Sections {
		if b.Sections[i].ParentID != nil && *b.Sections[i].ParentID == parentID {
			result = append(result, b.Sections[i])
		}
	}
	return result
}

// FindSectionByID finds a section by its ID. Returns nil if not found.
func (b *Booklet) FindSectionByID(id string) *Section {
	for i := range b.Sections {
		if b.Sections[i].ID == id {
			return &b.Sections[i]
		}
	}
	return nil
}

// SectionCount returns the total number of sections.
func (b *Booklet) SectionCount() int {
	return len(b.Sections)
}

// MaxLevel returns the deepest nesting level in the section tree.
func (b *Booklet) MaxLevel() int {
	level := 0
	for i := range b.Sections {
		if b.Sections[i].Level > level {
			level = b.Sections[i].Level
		}
	}
	return level
}
