package booklet

import (
	"testing"

	"github.com/openbooklet/openbooklet/internal/section"
)

func TestInMemoryRepoSaveAndGetBooklet(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	b := &Booklet{ID: "b1", Title: "Test", Type: "sop", Status: BookletStatusDraft}

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
}

func TestInMemoryRepoGetNotFound(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	_, err := repo.GetBooklet("nonexistent")
	if err == nil {
		t.Fatal("GetBooklet should return error for missing booklet")
	}
	if err.Error() == "" {
		t.Error("Expected non-empty error")
	}
}

func TestInMemoryRepoGetAllBooklets(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	booklets := []*Booklet{
		{ID: "b1", Title: "First"},
		{ID: "b2", Title: "Second"},
	}
	for _, b := range booklets {
		repo.SaveBooklet(b)
	}

	all, err := repo.GetAllBooklets()
	if err != nil {
		t.Fatalf("GetAllBooklets failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("GetAllBooklets() = %d, want 2", len(all))
	}
}

func TestInMemoryRepoDeleteBooklet(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	repo.SaveBooklet(&Booklet{ID: "b1", Title: "Test"})

	if err := repo.DeleteBooklet("b1"); err != nil {
		t.Fatalf("DeleteBooklet failed: %v", err)
	}

	_, err := repo.GetBooklet("b1")
	if err == nil {
		t.Fatal("GetBooklet should return error after delete")
	}
}

func TestInMemoryRepoSaveAndGetSection(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	repo.SaveBooklet(&Booklet{ID: "b1", Title: "Test", Status: BookletStatusDraft})

	sec := &Section{
		ID:     "s1",
		Title:  "Test Section",
		Level:  2,
		Status: section.SectionStatusDraft,
	}
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

func TestInMemoryRepoSaveSectionUpdates(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	repo.SaveBooklet(&Booklet{ID: "b1", Title: "Test", Status: BookletStatusDraft})

	sec := &Section{ID: "s1", Title: "Original", Level: 2, Status: section.SectionStatusDraft}
	repo.SaveSection("b1", sec)

	sec.Title = "Updated"
	repo.SaveSection("b1", sec)

	got, _ := repo.GetSection("b1", "s1")
	if got.Title != "Updated" {
		t.Errorf("SaveSection update failed, got %q, want 'Updated'", got.Title)
	}
}

func TestInMemoryRepoDeleteSection(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	repo.SaveBooklet(&Booklet{ID: "b1", Title: "Test", Status: BookletStatusDraft})
	repo.SaveSection("b1", &Section{ID: "s1", Title: "Test", Level: 2, Status: section.SectionStatusDraft})

	if err := repo.DeleteSection("b1", "s1"); err != nil {
		t.Fatalf("DeleteSection failed: %v", err)
	}

	_, err := repo.GetSection("b1", "s1")
	if err == nil {
		t.Fatal("GetSection should return error after delete")
	}
}

func TestInMemoryRepoSaveSectionBookletNotFound(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	err := repo.SaveSection("b1", &Section{ID: "s1", Title: "Test", Level: 2, Status: section.SectionStatusDraft})
	if err == nil {
		t.Fatal("SaveSection should return error for missing booklet")
	}
}

func TestInMemoryRepoGetSectionBookletNotFound(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	_, err := repo.GetSection("b1", "s1")
	if err == nil {
		t.Fatal("GetSection should return error for missing booklet")
	}
}

func TestInMemoryRepoDeleteSectionBookletNotFound(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	err := repo.DeleteSection("b1", "s1")
	if err == nil {
		t.Fatal("DeleteSection should return error for missing booklet")
	}
}

func TestServiceCreateBooklet(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)

	b, err := svc.CreateBooklet("b1", "Test Booklet", "sop", "devops", "Use formal language.")
	if err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}
	if b.ID != "b1" {
		t.Errorf("CreateBooklet() ID = %q, want 'b1'", b.ID)
	}
	if b.Status != BookletStatusDraft {
		t.Errorf("CreateBooklet() status = %q, want 'draft'", b.Status)
	}
}

func TestServiceCreateDuplicateBooklet(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)

	_, err := svc.CreateBooklet("b1", "Test", "sop", "", "")
	if err != nil {
		t.Fatalf("First CreateBooklet failed: %v", err)
	}
	_, err = svc.CreateBooklet("b1", "Test2", "sop", "", "")
	if err == nil {
		t.Fatal("Second CreateBooklet should return error")
	}
}

func TestServiceGetBooklet(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)

	svc.CreateBooklet("b1", "Test", "sop", "", "")
	b, err := svc.GetBooklet("b1")
	if err != nil {
		t.Fatalf("GetBooklet failed: %v", err)
	}
	if b.Title != "Test" {
		t.Errorf("GetBooklet() title = %q, want 'Test'", b.Title)
	}
}

func TestServiceAddSection(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")

	sec := &Section{ID: "s1", Title: "Test Section", Level: 2, Status: section.SectionStatusDraft}
	if err := svc.AddSection("b1", sec); err != nil {
		t.Fatalf("AddSection failed: %v", err)
	}

	got, err := svc.GetSection("b1", "s1")
	if err != nil {
		t.Fatalf("GetSection failed: %v", err)
	}
	if got.Title != "Test Section" {
		t.Errorf("GetSection() title = %q, want 'Test Section'", got.Title)
	}
}

