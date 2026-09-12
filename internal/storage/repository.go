package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/openbooklet/openbooklet/internal/booklet"
)

// SQLiteBookletRepository implements BookletRepository using SQLite.
type SQLiteBookletRepository struct {
	db *sql.DB
}

// NewSQLiteBookletRepository creates a new SQLite-backed booklet repository.
func NewSQLiteBookletRepository(db *sql.DB) *SQLiteBookletRepository {
	return &SQLiteBookletRepository{db: db}
}

// SaveBooklet stores or updates a booklet.
func (r *SQLiteBookletRepository) SaveBooklet(b *booklet.Booklet) error {
	if b == nil {
		return fmt.Errorf("booklet cannot be nil")
	}

	createdAt := formatTime(b.CreatedAt)
	updatedAt := formatTime(b.UpdatedAt)
	if createdAt == "" {
		createdAt = formatTime(time.Now())
		b.CreatedAt = time.Now()
	}
	if updatedAt == "" {
		updatedAt = formatTime(time.Now())
		b.UpdatedAt = time.Now()
	}

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO booklets (id, title, type, version, status, audience, instructions, template, header, footer, show_footer, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, b.ID, b.Title, b.Type, b.Version, b.Status, b.Audience, b.Instructions, b.Template, b.Header, b.Footer, boolToInt(b.ShowFooter), createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("saving booklet: %w", err)
	}

	// Delete existing sections and re-insert
	if _, err := r.db.Exec("DELETE FROM sections WHERE booklet_id = ?", b.ID); err != nil {
		return fmt.Errorf("deleting old sections: %w", err)
	}

	for i := range b.Sections {
		if err := r.saveSection(b.ID, &b.Sections[i]); err != nil {
			return err
		}
	}

	// Delete old booklet_references and re-insert
	if _, err := r.db.Exec("DELETE FROM booklet_references WHERE booklet_id = ?", b.ID); err != nil {
		return fmt.Errorf("deleting old booklet_references: %w", err)
	}
	for i := range b.References {
		ref := b.References[i]
		dependsOn := joinStrings(ref.DependsOn)
		_, err := r.db.Exec(`
			INSERT OR REPLACE INTO booklet_references (id, booklet_id, depends_on, description)
			VALUES (?, ?, ?, ?)
		`, ref.ID, b.ID, dependsOn, ref.Description)
		if err != nil {
			return fmt.Errorf("saving reference: %w", err)
		}
	}

	return nil
}

// saveSection stores a single section.
func (r *SQLiteBookletRepository) saveSection(bookletID string, sec *booklet.Section) error {
	parentID := ""
	if sec.ParentID != nil {
		parentID = *sec.ParentID
	}

	dependencies := joinStrings(sec.Dependencies)
	contextRefs := joinStrings(sec.ContextRefs)

	var temperature *float64
	if sec.Generation != nil && sec.Generation.Temperature != nil {
		temperature = sec.Generation.Temperature
	}

	var maxTokens int64
	if sec.Generation != nil && sec.Generation.MaxTokens != nil {
		maxTokens = int64(*sec.Generation.MaxTokens)
	}

	var durationMs int64
	if sec.Generation != nil {
		durationMs = int64(sec.Generation.Duration / time.Millisecond)
	}

	var inputTokens, outputTokens int
	if sec.Generation != nil {
		inputTokens = sec.Generation.InputTokens
		outputTokens = sec.Generation.OutputTokens
	}

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO sections
		(id, booklet_id, parent_id, title, level, prompt, content, dependencies, context_refs, status, provider, model, prompt_snapshot, temperature, max_tokens, input_tokens, output_tokens, duration_ms, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sec.ID, bookletID, parentID, sec.Title, sec.Level, sec.Prompt, sec.Content,
		dependencies, contextRefs, sec.Status,
		genField(sec.Generation, "Provider"),
		genField(sec.Generation, "Model"),
		genField(sec.Generation, "Prompt"),
		temperature,
		maxTokens,
		inputTokens, outputTokens,
		durationMs,
		formatTime(sec.CreatedAt), formatTime(sec.UpdatedAt))
	return err
}

