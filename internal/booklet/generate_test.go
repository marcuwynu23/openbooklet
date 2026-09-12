package booklet

import (
	"context"
	"strings"
	"testing"

	"github.com/openbooklet/openbooklet/internal/provider"
	"github.com/openbooklet/openbooklet/internal/section"
)

type generateFakeProvider struct {
	tokens []string
	err    error
}

func (f *generateFakeProvider) Name() string {
	return "fake"
}

func (f *generateFakeProvider) Chat(_ context.Context, _ provider.Request) (provider.Response, error) {
	return provider.Response{}, f.err
}

func (f *generateFakeProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	if f.err != nil {
		return nil, f.err
	}
	ch := make(chan provider.Chunk)
	go func() {
		defer close(ch)
		for _, tok := range f.tokens {
			select {
			case <-ctx.Done():
				return
			case ch <- provider.Chunk{Token: tok}:
			}
		}
	}()
	return ch, nil
}

func (f *generateFakeProvider) Models(_ context.Context) ([]provider.Model, error) {
	return nil, nil
}

const generateFixture = "# Deploy Guide\n\nIntro paragraph.\n\n## Prerequisites\n\n- kubectl\n\n## Steps\n\nRun it.\n"

func TestGenerateSections(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}

	p := &generateFakeProvider{tokens: []string{generateFixture}}
	sections, err := svc.GenerateSections(context.Background(), "b1", p, "fake-model", "Write a deploy guide.")
	if err != nil {
		t.Fatalf("GenerateSections failed: %v", err)
	}

	if len(sections) != 3 {
		t.Fatalf("got %d sections, want 3", len(sections))
	}
	if sections[0].Title != "Deploy Guide" || sections[0].ParentID != nil {
		t.Errorf("first section = %+v, want top-level Deploy Guide", sections[0])
	}
	if sections[1].ParentID == nil || *sections[1].ParentID != sections[0].ID {
		t.Errorf("Prerequisites parent = %v, want %q", sections[1].ParentID, sections[0].ID)
	}
	for _, sec := range sections {
		if sec.Status != section.SectionStatusGenerated {
			t.Errorf("section %q status = %q, want generated", sec.ID, sec.Status)
		}
		if sec.Prompt != "Write a deploy guide." {
			t.Errorf("section %q prompt not preserved", sec.ID)
		}
		if sec.Generation == nil {
			t.Errorf("section %q missing generation metadata", sec.ID)
			continue
		}
		if sec.Generation.Provider != "fake" || sec.Generation.Model != "fake-model" {
			t.Errorf("section %q generation = %+v", sec.ID, sec.Generation)
		}
	}

	stored, err := repo.GetBooklet("b1")
	if err != nil {
		t.Fatalf("GetBooklet failed: %v", err)
	}
	if len(stored.Sections) != 3 {
		t.Errorf("stored sections = %d, want 3", len(stored.Sections))
	}
}

func TestGenerateSectionsPreambleFolded(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}

	p := &generateFakeProvider{tokens: []string{"Leading words.\n\n# Title\n\nBody.\n"}}
	sections, err := svc.GenerateSections(context.Background(), "b1", p, "m", "prompt")
	if err != nil {
		t.Fatalf("GenerateSections failed: %v", err)
	}
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(sections))
	}
	if !strings.Contains(sections[0].Content, "Leading words.") {
		t.Errorf("preamble lost: %q", sections[0].Content)
	}
}

func TestGenerateSectionsErrors(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	p := &generateFakeProvider{tokens: []string{generateFixture}}

	if _, err := svc.GenerateSections(context.Background(), "missing", p, "m", "prompt"); err == nil {
		t.Error("missing booklet should return an error")
	}

	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}
	if _, err := svc.GenerateSections(context.Background(), "b1", nil, "m", "prompt"); err == nil {
		t.Error("nil provider should return an error")
	}
	if _, err := svc.GenerateSections(context.Background(), "b1", p, "m", "  "); err == nil {
		t.Error("empty prompt should return an error")
	}

	noHeadings := &generateFakeProvider{tokens: []string{"Just prose, no headings.\n"}}
	if _, err := svc.GenerateSections(context.Background(), "b1", noHeadings, "m", "prompt"); err == nil {
		t.Error("heading-less output should return an error")
	}

	failing := &generateFakeProvider{err: context.DeadlineExceeded}
	if _, err := svc.GenerateSections(context.Background(), "b1", failing, "m", "prompt"); err == nil {
		t.Error("provider error should propagate")
	}
}

