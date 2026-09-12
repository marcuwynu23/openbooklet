package booklet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMarkdownGoldenRoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "sop.md")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading test document failed: %v", err)
	}

	doc, err := ParseMarkdown(string(input))
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}
	rendered, err := RenderMarkdown(doc)
	if err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}

	if *updateGolden {
		if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
			t.Fatalf("writing golden file failed: %v", err)
		}
		input = []byte(rendered)
	}
	if rendered != string(input) {
		t.Error("RenderMarkdown(ParseMarkdown(doc)) differs from input; Markdown round-trip is lossy")
	}
}

func TestParseMarkdownHierarchy(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("testdata", "sop.md"))
	if err != nil {
		t.Fatalf("reading test document failed: %v", err)
	}
	doc, err := ParseMarkdown(string(input))
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	if doc.FrontMatter == "" {
		t.Error("expected front matter to be preserved")
	}
	if !strings.Contains(doc.Preamble, "generated for parser testing") {
		t.Errorf("expected preamble to be preserved, got %q", doc.Preamble)
	}

	wantTitles := []struct {
		title  string
		level  int
		parent string // empty means top-level
	}{
		{"Kubernetes Deployment SOP", 1, ""},
		{"Purpose", 2, "Kubernetes Deployment SOP"},
		{"Scope", 3, "Purpose"},
		{"Prerequisites", 2, "Kubernetes Deployment SOP"},
		{"Deployment", 2, "Kubernetes Deployment SOP"},
		{"Verify", 3, "Deployment"},
		{"Appendix", 1, ""},
	}
	if len(doc.Sections) != len(wantTitles) {
		t.Fatalf("got %d sections, want %d", len(doc.Sections), len(wantTitles))
	}
	byTitle := make(map[string]*Section)
	for i := range doc.Sections {
		byTitle[doc.Sections[i].Title] = &doc.Sections[i]
		if doc.Sections[i].Position != i {
			t.Errorf("section %q position = %d, want document order %d", doc.Sections[i].Title, doc.Sections[i].Position, i)
		}
	}
	for _, want := range wantTitles {
		sec, ok := byTitle[want.title]
		if !ok {
			t.Errorf("missing section %q", want.title)
			continue
		}
		if sec.Level != want.level {
			t.Errorf("section %q level = %d, want %d", want.title, sec.Level, want.level)
		}
		if want.parent == "" {
			if sec.ParentID != nil {
				t.Errorf("section %q should be top-level", want.title)
			}
			continue
		}
		parent, ok := byTitle[want.parent]
		if !ok {
			t.Errorf("parent %q of %q not found", want.parent, want.title)
			continue
		}
		if sec.ParentID == nil || *sec.ParentID != parent.ID {
			t.Errorf("section %q parent = %v, want %q", want.title, sec.ParentID, parent.ID)
		}
	}
}

func TestParseMarkdownPreservesCodeBlocks(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("testdata", "sop.md"))
	if err != nil {
		t.Fatalf("reading test document failed: %v", err)
	}
	doc, err := ParseMarkdown(string(input))
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	var prereqs *Section
	for i := range doc.Sections {
		if doc.Sections[i].Title == "Prerequisites" {
			prereqs = &doc.Sections[i]
		}
	}
	if prereqs == nil {
		t.Fatal("Prerequisites section not found")
	}
	for _, want := range []string{
		"## also not a heading",
		"```bash",
		"flowchart TD",
		"| kubectl | >=",
		"> # not a heading",
		"# indented code is body text",
	} {
		if !strings.Contains(prereqs.Content, want) {
			t.Errorf("section content missing %q", want)
		}
	}
}

func TestParseMarkdownEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantSections int
		wantTitles   []string
	}{
		{name: "empty", input: "", wantSections: 0},
		{
			name:         "preamble only",
			input:        "Just some text.\nNo headings here.\n",
			wantSections: 0,
		},
		{
			name:         "hash without space is body",
			input:        "#NoSpace\n#AlsoNoSpace\n",
			wantSections: 0,
		},
		{
			name:         "empty hash is body",
			input:        "#\n##\n",
			wantSections: 0,
		},
		{
			name:         "closing hashes stripped",
			input:        "## Title ##\n",
			wantSections: 1,
			wantTitles:   []string{"Title"},
		},
		{
			name:         "duplicate titles get unique ids",
			input:        "# Same\n# Same\n# Same\n",
			wantSections: 3,
			wantTitles:   []string{"Same", "Same", "Same"},
		},
		{
			name:         "unclosed fence keeps rest as body",
			input:        "# A\n```go\n# not a heading\nmore code\n",
			wantSections: 1,
			wantTitles:   []string{"A"},
		},
		{
			name:         "longer closing fence",
			input:        "# A\n````\ncode\n`````\n# B\n",
			wantSections: 2,
			wantTitles:   []string{"A", "B"},
		},
		{
			name:         "unclosed front matter is body",
			input:        "---\ntitle: x\n# A\n",
			wantSections: 1,
			wantTitles:   []string{"A"},
		},
		{
			name:         "skipped level nests under ancestor",
			input:        "# A\n### Deep\n",
			wantSections: 2,
			wantTitles:   []string{"A", "Deep"},
		},
		{
			name:         "crlf normalized",
			input:        "# A\r\n\r\nBody line.\r\n",
			wantSections: 1,
			wantTitles:   []string{"A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseMarkdown(tt.input)
			if err != nil {
				t.Fatalf("ParseMarkdown failed: %v", err)
			}
			if len(doc.Sections) != tt.wantSections {
				t.Fatalf("got %d sections, want %d", len(doc.Sections), tt.wantSections)
			}
			for i, want := range tt.wantTitles {
				if doc.Sections[i].Title != want {
					t.Errorf("section %d title = %q, want %q", i, doc.Sections[i].Title, want)
				}
			}
			if tt.name == "duplicate titles get unique ids" {
				seen := make(map[string]bool)
				for _, s := range doc.Sections {
					if seen[s.ID] {
						t.Errorf("duplicate section ID %q", s.ID)
					}
					seen[s.ID] = true
				}
			}
			if tt.name == "unclosed fence keeps rest as body" {
				if !strings.Contains(doc.Sections[0].Content, "# not a heading") {
					t.Errorf("fenced content lost: %q", doc.Sections[0].Content)
				}
			}
			if tt.name == "crlf normalized" {
				rendered, err := RenderMarkdown(doc)
				if err != nil {
					t.Fatalf("RenderMarkdown failed: %v", err)
				}
				if strings.Contains(rendered, "\r") {
					t.Errorf("rendered output contains CR: %q", rendered)
				}
				if rendered != "# A\n\nBody line.\n" {
					t.Errorf("rendered = %q", rendered)
				}
			}
			// Every parse must re-render without error.
			if _, err := RenderMarkdown(doc); err != nil {
				t.Errorf("RenderMarkdown failed: %v", err)
			}
		})
	}
}

func TestRenderMarkdownErrors(t *testing.T) {
	if _, err := RenderMarkdown(nil); err == nil {
		t.Error("RenderMarkdown(nil) should return an error")
	}
	doc := &ParsedMarkdown{Sections: []Section{{ID: "s1", Title: "S", Level: 7}}}
	if _, err := RenderMarkdown(doc); err == nil {
		t.Error("RenderMarkdown with level 7 should return an error")
	}
}

func TestParseMarkdownNeverPanics(t *testing.T) {
	inputs := []string{
		"```",
		"~~~",
		"---",
		"---\n---",
		"#",
		"####### too many",
		"    # indented",
		"\x00\x01\x02",
		"# \t ",
		"```\n```\n```\n# A\n",
		"| # |\n| - |\n",
	}
	for _, input := range inputs {
		doc, err := ParseMarkdown(input)
		if err != nil {
			t.Errorf("ParseMarkdown(%q) failed: %v", input, err)
		}
		if _, err := RenderMarkdown(doc); err != nil {
			t.Errorf("RenderMarkdown(%q) failed: %v", input, err)
		}
	}
}
