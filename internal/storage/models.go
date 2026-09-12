package storage

import (
	"time"

	"github.com/openbooklet/openbooklet/internal/booklet"
)

// GORM models mirror the legacy tables column-for-column so existing
// databases keep working. Timestamps stay RFC3339 strings and string lists
// stay quoted-CSV, exactly as the previous raw-SQL layer stored them.
// Domain structs remain persistence-free; all mapping lives here.

// bookletModel maps the booklets table.
type bookletModel struct {
	ID           string `gorm:"column:id;primaryKey"`
	Title        string `gorm:"column:title;not null"`
	Type         string `gorm:"column:type"`
	Version      string `gorm:"column:version"`
	Status       string `gorm:"column:status;not null"`
	Audience     string `gorm:"column:audience"`
	Instructions string `gorm:"column:instructions"`
	Template     string `gorm:"column:template"`
	Header       string `gorm:"column:header"`
	Footer       string `gorm:"column:footer"`
	ShowFooter   int64  `gorm:"column:show_footer"`
	CreatedAt    string `gorm:"column:created_at;not null"`
	UpdatedAt    string `gorm:"column:updated_at;not null"`
}

// TableName pins the legacy table name.
func (bookletModel) TableName() string { return "booklets" }

// sectionModel maps the sections table.
type sectionModel struct {
	ID                  string   `gorm:"column:id;primaryKey"`
	BookletID           string   `gorm:"column:booklet_id;primaryKey;index"`
	ParentID            string   `gorm:"column:parent_id"`
	Title               string   `gorm:"column:title;not null"`
	Level               int      `gorm:"column:level;not null"`
	Position            int      `gorm:"column:position"`
	Prompt              string   `gorm:"column:prompt"`
	Content             string   `gorm:"column:content"`
	Dependencies        string   `gorm:"column:dependencies"`
	ContextRefs         string   `gorm:"column:context_refs"`
	Status              string   `gorm:"column:status;not null"`
	GenerationProvider  string   `gorm:"column:provider"`
	GenerationModel     string   `gorm:"column:model"`
	GenerationPrompt    string   `gorm:"column:prompt_snapshot"`
	GenerationCtxRefs   string   `gorm:"column:generation_context_refs"`
	GenerationTemp      *float64 `gorm:"column:temperature"`
	GenerationMaxTokens int64    `gorm:"column:max_tokens"`
	GenerationInputTok  int      `gorm:"column:input_tokens"`
	GenerationOutputTok int      `gorm:"column:output_tokens"`
	GenerationDurMs     int64    `gorm:"column:duration_ms"`
	CreatedAt           string   `gorm:"column:created_at;not null"`
	UpdatedAt           string   `gorm:"column:updated_at;not null"`
}

// TableName pins the legacy table name.
func (sectionModel) TableName() string { return "sections" }

// contentVersionModel maps the content_versions table (read-only; history
// rows are loaded but never written, matching prior behavior).
type contentVersionModel struct {
	ID          uint   `gorm:"column:id;primaryKey;autoIncrement"`
	SectionID   string `gorm:"column:section_id;index"`
	BookletID   string `gorm:"column:booklet_id;index"`
	Version     int    `gorm:"column:version"`
	Content     string `gorm:"column:content"`
	Status      string `gorm:"column:status"`
	Description string `gorm:"column:description"`
	CreatedAt   string `gorm:"column:created_at"`
}

// TableName pins the legacy table name.
func (contentVersionModel) TableName() string { return "content_versions" }

// referenceModel maps the booklet_references table.
type referenceModel struct {
	ID          string `gorm:"column:id;primaryKey"`
	BookletID   string `gorm:"column:booklet_id;index"`
	DependsOn   string `gorm:"column:depends_on"`
	Description string `gorm:"column:description"`
}

// TableName pins the legacy table name.
func (referenceModel) TableName() string { return "booklet_references" }

