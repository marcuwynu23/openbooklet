package storage

import (
	"testing"
	"time"

	"github.com/openbooklet/openbooklet/internal/booklet"
	"github.com/openbooklet/openbooklet/internal/section"
)

func TestSQLiteSaveAndGetBooklet(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewSQLiteBookletRepository(s.DB())
	b := &booklet.Booklet{
		ID:     "b1",
		Title:  "Test",
		Type:   "sop",
		Status: booklet.BookletStatusDraft,
	}
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()

	if err := repo.SaveBooklet(b); err != nil {
		t.Fatalf("SaveBooklet failed: %v", err)
	}

	got, err := repo.GetBooklet("b1")
	if err != nil {
		t.Fatalf("GetBooklet failed: %v", err)
	}
	if got.Title != "Test" {
		t.Errorf("GetBooklet() title = %q, want 'Test'", got.Title)
	}
	if got.Status != booklet.BookletStatusDraft {
		t.Errorf("GetBooklet() status = %q, want 'draft'", got.Status)
	}
}

func TestSQLiteGetNotFound(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewSQLiteBookletRepository(s.DB())
	_, err = repo.GetBooklet("nonexistent")
	if err == nil {
		t.Fatal("GetBooklet should return error for missing booklet")
	}
}

func TestSQLiteSaveAndGetSection(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewSQLiteBookletRepository(s.DB())
	repo.SaveBooklet(&booklet.Booklet{ID: "b1", Title: "Test", Status: booklet.BookletStatusDraft})

	sec := &booklet.Section{
		ID:     "s1",
		Title:  "Test Section",
		Level:  2,
		Status: section.SectionStatusDraft,
	}
	sec.CreatedAt = time.Now()
	sec.UpdatedAt = time.Now()
	if err := repo.SaveSection("b1", sec); err != nil {
		t.Fatalf("SaveSection failed: %v", err)
	}

	got, err := repo.GetSection("b1", "s1")
	if err != nil {
		t.Fatalf("GetSection failed: %v", err)
	}
	if got.Title != "Test Section" {
		t.Errorf("GetSection() title = %q, want 'Test Section'", got.Title)
	}
}

func TestSQLiteSaveAndGetAllBooklets(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewSQLiteBookletRepository(s.DB())
	repo.SaveBooklet(&booklet.Booklet{ID: "b1", Title: "First"})
	repo.SaveBooklet(&booklet.Booklet{ID: "b2", Title: "Second"})

	all, err := repo.GetAllBooklets()
	if err != nil {
		t.Fatalf("GetAllBooklets failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("GetAllBooklets() = %d, want 2", len(all))
	}
}

func TestSQLiteDeleteBooklet(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewSQLiteBookletRepository(s.DB())
	repo.SaveBooklet(&booklet.Booklet{ID: "b1", Title: "Test"})

	if err := repo.DeleteBooklet("b1"); err != nil {
		t.Fatalf("DeleteBooklet failed: %v", err)
	}

	_, err = repo.GetBooklet("b1")
	if err == nil {
		t.Fatal("GetBooklet should return error after delete")
	}
}

func TestSQLiteDeleteSection(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewSQLiteBookletRepository(s.DB())
	repo.SaveBooklet(&booklet.Booklet{ID: "b1", Title: "Test"})
	repo.SaveSection("b1", &booklet.Section{ID: "s1", Title: "Test", Level: 2, Status: section.SectionStatusDraft})

	if err := repo.DeleteSection("b1", "s1"); err != nil {
		t.Fatalf("DeleteSection failed: %v", err)
	}

	_, err = repo.GetSection("b1", "s1")
	if err == nil {
		t.Fatal("GetSection should return error after delete")
	}
}

func TestSQLiteFullRoundTrip(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewSQLiteBookletRepository(s.DB())

	now := time.Now()
	b := &booklet.Booklet{
		ID:           "b1",
		Title:        "Test Booklet",
		Type:         "sop",
		Version:      "1.0",
		Status:       booklet.BookletStatusDraft,
		Audience:     "devops",
		Instructions: "Use formal language.",
		Header:       "Date: September 12, 2026\n",
		Footer:       "End of document.\n",
		ShowFooter:   true,
		Sections: []booklet.Section{
			{
				ID:        "s1",
				Title:     "Purpose",
				Level:     2,
				Status:    section.SectionStatusGenerated,
				Prompt:    "Explain the purpose",
				Content:   "This is the purpose.",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		References: []booklet.Reference{
			{ID: "r1", DependsOn: []string{"s1"}, Description: "Purpose reference"},
		},
	}
	b.CreatedAt = now
	b.UpdatedAt = now

	if err := repo.SaveBooklet(b); err != nil {
		t.Fatalf("SaveBooklet failed: %v", err)
	}

	got, err := repo.GetBooklet("b1")
	if err != nil {
		t.Fatalf("GetBooklet failed: %v", err)
	}
	if got.Title != "Test Booklet" {
		t.Errorf("Title = %q, want 'Test Booklet'", got.Title)
	}
	if len(got.Sections) != 1 {
		t.Errorf("Sections count = %d, want 1", len(got.Sections))
	}
	if got.Sections[0].Title != "Purpose" {
		t.Errorf("Section[0].Title = %q, want 'Purpose'", got.Sections[0].Title)
	}
	if len(got.References) != 1 {
		t.Errorf("References count = %d, want 1", len(got.References))
	}
	if got.Header != "Date: September 12, 2026\n" || got.Footer != "End of document.\n" || !got.ShowFooter {
		t.Errorf("header/footer = %q/%q/%v", got.Header, got.Footer, got.ShowFooter)
	}
}