func TestStreamGeneration(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}

	p := &generateFakeProvider{tokens: []string{generateFixture}}
	events, err := svc.StreamGeneration(context.Background(), "b1", p, "fake-model", "Write a deploy guide.")
	if err != nil {
		t.Fatalf("StreamGeneration failed: %v", err)
	}

	var sawStart, sawToken bool
	var sections []Section
	for event := range events {
		switch event.Type {
		case SectionEventStart:
			sawStart = true
			if event.GenerationID == "" {
				t.Error("start event missing generation ID")
			}
		case SectionEventToken:
			sawToken = true
		case SectionEventComplete:
			sections = event.Sections
		case SectionEventError:
			t.Fatalf("stream error: %s", event.Message)
		}
	}
	if !sawStart || !sawToken {
		t.Error("stream missing start or token events")
	}
	if len(sections) != 3 {
		t.Fatalf("complete carries %d sections, want 3", len(sections))
	}

	stored, err := repo.GetBooklet("b1")
	if err != nil {
		t.Fatalf("GetBooklet failed: %v", err)
	}
	if len(stored.Sections) != 3 {
		t.Errorf("stored sections = %d, want 3", len(stored.Sections))
	}
}

func TestStreamGenerationNoHeadings(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}

	p := &generateFakeProvider{tokens: []string{"Just prose.\n"}}
	events, err := svc.StreamGeneration(context.Background(), "b1", p, "m", "prompt")
	if err != nil {
		t.Fatalf("StreamGeneration failed: %v", err)
	}
	for event := range events {
		if event.Type == SectionEventError {
			return
		}
		if event.Type == SectionEventComplete {
			t.Fatal("heading-less output should end in error, not complete")
		}
	}
	t.Error("stream closed without a terminal event")
}

func TestRegenerateSection(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}
	original := &Section{
		ID:      "s1",
		Title:   "Steps",
		Level:   2,
		Content: "Run it.\n",
		Status:  section.SectionStatusEdited,
	}
	if err := svc.AddSection("b1", original); err != nil {
		t.Fatalf("AddSection failed: %v", err)
	}

	p := &generateFakeProvider{tokens: []string{"Run it twice.\n"}}
	updated, err := svc.RegenerateSection(context.Background(), "b1", "s1", p, "m", RegenerateExpand, "")
	if err != nil {
		t.Fatalf("RegenerateSection failed: %v", err)
	}
	if !strings.Contains(updated.Content, "Run it twice.") {
		t.Errorf("content = %q, want regenerated text", updated.Content)
	}
	if updated.Status != section.SectionStatusGenerated {
		t.Errorf("status = %q, want generated", updated.Status)
	}
	if len(updated.History) != 1 || !strings.Contains(updated.History[0].Content, "Run it.") {
		t.Errorf("history = %+v, want snapshot of original", updated.History)
	}
	if updated.Generation == nil || updated.Generation.Provider != "fake" {
		t.Errorf("generation = %+v, want metadata", updated.Generation)
	}

	if _, err := svc.RegenerateSection(context.Background(), "b1", "s1", p, "m", RegenerateEdit, ""); err == nil {
		t.Error("edit mode without instruction should return an error")
	}
	if _, err := svc.RegenerateSection(context.Background(), "b1", "s1", p, "m", "bogus", ""); err == nil {
		t.Error("unknown mode should return an error")
	}
	if _, err := svc.RegenerateSection(context.Background(), "b1", "missing", p, "m", RegenerateRewrite, ""); err == nil {
		t.Error("missing section should return an error")
	}
}
