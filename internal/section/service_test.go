package section

import (
	"fmt"
	"testing"
	"time"
)

func TestSectionStatusCanTransitionTo(t *testing.T) {
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

func TestSectionStatusIsValid(t *testing.T) {
	valid := []SectionStatus{SectionStatusDraft, SectionStatusGenerated, SectionStatusEdited, SectionStatusReviewed, SectionStatusApproved}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("IsValid(%q) = false, want true", s)
		}
	}
	if SectionStatus("invalid").IsValid() {
		t.Error("IsValid('invalid') = true, want false")
	}
}

func TestSectionValidate(t *testing.T) {
	tests := []struct {
		name    string
		section Section
		wantErr bool
	}{
		{"empty ID", Section{ID: "", Title: "Test", Level: 2, Status: SectionStatusDraft}, true},
		{"empty title", Section{ID: "s1", Title: "", Level: 2, Status: SectionStatusDraft}, true},
		{"level too low", Section{ID: "s1", Title: "Test", Level: 0, Status: SectionStatusDraft}, true},
		{"level too high", Section{ID: "s1", Title: "Test", Level: 7, Status: SectionStatusDraft}, true},
		{"invalid status", Section{ID: "s1", Title: "Test", Level: 2, Status: "invalid"}, true},
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

func TestSectionAddHistoryRecord(t *testing.T) {
	sec := &Section{ID: "s1", Title: "Test", Level: 2, Status: SectionStatusDraft}
	sec.AddHistoryRecord("content v1", SectionStatusDraft, "Initial version")
	sec.AddHistoryRecord("content v2", SectionStatusGenerated, "AI generated")

	if len(sec.History) != 2 {
		t.Fatalf("History length = %d, want 2", len(sec.History))
	}
	if sec.History[0].Version != 1 {
		t.Errorf("History[0].Version = %d, want 1", sec.History[0].Version)
	}
	if sec.History[1].Version != 2 {
		t.Errorf("History[1].Version = %d, want 2", sec.History[1].Version)
	}
	if sec.History[1].Content != "content v2" {
		t.Errorf("History[1].Content = %q, want 'content v2'", sec.History[1].Content)
	}
}

func TestSectionGetLatestHistory(t *testing.T) {
	sec := &Section{ID: "s1", Title: "Test", Level: 2}
	if sec.GetLatestHistory() != nil {
		t.Error("GetLatestHistory() should return nil for empty history")
	}
	sec.AddHistoryRecord("v1", SectionStatusDraft, "first")
	latest := sec.GetLatestHistory()
	if latest == nil || latest.Version != 1 {
		t.Error("GetLatestHistory() failed")
	}
}

func TestSectionRestoreVersion(t *testing.T) {
	sec := &Section{ID: "s1", Title: "Test", Level: 2, Status: SectionStatusDraft}
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

func TestSectionServiceCreateSection(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	sec, err := svc.CreateSection("b1", "Test Title", 2, "Test prompt")
	if err != nil {
		t.Fatalf("CreateSection failed: %v", err)
	}
	if sec.Title != "Test Title" {
		t.Errorf("CreateSection() title = %q, want 'Test Title'", sec.Title)
	}
	if sec.Level != 2 {
		t.Errorf("CreateSection() level = %d, want 2", sec.Level)
	}
	if sec.Status != SectionStatusDraft {
		t.Errorf("CreateSection() status = %q, want 'draft'", sec.Status)
	}
	if sec.ID == "" {
		t.Error("CreateSection() ID should not be empty")
	}
}

func TestSectionServiceCreateSectionInvalidLevel(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	_, err := svc.CreateSection("b1", "Test", 0, "prompt")
	if err == nil {
		t.Fatal("CreateSection should reject level 0")
	}
	_, err = svc.CreateSection("b1", "Test", 7, "prompt")
	if err == nil {
		t.Fatal("CreateSection should reject level 7")
	}
}

func TestSectionServiceCreateSectionEmptyTitle(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	_, err := svc.CreateSection("b1", "", 2, "prompt")
	if err == nil {
		t.Fatal("CreateSection should reject empty title")
	}
}

func TestSectionServiceGenerateSection(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	sec, err := svc.GenerateSection("b1", "Generated Title", 2, "prompt", "# Content")
	if err != nil {
		t.Fatalf("GenerateSection failed: %v", err)
	}
	if sec.Content != "# Content" {
		t.Errorf("GenerateSection() content = %q, want '# Content'", sec.Content)
	}
	if sec.Status != SectionStatusGenerated {
		t.Errorf("GenerateSection() status = %q, want 'generated'", sec.Status)
	}
	if sec.Generation == nil {
		t.Error("GenerateSection() Generation should not be nil")
	}
}

func TestSectionServiceAddHistoryRecord(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	sec, _ := svc.CreateSection("b1", "Test", 2, "prompt")
	err := svc.AddHistoryRecord("b1", sec.ID, "v2 content", SectionStatusEdited, "Human edit")
	if err != nil {
		t.Fatalf("AddHistoryRecord failed: %v", err)
	}

	got, _ := svc.GetSection("b1", sec.ID)
	if len(got.History) != 1 {
		t.Errorf("History length = %d, want 1", len(got.History))
	}
	if got.History[0].Description != "Human edit" {
		t.Errorf("History[0].Description = %q, want 'Human edit'", got.History[0].Description)
	}
}

func TestSectionServiceTransitionStatus(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	sec, _ := svc.CreateSection("b1", "Test", 2, "prompt")
	err := svc.TransitionStatus("b1", sec.ID, SectionStatusGenerated)
	if err != nil {
		t.Fatalf("TransitionStatus failed: %v", err)
	}

	got, _ := svc.GetSection("b1", sec.ID)
	if got.Status != SectionStatusGenerated {
		t.Errorf("TransitionStatus() status = %q, want 'generated'", got.Status)
	}
}

func TestSectionServiceTransitionStatusInvalid(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	sec, _ := svc.CreateSection("b1", "Test", 2, "prompt")
	err := svc.TransitionStatus("b1", sec.ID, SectionStatusApproved)
	if err == nil {
		t.Fatal("TransitionStatus should reject invalid transition")
	}
}

func TestSectionServiceRestoreVersion(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	sec, _ := svc.CreateSection("b1", "Test", 2, "prompt")
	svc.AddHistoryRecord("b1", sec.ID, "original content", SectionStatusDraft, "first")
	sec.Content = "modified"

	refreshed, _ := svc.GetSection("b1", sec.ID)
	version := refreshed.History[0]
	err := svc.RestoreVersion("b1", sec.ID, version)
	if err != nil {
		t.Fatalf("RestoreVersion failed: %v", err)
	}

	got, _ := svc.GetSection("b1", sec.ID)
	if got.Content != "original content" {
		t.Errorf("RestoreVersion() content = %q, want 'original content'", got.Content)
	}
}

func TestSectionServiceGetHistory(t *testing.T) {
	r := &mockRepo{sections: make(map[string]Section)}
	svc := NewService(r)

	sec, _ := svc.CreateSection("b1", "Test", 2, "prompt")
	svc.AddHistoryRecord("b1", sec.ID, "v1", SectionStatusDraft, "first")
	svc.AddHistoryRecord("b1", sec.ID, "v2", SectionStatusGenerated, "second")

	history, err := svc.GetHistory("b1", sec.ID)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("GetHistory() = %d, want 2", len(history))
	}
}

func TestContentVersion(t *testing.T) {
	cv := ContentVersion{
		Version:     1,
		Content:     "test",
		Status:      SectionStatusDraft,
		Description: "first",
		CreatedAt:   time.Now(),
	}
	if cv.Version != 1 {
		t.Errorf("ContentVersion.Version = %d, want 1", cv.Version)
	}
	if cv.Content != "test" {
		t.Errorf("ContentVersion.Content = %q, want 'test'", cv.Content)
	}
}

func TestGenerationMetadata(t *testing.T) {
	gm := &GenerationMetadata{
		Provider:     "openai",
		Model:        "gpt-4",
		Prompt:       "test prompt",
		InputTokens:  100,
		OutputTokens: 50,
	}
	if gm.Provider != "openai" {
		t.Errorf("GenerationMetadata.Provider = %q, want 'openai'", gm.Provider)
	}
}

type mockRepo struct {
	sections map[string]Section
}

func (r *mockRepo) SaveSection(bookletID string, section *Section) error {
	r.sections[section.ID] = *section
	return nil
}

func (r *mockRepo) GetSection(bookletID, sectionID string) (*Section, error) {
	for _, s := range r.sections {
		if s.ID == sectionID {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("section not found")
}

func (r *mockRepo) DeleteSection(bookletID, sectionID string) error {
	for k, s := range r.sections {
		if s.ID == sectionID {
			delete(r.sections, k)
			return nil
		}
	}
	return fmt.Errorf("section not found")
}
