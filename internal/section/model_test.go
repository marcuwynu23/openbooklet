package section

import (
	"testing"
)

func TestSectionModelCanTransitionTo(t *testing.T) {
	tests := []struct {
		from     SectionStatus
		to       SectionStatus
		expected bool
	}{
		{SectionStatusDraft, SectionStatusGenerated, true},
		{SectionStatusDraft, SectionStatusEdited, true},
		{SectionStatusGenerated, SectionStatusEdited, true},
		{SectionStatusGenerated, SectionStatusReviewed, true},
		{SectionStatusEdited, SectionStatusReviewed, true},
		{SectionStatusEdited, SectionStatusGenerated, true},
		{SectionStatusReviewed, SectionStatusApproved, true},
		{SectionStatusReviewed, SectionStatusDraft, true},
		{SectionStatusApproved, SectionStatusEdited, true},
		{SectionStatusDraft, SectionStatusApproved, false},
		{SectionStatusGenerated, SectionStatusDraft, false},
		{SectionStatusApproved, SectionStatusDraft, false},
		{SectionStatusApproved, SectionStatusGenerated, false},
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

func TestSectionModelValidate(t *testing.T) {
	tests := []struct {
		name    string
		section Section
		wantErr bool
	}{
		{"empty ID", Section{ID: "", Title: "Test", Level: 2}, true},
		{"empty title", Section{ID: "s1", Title: "", Level: 2}, true},
		{"level 0", Section{ID: "s1", Title: "Test", Level: 0}, true},
		{"level 7", Section{ID: "s1", Title: "Test", Level: 7}, true},
		{"invalid status", Section{ID: "s1", Title: "Test", Level: 2, Status: "bad"}, true},
		{"valid", Section{ID: "s1", Title: "Test", Level: 2, Status: SectionStatusDraft}, false},
		{"level 1 valid", Section{ID: "s1", Title: "Test", Level: 1, Status: SectionStatusGenerated}, false},
		{"level 6 valid", Section{ID: "s1", Title: "Test", Level: 6, Status: SectionStatusApproved}, false},
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

func TestSectionModelAddHistoryRecord(t *testing.T) {
	sec := &Section{ID: "s1", Title: "Test", Level: 2}
	sec.AddHistoryRecord("content", SectionStatusDraft, "test")
	if len(sec.History) != 1 {
		t.Errorf("History length = %d, want 1", len(sec.History))
	}
	if sec.History[0].Content != "content" {
		t.Errorf("History[0].Content = %q, want 'content'", sec.History[0].Content)
	}
}

func TestSectionModelRestoreVersion(t *testing.T) {
	sec := &Section{ID: "s1", Title: "Test", Level: 2}
	sec.AddHistoryRecord("original", SectionStatusDraft, "first")
	sec.Content = "modified"
	sec.Status = SectionStatusGenerated

	version := sec.History[0]
	sec.RestoreVersion(version)

	if sec.Content != "original" {
		t.Errorf("RestoreVersion() content = %q, want 'original'", sec.Content)
	}
	if sec.Status != SectionStatusDraft {
		t.Errorf("RestoreVersion() status = %q, want 'draft'", sec.Status)
	}
}

func TestSectionModelGetLatestHistory(t *testing.T) {
	sec := &Section{ID: "s1", Title: "Test", Level: 2}
	if sec.GetLatestHistory() != nil {
		t.Error("GetLatestHistory should return nil for empty history")
	}
	sec.AddHistoryRecord("v1", SectionStatusDraft, "first")
	latest := sec.GetLatestHistory()
	if latest == nil || latest.Version != 1 {
		t.Error("GetLatestHistory failed")
	}
}
