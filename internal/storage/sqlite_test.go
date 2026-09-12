package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/openbooklet/openbooklet/internal/booklet"
	"github.com/openbooklet/openbooklet/internal/section"

	_ "github.com/glebarez/sqlite"
)

func TestSQLiteSaveAndGetBooklet(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewRepository(s.DB())
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

	repo := NewRepository(s.DB())
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

	repo := NewRepository(s.DB())
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

	repo := NewRepository(s.DB())
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

	repo := NewRepository(s.DB())
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

	repo := NewRepository(s.DB())
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

	repo := NewRepository(s.DB())

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

// TestSQLitePreservesSectionOrder pins document order: titles are chosen so
// ID sorting differs from file layout, and reads must follow layout.
func TestSQLitePreservesSectionOrder(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewRepository(s.DB())
	now := time.Now()
	b := &booklet.Booklet{
		ID:     "b1",
		Title:  "Ordered",
		Status: booklet.BookletStatusDraft,
		Sections: []booklet.Section{
			{ID: "zebra", Title: "Zebra", Level: 1, Position: 0, Status: section.SectionStatusDraft, CreatedAt: now, UpdatedAt: now},
			{ID: "mango", Title: "Mango", Level: 1, Position: 1, Status: section.SectionStatusDraft, CreatedAt: now, UpdatedAt: now},
			{ID: "apple", Title: "Apple", Level: 1, Position: 2, Status: section.SectionStatusDraft, CreatedAt: now, UpdatedAt: now},
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
	want := []string{"Zebra", "Mango", "Apple"}
	for i, title := range want {
		if got.Sections[i].Title != title {
			t.Fatalf("section %d = %q, want %q (file order)", i, got.Sections[i].Title, title)
		}
	}
}

// TestSQLiteFileBackedListBooklets mirrors production: a real database file
// (like data/openbooklet.db) must save and list booklets. This guards the
// file-backed migration path that :memory: tests never exercise.
func TestSQLiteFileBackedListBooklets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openbooklet.db")
	s, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	repo := NewRepository(s.DB())
	for _, id := range []string{"b1", "b2"} {
		b := &booklet.Booklet{ID: id, Title: "Booklet " + id, Status: booklet.BookletStatusDraft}
		b.CreatedAt = time.Now()
		b.UpdatedAt = time.Now()
		if err := repo.SaveBooklet(b); err != nil {
			t.Fatalf("SaveBooklet failed: %v", err)
		}
	}

	all, err := repo.GetAllBooklets()
	if err != nil {
		t.Fatalf("GetAllBooklets failed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("GetAllBooklets() = %d, want 2", len(all))
	}

	// Reopen the same file: migrations must be idempotent.
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	s2, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer s2.Close()
	if _, err := NewRepository(s2.DB()).GetAllBooklets(); err != nil {
		t.Fatalf("GetAllBooklets after reopen failed: %v", err)
	}
}

// TestSQLiteMigrationFromOldSchema opens a database created before the
// header/footer/show_footer columns existed and verifies the ALTER TABLE
// migration makes it fully readable.
func TestSQLiteMigrationFromOldSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = raw.Exec(`CREATE TABLE booklets (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		type TEXT,
		version TEXT,
		status TEXT NOT NULL DEFAULT 'draft',
		audience TEXT,
		instructions TEXT,
		template TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("creating old schema failed: %v", err)
	}
	_, err = raw.Exec(`INSERT INTO booklets
		(id, title, type, version, status, audience, instructions, template, created_at, updated_at)
		VALUES ('b1', 'Legacy', 'sop', '1.0', 'draft', '', '', '', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z')`)
	if err != nil {
		t.Fatalf("seeding old schema failed: %v", err)
	}
	_, err = raw.Exec(`CREATE TABLE sections (
		id TEXT NOT NULL,
		booklet_id TEXT NOT NULL,
		parent_id TEXT,
		title TEXT NOT NULL,
		level INTEGER NOT NULL,
		prompt TEXT,
		content TEXT,
		dependencies TEXT,
		context_refs TEXT,
		status TEXT NOT NULL DEFAULT 'draft',
		provider TEXT,
		model TEXT,
		prompt_snapshot TEXT,
		temperature REAL,
		max_tokens INTEGER,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		duration_ms INTEGER DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		PRIMARY KEY (id, booklet_id)
	)`)
	if err != nil {
		t.Fatalf("creating old sections failed: %v", err)
	}
	_, err = raw.Exec(`INSERT INTO sections
		(id, booklet_id, parent_id, title, level, prompt, content, dependencies, context_refs, status,
		 provider, model, prompt_snapshot, temperature, max_tokens, input_tokens, output_tokens, duration_ms,
		 created_at, updated_at)
		VALUES ('s1', 'b1', '', 'Old Section', 2, '', 'Body.', '', '', 'generated',
		 'fake', 'm', 'p', NULL, 0, 1, 2, 0,
		 '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z')`)
	if err != nil {
		t.Fatalf("seeding old section failed: %v", err)
	}
	_, err = raw.Exec(`CREATE TABLE booklet_references (
		id TEXT PRIMARY KEY,
		booklet_id TEXT NOT NULL,
		depends_on TEXT,
		description TEXT
	)`)
	if err != nil {
		t.Fatalf("creating old references failed: %v", err)
	}
	_, err = raw.Exec(`INSERT INTO booklet_references (id, booklet_id, depends_on, description)
		VALUES ('r1', 'b1', '"s1"', 'Depends on s1')`)
	if err != nil {
		t.Fatalf("seeding old reference failed: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("raw close failed: %v", err)
	}

	s, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatalf("NewSQLiteStorage on old schema failed: %v", err)
	}
	defer s.Close()

	repo := NewRepository(s.DB())
	got, err := repo.GetBooklet("b1")
	if err != nil {
		t.Fatalf("GetBooklet failed: %v", err)
	}
	if got.Title != "Legacy" || got.Header != "" || got.ShowFooter {
		t.Errorf("booklet = %+v, want legacy row with empty header/footer", got)
	}
	if len(got.Sections) != 1 || got.Sections[0].Title != "Old Section" {
		t.Errorf("sections = %+v, want the legacy section", got.Sections)
	}
	if len(got.Sections) == 1 && got.Sections[0].Generation == nil {
		t.Error("legacy section should load generation metadata")
	}
	if len(got.References) != 1 || got.References[0].ID != "r1" {
		t.Errorf("references = %+v, want the legacy reference", got.References)
	}
	if _, err := repo.GetAllBooklets(); err != nil {
		t.Fatalf("GetAllBooklets failed: %v", err)
	}
}

// TestSQLiteMigrationBackfillsNulls simulates a database migrated by an
// earlier nullable ALTER (leaving NULLs behind) and verifies opening it
// backfills safe defaults instead of failing every read.
func TestSQLiteMigrationBackfillsNulls(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nullable.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = raw.Exec(`CREATE TABLE booklets (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		type TEXT,
		version TEXT,
		status TEXT NOT NULL DEFAULT 'draft',
		audience TEXT,
		instructions TEXT,
		template TEXT,
		header TEXT,
		footer TEXT,
		show_footer INTEGER,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("creating schema failed: %v", err)
	}
	_, err = raw.Exec(`INSERT INTO booklets
		(id, title, type, version, status, audience, instructions, template, created_at, updated_at)
		VALUES ('b1', 'Nulls', '', '', 'draft', '', '', '', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z')`)
	if err != nil {
		t.Fatalf("seeding failed: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("raw close failed: %v", err)
	}

	s, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	got, err := NewRepository(s.DB()).GetBooklet("b1")
	if err != nil {
		t.Fatalf("GetBooklet failed: %v", err)
	}
	if got.Title != "Nulls" || got.Header != "" || got.Footer != "" || got.ShowFooter {
		t.Errorf("booklet = %+v, want backfilled defaults", got)
	}
}

func TestMigrationsRecordedAndIdempotent(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer s.Close()

	versions, err := appliedVersions(s.DB())
	if err != nil {
		t.Fatalf("appliedVersions failed: %v", err)
	}
	if len(versions) != len(migrationList()) {
		t.Fatalf("applied versions = %v, want all %d migrations", versions, len(migrationList()))
	}
	for i, v := range versions {
		if v != i+1 {
			t.Fatalf("applied versions = %v, want sequential from 1", versions)
		}
	}

	if err := Migrate(s.DB()); err != nil {
		t.Fatalf("re-migrate failed: %v", err)
	}
	again, err := appliedVersions(s.DB())
	if err != nil {
		t.Fatalf("appliedVersions failed: %v", err)
	}
	if len(again) != len(versions) {
		t.Errorf("re-migrate recorded %d versions, want %d", len(again), len(versions))
	}
}

func TestParseDriver(t *testing.T) {
	for _, tc := range []struct {
		name    string
		want    Driver
		wantErr bool
	}{
		{"", DriverSQLite, false},
		{"sqlite", DriverSQLite, false},
		{"postgres", DriverPostgres, false},
		{"mysql", DriverMySQL, false},
		{"oracle", "", true},
	} {
		got, err := ParseDriver(tc.name)
		if tc.wantErr && err == nil {
			t.Errorf("ParseDriver(%q) should return an error", tc.name)
		}
		if !tc.wantErr && (err != nil || got != tc.want) {
			t.Errorf("ParseDriver(%q) = %v, %v; want %v, nil", tc.name, got, err, tc.want)
		}
	}
}

func TestOpenSQLiteMemory(t *testing.T) {
	db, err := Open(DriverSQLite, ":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if DriverName(db) == "" {
		t.Error("DriverName should report the dialect")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() failed: %v", err)
	}
	sqlDB.Close()
}
