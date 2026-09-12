package booklet

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// FormatVersion is the current .obk document version.
const FormatVersion = "1"

// MarshalBooklet serializes a booklet to the versioned .obk YAML format.
func MarshalBooklet(b *Booklet) ([]byte, error) {
	if b == nil {
		return nil, errors.New("booklet is required")
	}
	if err := b.Validate(); err != nil {
		return nil, fmt.Errorf("invalid booklet: %w", err)
	}

	doc := obkDocument{
		Version:    FormatVersion,
		Booklet:    newOBKBooklet(b),
		Sections:   newOBKSections(b.Sections),
		References: newOBKReferences(b.References),
	}
	data, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("marshaling booklet: %w", err)
	}
	return data, nil
}

// UnmarshalBooklet parses versioned .obk YAML bytes into a validated booklet.
func UnmarshalBooklet(data []byte) (*Booklet, error) {
	var doc obkDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing booklet: %w", err)
	}
	if err := migrateOBKDocument(&doc); err != nil {
		return nil, err
	}

	b, err := doc.toModel()
	if err != nil {
		return nil, err
	}
	if err := b.Validate(); err != nil {
		return nil, fmt.Errorf("invalid booklet: %w", err)
	}
	return b, nil
}

type obkDocument struct {
	Version    string         `yaml:"version"`
	Booklet    obkBooklet     `yaml:"booklet"`
	Sections   []obkSection   `yaml:"sections,omitempty"`
	References []obkReference `yaml:"references,omitempty"`
}

type obkBooklet struct {
	ID           string `yaml:"id"`
	Title        string `yaml:"title"`
	Type         string `yaml:"type,omitempty"`
	Version      string `yaml:"version,omitempty"`
	Status       string `yaml:"status"`
	Audience     string `yaml:"audience,omitempty"`
	Instructions string `yaml:"instructions,omitempty"`
	Template     string `yaml:"template,omitempty"`
	Header       string `yaml:"header,omitempty"`
	Footer       string `yaml:"footer,omitempty"`
	ShowFooter   bool   `yaml:"show_footer,omitempty"`
	CreatedAt    string `yaml:"created_at,omitempty"`
	UpdatedAt    string `yaml:"updated_at,omitempty"`
}

type obkSection struct {
	ID           string         `yaml:"id"`
	ParentID     *string        `yaml:"parent_id,omitempty"`
	Title        string         `yaml:"title"`
	Level        int            `yaml:"level"`
	Position     int            `yaml:"position"`
	Prompt       string         `yaml:"prompt,omitempty"`
	Content      string         `yaml:"content,omitempty"`
	Dependencies []string       `yaml:"depends_on,omitempty"`
	ContextRefs  []string       `yaml:"context_refs,omitempty"`
	Status       string         `yaml:"status"`
	Generation   *obkGeneration `yaml:"generation,omitempty"`
	History      []obkHistory   `yaml:"history,omitempty"`
	CreatedAt    string         `yaml:"created_at,omitempty"`
	UpdatedAt    string         `yaml:"updated_at,omitempty"`
}

type obkGeneration struct {
	Provider      string   `yaml:"provider,omitempty"`
	Model         string   `yaml:"model,omitempty"`
	PromptSnap    string   `yaml:"prompt_snapshot,omitempty"`
	ContextRefs   []string `yaml:"context_refs,omitempty"`
	Temperature   *float64 `yaml:"temperature,omitempty"`
	MaxTokens     *int     `yaml:"max_tokens,omitempty"`
	InputTokens   int      `yaml:"input_tokens,omitempty"`
	OutputTokens  int      `yaml:"output_tokens,omitempty"`
	DurationNanos int64    `yaml:"duration_nanos,omitempty"`
	CreatedAt     string   `yaml:"created_at,omitempty"`
}

type obkHistory struct {
	Version     int    `yaml:"version"`
	Content     string `yaml:"content"`
	Status      string `yaml:"status"`
	Description string `yaml:"description,omitempty"`
	CreatedAt   string `yaml:"created_at,omitempty"`
}

