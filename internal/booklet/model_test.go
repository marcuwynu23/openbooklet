package booklet

import (
	"testing"

	"github.com/openbooklet/openbooklet/internal/section"
)

func TestBookletStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		from     BookletStatus
		to       BookletStatus
		expected bool
	}{
		{BookletStatusDraft, BookletStatusInReview, true},
		{BookletStatusInReview, BookletStatusApproved, true},
		{BookletStatusInReview, BookletStatusDraft, true},
		{BookletStatusApproved, BookletStatusPublished, true},
		{BookletStatusApproved, BookletStatusInReview, true},
		{BookletStatusPublished, BookletStatusArchived, true},
		{BookletStatusPublished, BookletStatusApproved, true},
		{BookletStatusArchived, BookletStatusDraft, false},
		{BookletStatusDraft, BookletStatusApproved, false},
		{BookletStatusDraft, BookletStatusPublished, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionTo(%q, %q) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestBookletStatusIsValid(t *testing.T) {
	valid := []BookletStatus{BookletStatusDraft, BookletStatusInReview, BookletStatusApproved, BookletStatusPublished, BookletStatusArchived}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("IsValid(%q) = false, want true", s)
		}
	}
	if BookletStatus("invalid").IsValid() {
		t.Error("IsValid('invalid') = true, want false")
	}
}

func TestSectionStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		from     section.SectionStatus
		to       section.SectionStatus
		expected bool
	}{
		{section.SectionStatusDraft, section.SectionStatusGenerated, true},
		{section.SectionStatusDraft, section.SectionStatusEdited, true},
		{section.SectionStatusGenerated, section.SectionStatusEdited, true},
		{section.SectionStatusGenerated, section.SectionStatusReviewed, true},
		{section.SectionStatusEdited, section.SectionStatusReviewed, true},
		{section.SectionStatusEdited, section.SectionStatusGenerated, true},
		{section.SectionStatusReviewed, section.SectionStatusApproved, true},
		{section.SectionStatusReviewed, section.SectionStatusDraft, true},
		{section.SectionStatusApproved, section.SectionStatusEdited, true},
		{section.SectionStatusDraft, section.SectionStatusApproved, false},
		{section.SectionStatusGenerated, section.SectionStatusDraft, false},
		{section.SectionStatusApproved, section.SectionStatusDraft, false},
		{section.SectionStatusApproved, section.SectionStatusGenerated, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionTo(%q, %q) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestSectionStatusIsValid(t *testing.T) {
	valid := []section.SectionStatus{section.SectionStatusDraft, section.SectionStatusGenerated, section.SectionStatusEdited, section.SectionStatusReviewed, section.SectionStatusApproved}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("IsValid(%q) = false, want true", s)
		}
	}
	if section.SectionStatus("invalid").IsValid() {
		t.Error("IsValid('invalid') = true, want false")
	}
}