func bookletToDomain(m *bookletModel) *booklet.Booklet {
	return &booklet.Booklet{
		ID:           m.ID,
		Title:        m.Title,
		Type:         m.Type,
		Version:      m.Version,
		Status:       booklet.BookletStatus(m.Status),
		Audience:     m.Audience,
		Instructions: m.Instructions,
		Template:     m.Template,
		Header:       m.Header,
		Footer:       m.Footer,
		ShowFooter:   m.ShowFooter != 0,
		CreatedAt:    parseTime(m.CreatedAt),
		UpdatedAt:    parseTime(m.UpdatedAt),
	}
}

func bookletToModel(b *booklet.Booklet, createdAt, updatedAt string) *bookletModel {
	return &bookletModel{
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
		ShowFooter:   boolToInt(b.ShowFooter),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func sectionToDomain(m *sectionModel) *booklet.Section {
	var parentID *string
	if m.ParentID != "" {
		parentID = &m.ParentID
	}
	return &booklet.Section{
		ID:           m.ID,
		ParentID:     parentID,
		Title:        m.Title,
		Level:        m.Level,
		Position:     m.Position,
		Prompt:       m.Prompt,
		Content:      m.Content,
		Dependencies: parseJSONArray(m.Dependencies),
		ContextRefs:  parseJSONArray(m.ContextRefs),
		Status:       booklet.SectionStatus(m.Status),
		Generation: &booklet.GenerationMetadata{
			Provider:     m.GenerationProvider,
			Model:        m.GenerationModel,
			Prompt:       m.GenerationPrompt,
			ContextRefs:  parseJSONArray(m.GenerationCtxRefs),
			Temperature:  m.GenerationTemp,
			MaxTokens:    intPtrOrNil(m.GenerationMaxTokens),
			InputTokens:  m.GenerationInputTok,
			OutputTokens: m.GenerationOutputTok,
			Duration:     time.Duration(m.GenerationDurMs) * time.Millisecond,
			CreatedAt:    parseTime(m.CreatedAt),
		},
		CreatedAt: parseTime(m.CreatedAt),
		UpdatedAt: parseTime(m.UpdatedAt),
	}
}

func sectionToModel(bookletID string, sec *booklet.Section, createdAt, updatedAt string) *sectionModel {
	parentID := ""
	if sec.ParentID != nil {
		parentID = *sec.ParentID
	}
	m := &sectionModel{
		ID:           sec.ID,
		BookletID:    bookletID,
		ParentID:     parentID,
		Title:        sec.Title,
		Level:        sec.Level,
		Position:     sec.Position,
		Prompt:       sec.Prompt,
		Content:      sec.Content,
		Dependencies: joinStrings(sec.Dependencies),
		ContextRefs:  joinStrings(sec.ContextRefs),
		Status:       string(sec.Status),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	if sec.Generation != nil {
		m.GenerationProvider = sec.Generation.Provider
		m.GenerationModel = sec.Generation.Model
		m.GenerationPrompt = sec.Generation.Prompt
		m.GenerationCtxRefs = joinStrings(sec.Generation.ContextRefs)
		m.GenerationTemp = sec.Generation.Temperature
		if sec.Generation.MaxTokens != nil {
			m.GenerationMaxTokens = int64(*sec.Generation.MaxTokens)
		}
		m.GenerationInputTok = sec.Generation.InputTokens
		m.GenerationOutputTok = sec.Generation.OutputTokens
		m.GenerationDurMs = int64(sec.Generation.Duration / time.Millisecond)
	}
	return m
}

func referenceToDomain(m *referenceModel) booklet.Reference {
	return booklet.Reference{
		ID:          m.ID,
		DependsOn:   parseJSONArray(m.DependsOn),
		Description: m.Description,
	}
}

func historyToDomain(m *contentVersionModel) booklet.ContentVersion {
	return booklet.ContentVersion{
		Version:     m.Version,
		Content:     m.Content,
		Status:      booklet.SectionStatus(m.Status),
		Description: m.Description,
		CreatedAt:   parseTime(m.CreatedAt),
	}
}
