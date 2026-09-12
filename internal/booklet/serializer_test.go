package booklet

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/openbooklet/openbooklet/internal/section"
)

var updateOBKGolden = flag.Bool("update", false, "update .obk golden files")

func obkTestBooklet() *Booklet {
	created := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 9, 12, 12, 30, 0, 0, time.UTC)
	temperature := 0.2
	maxTokens := 1024
	parentID := "purpose"

	return &Booklet{
		ID:           "kubernetes-sop",
		Title:        "Kubernetes Deployment SOP",
		Type:         "sop",
		Version:      "1.0",
		Status:       BookletStatusDraft,
		Audience:     "l1-operations",
		Instructions: "Use formal technical language.\nDo not invent infrastructure information.\n",
		Template:     "sop",
		References: []Reference{
			{ID: "deploy-order", DependsOn: []string{"prerequisites"}, Description: "Deploy only after prerequisites"},
		},
		Sections: []Section{
			{
				ID:          "purpose",
				Title:       "Purpose",
				Level:       2,
				Prompt:      "Explain the purpose of this SOP.\n",
				Content:     "This SOP describes the standard Kubernetes deployment procedure.\n",
				ContextRefs: []string{"cluster-docs"},
				Status:      section.SectionStatusGenerated,
				Generation: &GenerationMetadata{
					Provider:     "test-provider",
					Model:        "test-model",
					Prompt:       "Explain the purpose of this SOP.\n",
					ContextRefs:  []string{"cluster-docs"},
					Temperature:  &temperature,
					MaxTokens:    &maxTokens,
					InputTokens:  128,
					OutputTokens: 256,
					Duration:     1500 * time.Millisecond,
					CreatedAt:    created,
				},
				History: []ContentVersion{
					{
						Version:     1,
						Content:     "This SOP describes deployments.\n",
						Status:      section.SectionStatusGenerated,
						Description: "initial generation",
						CreatedAt:   created,
					},
				},
				CreatedAt: created,
				UpdatedAt: updated,
			},
			{
				ID:           "prerequisites",
				ParentID:     &parentID,
				Title:        "Prerequisites",
				Level:        3,
				Prompt:       "Generate the required prerequisites.\n",
				Content:      "- Kubernetes cluster access\n- kubectl\n- Required permissions\n",
				Dependencies: []string{"purpose"},
				Status:       section.SectionStatusEdited,
				History: []ContentVersion{
					{
						Version:     1,
						Content:     "- kubectl\n",
						Status:      section.SectionStatusGenerated,
						Description: "initial generation",
						CreatedAt:   created,
					},
					{
						Version:     2,
						Content:     "- Kubernetes cluster access\n- kubectl\n- Required permissions\n",
						Status:      section.SectionStatusEdited,
						Description: "human edit",
						CreatedAt:   updated,
					},
				},
				CreatedAt: created,
				UpdatedAt: updated,
			},
		},
		CreatedAt: created,
		UpdatedAt: updated,
	}
}

func TestMarshalBookletGolden(t *testing.T) {
	got, err := MarshalBooklet(obkTestBooklet())
	if err != nil {
		t.Fatalf("MarshalBooklet failed: %v", err)
	}

	path := filepath.Join("testdata", "sop.obk.golden")
	if *updateOBKGolden {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("writing golden file failed: %v", err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file failed: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("MarshalBooklet() output differs from golden file; run with -update to refresh")
	}
}

func TestOBKRoundTrip(t *testing.T) {
	want := obkTestBooklet()

	data, err := MarshalBooklet(want)
	if err != nil {
		t.Fatalf("MarshalBooklet failed: %v", err)
	}
	got, err := UnmarshalBooklet(data)
	if err != nil {
		t.Fatalf("UnmarshalBooklet failed: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip booklet differs:\ngot:  %+v\nwant: %+v", got, want)
	}

	again, err := MarshalBooklet(got)
	if err != nil {
		t.Fatalf("MarshalBooklet failed: %v", err)
	}
	if string(again) != string(data) {
		t.Error("re-marshaled booklet differs; .obk round-trip is not stable")
	}
}

func TestUnmarshalBookletMinimal(t *testing.T) {
	got, err := UnmarshalBooklet([]byte("version: \"1\"\nbooklet:\n  id: minimal\n  title: Minimal\n  status: draft\n"))
	if err != nil {
		t.Fatalf("UnmarshalBooklet failed: %v", err)
	}
	if got.ID != "minimal" || got.Title != "Minimal" || got.Status != BookletStatusDraft {
		t.Errorf("unexpected booklet: %+v", got)
	}
	if got.Sections != nil || got.References != nil {
		t.Errorf("expected nil sections and references, got %+v", got)
	}
}

func TestUnmarshalBookletErrors(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{
			name:    "unsupported version",
			data:    "version: \"2\"\nbooklet:\n  id: b1\n  title: T\n  status: draft\n",
			wantErr: "unsupported .obk version",
		},
		{
			name:    "missing version",
			data:    "booklet:\n  id: b1\n  title: T\n  status: draft\n",
			wantErr: "unsupported .obk version",
		},
		{
			name:    "invalid yaml",
			data:    "version: [\"1\"\n",
			wantErr: "parsing booklet",
		},
		{
			name:    "missing title",
			data:    "version: \"1\"\nbooklet:\n  id: b1\n  status: draft\n",
			wantErr: "booklet title is required",
		},
		{
			name:    "invalid status",
			data:    "version: \"1\"\nbooklet:\n  id: b1\n  title: T\n  status: bogus\n",
			wantErr: "invalid booklet status",
		},
		{
			name:    "invalid section status",
			data:    "version: \"1\"\nbooklet:\n  id: b1\n  title: T\n  status: draft\nsections:\n  - id: s1\n    title: S\n    level: 2\n    status: bogus\n",
			wantErr: "invalid status",
		},
		{
			name:    "invalid time",
			data:    "version: \"1\"\nbooklet:\n  id: b1\n  title: T\n  status: draft\n  created_at: not-a-time\n",
			wantErr: "invalid booklet.created_at",
		},
		{
			name:    "negative duration",
			data:    "version: \"1\"\nbooklet:\n  id: b1\n  title: T\n  status: draft\nsections:\n  - id: s1\n    title: S\n    level: 2\n    status: generated\n    generation:\n      duration_nanos: -1\n",
			wantErr: "must not be negative",
		},
		{
			name:    "unknown parent",
			data:    "version: \"1\"\nbooklet:\n  id: b1\n  title: T\n  status: draft\nsections:\n  - id: s1\n    parent_id: missing\n    title: S\n    level: 2\n    status: draft\n",
			wantErr: "unknown parent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalBooklet([]byte(tt.data))
			if err == nil {
				t.Fatal("UnmarshalBooklet should return an error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMarshalBookletErrors(t *testing.T) {
	if _, err := MarshalBooklet(nil); err == nil {
		t.Error("MarshalBooklet(nil) should return an error")
	}
	if _, err := MarshalBooklet(&Booklet{ID: "b1", Status: BookletStatusDraft}); err == nil {
		t.Error("MarshalBooklet with missing title should return an error")
	}
}