type obkReference struct {
	ID          string   `yaml:"id"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

// migrateOBKDocument is the explicit entry point for future format migrations.
func migrateOBKDocument(doc *obkDocument) error {
	switch doc.Version {
	case FormatVersion:
		return nil
	default:
		return fmt.Errorf("unsupported .obk version %q: want %q", doc.Version, FormatVersion)
	}
}

func newOBKBooklet(b *Booklet) obkBooklet {
	return obkBooklet{
		ID:           b.ID,
		Title:        b.Title,
		Type:         b.Type,
		Version:      b.Version,
		Status:       string(b.Status),
		Audience:     b.Audience,
		Instructions: b.Instructions,
		Template:     b.Template,
		Header:       b.Header,
		Footer:       b.Footer,
		ShowFooter:   b.ShowFooter,
		CreatedAt:    formatOBKTime(b.CreatedAt),
		UpdatedAt:    formatOBKTime(b.UpdatedAt),
	}
}

func newOBKSections(sections []Section) []obkSection {
	if sections == nil {
		return nil
	}
	out := make([]obkSection, 0, len(sections))
	for i := range sections {
		s := &sections[i]
		out = append(out, obkSection{
			ID:           s.ID,
			ParentID:     s.ParentID,
			Title:        s.Title,
			Level:        s.Level,
			Position:     s.Position,
			Prompt:       s.Prompt,
			Content:      s.Content,
			Dependencies: s.Dependencies,
			ContextRefs:  s.ContextRefs,
			Status:       string(s.Status),
			Generation:   newOBKGeneration(s.Generation),
			History:      newOBKHistory(s.History),
			CreatedAt:    formatOBKTime(s.CreatedAt),
			UpdatedAt:    formatOBKTime(s.UpdatedAt),
		})
	}
	return out
}

func newOBKGeneration(g *GenerationMetadata) *obkGeneration {
	if g == nil {
		return nil
	}
	return &obkGeneration{
		Provider:      g.Provider,
		Model:         g.Model,
		PromptSnap:    g.Prompt,
		ContextRefs:   g.ContextRefs,
		Temperature:   g.Temperature,
		MaxTokens:     g.MaxTokens,
		InputTokens:   g.InputTokens,
		OutputTokens:  g.OutputTokens,
		DurationNanos: int64(g.Duration),
		CreatedAt:     formatOBKTime(g.CreatedAt),
	}
}

func newOBKHistory(history []ContentVersion) []obkHistory {
	if history == nil {
		return nil
	}
	out := make([]obkHistory, 0, len(history))
	for i := range history {
		h := &history[i]
		out = append(out, obkHistory{
			Version:     h.Version,
			Content:     h.Content,
			Status:      string(h.Status),
			Description: h.Description,
			CreatedAt:   formatOBKTime(h.CreatedAt),
		})
	}
	return out
}

func newOBKReferences(references []Reference) []obkReference {
	if references == nil {
		return nil
	}
	out := make([]obkReference, 0, len(references))
	for i := range references {
		r := &references[i]
		out = append(out, obkReference{
			ID:          r.ID,
			DependsOn:   r.DependsOn,
			Description: r.Description,
		})
	}
	return out
}

func (d *obkDocument) toModel() (*Booklet, error) {
	createdAt, err := parseOBKTime(d.Booklet.CreatedAt, "booklet.created_at")
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseOBKTime(d.Booklet.UpdatedAt, "booklet.updated_at")
	if err != nil {
		return nil, err
	}

	sections, err := obkSectionsToModel(d.Sections)
	if err != nil {
		return nil, err
	}
	var references []Reference
	if d.References != nil {
		references = make([]Reference, 0, len(d.References))
		for i := range d.References {
			r := &d.References[i]
			references = append(references, Reference{
				ID:          r.ID,
				DependsOn:   r.DependsOn,
				Description: r.Description,
			})
		}
	}

	return &Booklet{
		ID:           d.Booklet.ID,
		Title:        d.Booklet.Title,
		Type:         d.Booklet.Type,
		Version:      d.Booklet.Version,
		Status:       BookletStatus(d.Booklet.Status),
		Audience:     d.Booklet.Audience,
		Instructions: d.Booklet.Instructions,
		Template:     d.Booklet.Template,
		Header:       d.Booklet.Header,
		Footer:       d.Booklet.Footer,
		ShowFooter:   d.Booklet.ShowFooter,
		References:   references,
		Sections:     sections,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

func obkSectionsToModel(sections []obkSection) ([]Section, error) {
	if sections == nil {
		return nil, nil
	}
	out := make([]Section, 0, len(sections))
	for i := range sections {
		s := &sections[i]
		createdAt, err := parseOBKTime(s.CreatedAt, "sections["+s.ID+"].created_at")
		if err != nil {
			return nil, err
		}
		updatedAt, err := parseOBKTime(s.UpdatedAt, "sections["+s.ID+"].updated_at")
		if err != nil {
			return nil, err
		}
		generation, err := obkGenerationToModel(s.ID, s.Generation)
		if err != nil {
			return nil, err
		}
		history, err := obkHistoryToModel(s.ID, s.History)
		if err != nil {
			return nil, err
		}

		out = append(out, Section{
			ID:           s.ID,
			ParentID:     s.ParentID,
			Title:        s.Title,
			Level:        s.Level,
			Position:     s.Position,
			Prompt:       s.Prompt,
			Content:      s.Content,
			Dependencies: s.Dependencies,
			ContextRefs:  s.ContextRefs,
			Status:       SectionStatus(s.Status),
			Generation:   generation,
			History:      history,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}
	return out, nil
}

func obkGenerationToModel(sectionID string, g *obkGeneration) (*GenerationMetadata, error) {
	if g == nil {
		return nil, nil
	}
	if g.DurationNanos < 0 {
		return nil, fmt.Errorf("sections[%s].generation.duration_nanos must not be negative", sectionID)
	}
	createdAt, err := parseOBKTime(g.CreatedAt, "sections["+sectionID+"].generation.created_at")
	if err != nil {
		return nil, err
	}
	return &GenerationMetadata{
		Provider:     g.Provider,
		Model:        g.Model,
		Prompt:       g.PromptSnap,
		ContextRefs:  g.ContextRefs,
		Temperature:  g.Temperature,
		MaxTokens:    g.MaxTokens,
		InputTokens:  g.InputTokens,
		OutputTokens: g.OutputTokens,
		Duration:     time.Duration(g.DurationNanos),
		CreatedAt:    createdAt,
	}, nil
}

func obkHistoryToModel(sectionID string, history []obkHistory) ([]ContentVersion, error) {
	if history == nil {
		return nil, nil
	}
	out := make([]ContentVersion, 0, len(history))
	for i := range history {
		h := &history[i]
		createdAt, err := parseOBKTime(h.CreatedAt, "sections["+sectionID+"].history.created_at")
		if err != nil {
			return nil, err
		}
		out = append(out, ContentVersion{
			Version:     h.Version,
			Content:     h.Content,
			Status:      SectionStatus(h.Status),
			Description: h.Description,
			CreatedAt:   createdAt,
		})
	}
	return out, nil
}

func formatOBKTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseOBKTime(value, field string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s: %w", field, err)
	}
	return t.UTC(), nil
}