func TestServiceChangeSectionStatus(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")

	sec := &Section{ID: "s1", Title: "Test", Level: 2, Status: section.SectionStatusDraft}
	svc.AddSection("b1", sec)

	err := svc.ChangeSectionStatus("b1", "s1", section.SectionStatusGenerated)
	if err != nil {
		t.Fatalf("ChangeSectionStatus failed: %v", err)
	}

	got, _ := svc.GetSection("b1", "s1")
	if got.Status != section.SectionStatusGenerated {
		t.Errorf("ChangeSectionStatus() status = %q, want 'generated'", got.Status)
	}
}

func TestServiceChangeSectionStatusInvalid(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")

	sec := &Section{ID: "s1", Title: "Test", Level: 2, Status: section.SectionStatusApproved}
	svc.AddSection("b1", sec)

	err := svc.ChangeSectionStatus("b1", "s1", section.SectionStatusDraft)
	if err == nil {
		t.Fatal("ChangeSectionStatus should reject invalid transition")
	}
}

func TestServiceReorderSections(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")

	svc.AddSection("b1", &Section{ID: "s1", Title: "First", Level: 1, Status: section.SectionStatusDraft})
	svc.AddSection("b1", &Section{ID: "s2", Title: "Second", Level: 1, Status: section.SectionStatusDraft})
	svc.AddSection("b1", &Section{ID: "s3", Title: "Third", Level: 1, Status: section.SectionStatusDraft})

	err := svc.ReorderSections("b1", []string{"s3", "s1", "s2"})
	if err != nil {
		t.Fatalf("ReorderSections failed: %v", err)
	}

	b, _ := repo.GetBooklet("b1")
	if b.Sections[0].ID != "s3" || b.Sections[1].ID != "s1" || b.Sections[2].ID != "s2" {
		t.Errorf("ReorderSections did not reorder correctly")
	}
}

func TestServiceReorderSectionsInvalidID(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")

	err := svc.ReorderSections("b1", []string{"nonexistent"})
	if err == nil {
		t.Fatal("ReorderSections should return error for nonexistent section")
	}
}

func TestServiceMoveSection(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")

	svc.AddSection("b1", &Section{ID: "s1", Title: "Parent", Level: 1, Status: section.SectionStatusDraft})
	svc.AddSection("b1", &Section{ID: "s2", Title: "Child", Level: 2, Status: section.SectionStatusDraft})

	err := svc.MoveSection("b1", "s2", strPtr("s1"), 2)
	if err != nil {
		t.Fatalf("MoveSection failed: %v", err)
	}

	sec, _ := svc.GetSection("b1", "s2")
	if sec.ParentID == nil || *sec.ParentID != "s1" {
		t.Errorf("MoveSection did not set parent correctly")
	}
}

func TestServiceMoveSectionInvalidLevel(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")

	err := svc.MoveSection("b1", "s2", nil, 0)
	if err == nil {
		t.Fatal("MoveSection should reject invalid level")
	}
}

func TestServiceCountSections(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")
	svc.AddSection("b1", &Section{ID: "s1", Title: "A", Level: 1, Status: section.SectionStatusDraft})
	svc.AddSection("b1", &Section{ID: "s2", Title: "B", Level: 1, Status: section.SectionStatusDraft})

	count, err := svc.CountSections("b1")
	if err != nil {
		t.Fatalf("CountSections failed: %v", err)
	}
	if count != 2 {
		t.Errorf("CountSections() = %d, want 2", count)
	}
}

func TestServiceGetSectionTree(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")

	svc.AddSection("b1", &Section{ID: "s1", Title: "Parent", Level: 1, Status: section.SectionStatusDraft})
	svc.AddSection("b1", &Section{ID: "s2", Title: "Child", Level: 2, ParentID: strPtr("s1"), Status: section.SectionStatusDraft})

	tree, err := svc.GetSectionTree("b1")
	if err != nil {
		t.Fatalf("GetSectionTree failed: %v", err)
	}
	if len(tree) != 1 {
		t.Errorf("GetSectionTree() = %d, want 1", len(tree))
	}
	if tree[0].ID != "s1" {
		t.Errorf("GetSectionTree()[0].ID = %q, want 's1'", tree[0].ID)
	}
}

func TestServiceUpdateSection(t *testing.T) {
	repo := NewInMemoryBookletRepository()
	svc := NewService(repo)
	svc.CreateBooklet("b1", "Test", "sop", "", "")                                                                //nolint:errcheck
	svc.AddSection("b1", &Section{ID: "s1", Title: "Original", Level: 2, Status: section.SectionStatusGenerated}) //nolint:errcheck

	newContent := "Edited content."
	updated, err := svc.UpdateSection("b1", "s1", nil, nil, &newContent)
	if err != nil {
		t.Fatalf("UpdateSection failed: %v", err)
	}
	if updated.Content != newContent {
		t.Errorf("content = %q, want %q", updated.Content, newContent)
	}
	if updated.Title != "Original" {
		t.Errorf("title = %q, want untouched 'Original'", updated.Title)
	}
	if updated.Status != section.SectionStatusGenerated {
		t.Errorf("status = %q, want untouched 'generated'", updated.Status)
	}

	empty := ""
	if _, err := svc.UpdateSection("b1", "s1", &empty, nil, nil); err == nil {
		t.Error("empty title should fail validation")
	}
	if _, err := svc.UpdateSection("b1", "missing", nil, nil, &newContent); err == nil {
		t.Error("missing section should return an error")
	}
}