func TestSectionValidate(t *testing.T) {
	tests := []struct {
		name    string
		section Section
		wantErr bool
	}{
		{"empty ID", Section{ID: "", Title: "Test", Level: 2, Status: section.SectionStatusDraft}, true},
		{"empty title", Section{ID: "s1", Title: "", Level: 2, Status: section.SectionStatusDraft}, true},
		{"level too low", Section{ID: "s1", Title: "Test", Level: 0, Status: section.SectionStatusDraft}, true},
		{"level too high", Section{ID: "s1", Title: "Test", Level: 7, Status: section.SectionStatusDraft}, true},
		{"invalid status", Section{ID: "s1", Title: "Test", Level: 2, Status: "invalid"}, true},
		{"valid", Section{ID: "s1", Title: "Test", Level: 2, Status: section.SectionStatusDraft}, false},
		{"level 1 valid", Section{ID: "s1", Title: "Test", Level: 1, Status: section.SectionStatusGenerated}, false},
		{"level 6 valid", Section{ID: "s1", Title: "Test", Level: 6, Status: section.SectionStatusApproved}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.section.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBookletValidate(t *testing.T) {
	tests := []struct {
		name    string
		booklet Booklet
		wantErr bool
	}{
		{"empty ID", Booklet{ID: "", Title: "Test"}, true},
		{"empty title", Booklet{ID: "b1", Title: ""}, true},
		{"invalid status", Booklet{ID: "b1", Title: "Test", Status: "invalid"}, true},
		{"valid", Booklet{ID: "b1", Title: "Test", Status: BookletStatusDraft}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.booklet.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBookletValidateHierarchy(t *testing.T) {
	tests := []struct {
		name    string
		booklet Booklet
		wantErr bool
	}{
		{"valid hierarchy", Booklet{
			ID: "b1", Title: "Test", Status: BookletStatusDraft,
			Sections: []Section{
				{ID: "s1", Title: "Parent", Level: 1, Status: section.SectionStatusDraft},
				{ID: "s2", Title: "Child", Level: 2, ParentID: strPtr("s1"), Status: section.SectionStatusDraft},
			},
		}, false},
		{"unknown parent", Booklet{
			ID: "b1", Title: "Test", Status: BookletStatusDraft,
			Sections: []Section{
				{ID: "s2", Title: "Child", Level: 2, ParentID: strPtr("unknown"), Status: section.SectionStatusDraft},
			},
		}, true},
		{"parent level not less than child", Booklet{
			ID: "b1", Title: "Test", Status: BookletStatusDraft,
			Sections: []Section{
				{ID: "s1", Title: "Parent", Level: 2, Status: section.SectionStatusDraft},
				{ID: "s2", Title: "Child", Level: 1, ParentID: strPtr("s1"), Status: section.SectionStatusDraft},
			},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.booklet.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBookletTopLevelSections(t *testing.T) {
	b := Booklet{
		ID: "b1", Title: "Test", Status: BookletStatusDraft,
		Sections: []Section{
			{ID: "s1", Title: "Parent", Level: 1},
			{ID: "s2", Title: "Child", Level: 2, ParentID: strPtr("s1")},
			{ID: "s3", Title: "Sibling", Level: 1},
		},
	}
	top := b.TopLevelSections()
	if len(top) != 2 {
		t.Errorf("TopLevelSections() = %d, want 2", len(top))
	}
	for _, s := range top {
		if s.ParentID != nil {
			t.Errorf("TopLevelSections() returned section with ParentID %s", *s.ParentID)
		}
	}
}

func TestBookletFindSectionByID(t *testing.T) {
	b := Booklet{
		ID: "b1", Title: "Test", Status: BookletStatusDraft,
		Sections: []Section{
			{ID: "s1", Title: "First"},
			{ID: "s2", Title: "Second"},
		},
	}
	s := b.FindSectionByID("s1")
	if s == nil || s.Title != "First" {
		t.Error("FindSectionByID('s1') failed")
	}
	if b.FindSectionByID("s3") != nil {
		t.Error("FindSectionByID('s3') should return nil")
	}
}

func TestBookletSectionCount(t *testing.T) {
	b := Booklet{
		Sections: []Section{
			{ID: "s1"}, {ID: "s2"}, {ID: "s3"},
		},
	}
	if b.SectionCount() != 3 {
		t.Errorf("SectionCount() = %d, want 3", b.SectionCount())
	}
}

func TestBookletMaxLevel(t *testing.T) {
	b := Booklet{
		Sections: []Section{
			{ID: "s1", Level: 1},
			{ID: "s2", Level: 2},
			{ID: "s3", Level: 3},
		},
	}
	if b.MaxLevel() != 3 {
		t.Errorf("MaxLevel() = %d, want 3", b.MaxLevel())
	}
	b2 := Booklet{}
	if b2.MaxLevel() != 0 {
		t.Errorf("MaxLevel() for empty booklet = %d, want 0", b2.MaxLevel())
	}
}

func strPtr(s string) *string {
	return &s
}