func genField(g *booklet.GenerationMetadata, field string) string {
	if g == nil {
		return ""
	}
	switch field {
	case "Provider":
		return g.Provider
	case "Model":
		return g.Model
	case "Prompt":
		return g.Prompt
	}
	return ""
}

// GetBooklet retrieves a booklet by ID.
func (r *SQLiteBookletRepository) GetBooklet(id string) (*booklet.Booklet, error) {
	row := r.db.QueryRow(`
		SELECT id, title, type, version, status, audience, instructions, template, header, footer, show_footer, created_at, updated_at
		FROM booklets WHERE id = ?
	`, id)

	b, err := scanBookletRowSingle(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: booklet %q", booklet.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("getting booklet: %w", err)
	}

	// Load sections
	sections, err := r.getSections(id)
	if err != nil {
		return nil, fmt.Errorf("loading sections: %w", err)
	}
	b.Sections = sections

	// Load booklet_references
	refs, err := r.getBookletReferences(id)
	if err != nil {
		return nil, fmt.Errorf("loading booklet_references: %w", err)
	}
	b.References = refs

	return b, nil
}

// getSections loads all sections for a booklet.
func (r *SQLiteBookletRepository) getSections(bookletID string) ([]booklet.Section, error) {
	rows, err := r.db.Query(`
		SELECT id, booklet_id, parent_id, title, level, prompt, content, dependencies, context_refs, status, provider, model, prompt_snapshot, temperature, max_tokens, input_tokens, output_tokens, duration_ms, created_at, updated_at
		FROM sections WHERE booklet_id = ? ORDER BY id
	`, bookletID)
	if err != nil {
		return nil, fmt.Errorf("querying sections: %w", err)
	}
	defer rows.Close()

	var sections []booklet.Section
	for rows.Next() {
		sec := &booklet.Section{}
		if err := scanSectionRow(rows, sec); err != nil {
			return nil, fmt.Errorf("scanning section: %w", err)
		}
		sec.History = r.loadHistory(bookletID, sec.ID)
		sections = append(sections, *sec)
	}

	return sections, nil
}

// loadHistory loads content versions for a section.
func (r *SQLiteBookletRepository) loadHistory(bookletID, sectionID string) []booklet.ContentVersion {
	rows, err := r.db.Query(`
		SELECT version, content, status, description, created_at
		FROM content_versions WHERE booklet_id = ? AND section_id = ? ORDER BY version
	`, bookletID, sectionID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var history []booklet.ContentVersion
	for rows.Next() {
		var (
			version                                 int
			content, status, description, createdAt string
		)
		if err := rows.Scan(&version, &content, &status, &description, &createdAt); err != nil {
			continue
		}
		history = append(history, booklet.ContentVersion{
			Version:     version,
			Content:     content,
			Status:      booklet.SectionStatus(status),
			Description: description,
			CreatedAt:   parseTime(createdAt),
		})
	}

	return history
}

// GetAllBooklets returns all stored booklets.
func (r *SQLiteBookletRepository) GetAllBooklets() ([]*booklet.Booklet, error) {
	rows, err := r.db.Query(`
		SELECT id, title, type, version, status, audience, instructions, template, header, footer, show_footer, created_at, updated_at
		FROM booklets ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying booklets: %w", err)
	}
	defer rows.Close()

	var booklets []*booklet.Booklet
	for rows.Next() {
		b, err := scanBookletRow(rows)
		if err != nil {
			return nil, err
		}
		booklets = append(booklets, b)
	}

	return booklets, nil
}

// DeleteBooklet removes a booklet by ID.
func (r *SQLiteBookletRepository) DeleteBooklet(id string) error {
	result, err := r.db.Exec("DELETE FROM booklets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting booklet: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("%w: booklet %q", booklet.ErrNotFound, id)
	}
	return nil
}

// SaveSection stores or updates a section within a booklet.
func (r *SQLiteBookletRepository) SaveSection(bookletID string, sec *booklet.Section) error {
	if err := r.saveSection(bookletID, sec); err != nil {
		return fmt.Errorf("saving section: %w", err)
	}
	return nil
}

// GetSection retrieves a section by booklet ID and section ID.
func (r *SQLiteBookletRepository) GetSection(bookletID, sectionID string) (*booklet.Section, error) {
	row := r.db.QueryRow(`
		SELECT id, booklet_id, parent_id, title, level, prompt, content, dependencies, context_refs, status, provider, model, prompt_snapshot, temperature, max_tokens, input_tokens, output_tokens, duration_ms, created_at, updated_at
		FROM sections WHERE id = ? AND booklet_id = ?
	`, sectionID, bookletID)

	sec := &booklet.Section{}
	if err := scanSectionRowSingle(row, sec); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: section %q in booklet %q", booklet.ErrNotFound, sectionID, bookletID)
		}
		return nil, fmt.Errorf("getting section: %w", err)
	}
	sec.History = r.loadHistory(bookletID, sec.ID)

	return sec, nil
}

// DeleteSection removes a section from a booklet.
func (r *SQLiteBookletRepository) DeleteSection(bookletID, sectionID string) error {
	result, err := r.db.Exec("DELETE FROM sections WHERE id = ? AND booklet_id = ?", sectionID, bookletID)
	if err != nil {
		return fmt.Errorf("deleting section: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("%w: section %q in booklet %q", booklet.ErrNotFound, sectionID, bookletID)
	}
	return nil
}

// getBookletReferences loads booklet_references for a booklet.
func (r *SQLiteBookletRepository) getBookletReferences(bookletID string) ([]booklet.Reference, error) {
	rows, err := r.db.Query(`
		SELECT id, depends_on, description FROM booklet_references WHERE booklet_id = ?
	`, bookletID)
	if err != nil {
		return nil, fmt.Errorf("querying booklet_references: %w", err)
	}
	defer rows.Close()

	var refs []booklet.Reference
	for rows.Next() {
		var id, dependsOn, description string
		if err := rows.Scan(&id, &dependsOn, &description); err != nil {
			continue
		}
		refs = append(refs, booklet.Reference{
			ID:          id,
			DependsOn:   parseJSONArray(dependsOn),
			Description: description,
		})
	}

	return refs, nil
}

// scanBookletRow scans *sql.Rows into a Booklet (for GetAllBooklets).
func scanBookletRow(rows *sql.Rows) (*booklet.Booklet, error) {
	var (
		id, title, bookletType, version, status, audience, instructions, template, header, footer, createdAt, updatedAt string
		showFooter                                                                                                      int64
	)
	if err := rows.Scan(&id, &title, &bookletType, &version, &status, &audience, &instructions, &template, &header, &footer, &showFooter, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	return &booklet.Booklet{
		ID:           id,
		Title:        title,
		Type:         bookletType,
		Version:      version,
		Status:       booklet.BookletStatus(status),
		Audience:     audience,
		Instructions: instructions,
		Template:     template,
		Header:       header,
		Footer:       footer,
		ShowFooter:   showFooter != 0,
		CreatedAt:    parseTime(createdAt),
		UpdatedAt:    parseTime(updatedAt),
	}, nil
}

// scanSectionRow scans *sql.Rows into a Section (for getSections).
func scanSectionRow(rows *sql.Rows, sec *booklet.Section) error {
	var (
		id, bookletID, parentID, title                     string
		level                                              int
		prompt, content, dependencies, contextRefs, status string
		provider, model, promptSnapshot                    string
		temperature                                        *float64
		maxTokens, inputTokens, outputTokens, durationMs   int64
		createdAt, updatedAt                               string
	)
	if err := rows.Scan(&id, &bookletID, &parentID, &title, &level, &prompt, &content,
		&dependencies, &contextRefs, &status, &provider, &model, &promptSnapshot,
		&temperature, &maxTokens, &inputTokens, &outputTokens, &durationMs,
		&createdAt, &updatedAt); err != nil {
		return err
	}

	sec.ID = id
	sec.ParentID = ptrOrNil(parentID)
	sec.Title = title
	sec.Level = level
	sec.Prompt = prompt
	sec.Content = content
	sec.Dependencies = parseJSONArray(dependencies)
	sec.ContextRefs = parseJSONArray(contextRefs)
	sec.Status = booklet.SectionStatus(status)
	sec.Generation = &booklet.GenerationMetadata{
		Provider:     provider,
		Model:        model,
		Prompt:       promptSnapshot,
		Temperature:  temperature,
		MaxTokens:    intPtrOrNil(maxTokens),
		InputTokens:  int(inputTokens),
		OutputTokens: int(outputTokens),
		Duration:     time.Duration(durationMs) * time.Millisecond,
		CreatedAt:    parseTime(createdAt),
	}
	sec.CreatedAt = parseTime(createdAt)
	sec.UpdatedAt = parseTime(updatedAt)

	return nil
}

// scanBookletRowSingle scans a *sql.Row into a Booklet.
func scanBookletRowSingle(row *sql.Row) (*booklet.Booklet, error) {
	var (
		id, title, bookletType, version, status, audience, instructions, template, header, footer, createdAt, updatedAt string
		showFooter                                                                                                      int64
	)
	if err := row.Scan(&id, &title, &bookletType, &version, &status, &audience, &instructions, &template, &header, &footer, &showFooter, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	return &booklet.Booklet{
		ID:           id,
		Title:        title,
		Type:         bookletType,
		Version:      version,
		Status:       booklet.BookletStatus(status),
		Audience:     audience,
		Instructions: instructions,
		Template:     template,
		Header:       header,
		Footer:       footer,
		ShowFooter:   showFooter != 0,
		CreatedAt:    parseTime(createdAt),
		UpdatedAt:    parseTime(updatedAt),
	}, nil
}

// scanSectionRowSingle scans a *sql.Row into a Section.
func scanSectionRowSingle(row *sql.Row, sec *booklet.Section) error {
	var (
		id, bookletID, parentID, title                     string
		level                                              int
		prompt, content, dependencies, contextRefs, status string
		provider, model, promptSnapshot                    string
		temperature                                        *float64
		maxTokens, inputTokens, outputTokens, durationMs   int64
		createdAt, updatedAt                               string
	)
	if err := row.Scan(&id, &bookletID, &parentID, &title, &level, &prompt, &content,
		&dependencies, &contextRefs, &status, &provider, &model, &promptSnapshot,
		&temperature, &maxTokens, &inputTokens, &outputTokens, &durationMs,
		&createdAt, &updatedAt); err != nil {
		return err
	}

	sec.ID = id
	sec.ParentID = ptrOrNil(parentID)
	sec.Title = title
	sec.Level = level
	sec.Prompt = prompt
	sec.Content = content
	sec.Dependencies = parseJSONArray(dependencies)
	sec.ContextRefs = parseJSONArray(contextRefs)
	sec.Status = booklet.SectionStatus(status)
	sec.Generation = &booklet.GenerationMetadata{
		Provider:     provider,
		Model:        model,
		Prompt:       promptSnapshot,
		Temperature:  temperature,
		MaxTokens:    intPtrOrNil(maxTokens),
		InputTokens:  int(inputTokens),
		OutputTokens: int(outputTokens),
		Duration:     time.Duration(durationMs) * time.Millisecond,
		CreatedAt:    parseTime(createdAt),
	}
	sec.CreatedAt = parseTime(createdAt)
	sec.UpdatedAt = parseTime(updatedAt)

	return nil
}

// ptrOrNil returns a pointer to s if s is non-empty, otherwise nil.
func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// intPtrOrNil returns a pointer to n if n > 0, otherwise nil.
func intPtrOrNil(n int64) *int {
	if n <= 0 {
		return nil
	}
	p := int(n)
	return &p
}

// parseJSONArray parses a comma-separated string into a string slice.
func parseJSONArray(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	var current string
	inQuote := false
	for _, r := range s {
		if r == '"' {
			inQuote = !inQuote
		} else if r == ',' && !inQuote {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// joinStrings joins a string slice into a comma-separated quoted string.
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ","
		}
		result += `"` + s + `"`
	}
	return result
}

// boolToInt maps a bool to 0/1 for SQLite storage.
func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// formatTime converts time.Time to a string.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// parseTime converts a string to time.Time.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
