package booklet

import (
	"errors"
	"fmt"
	"strings"

	"github.com/openbooklet/openbooklet/internal/section"
)

// ParsedMarkdown is the result of splitting a Markdown document into cells.
// FrontMatter and Preamble preserve document-level content verbatim so that
// RenderMarkdown can reproduce the original byte-for-byte for canonical input.
//
// Canonical input rules (whitespace normalization applied by the parser):
//   - Line endings are normalized to "\n".
//   - Blocks (front matter, preamble, sections) are separated by exactly one
//     blank line; extra blank lines between blocks are collapsed.
//   - Headings are normalized to ATX form ("## Title"); closing hash runs and
//     extra inner spacing around the title are not preserved.
//   - Output always ends with a single trailing newline.
//
// Everything else — code blocks, tables, lists, links, Mermaid diagrams — is
// preserved verbatim inside section content. Setext headings are treated as
// body text, not section boundaries.
type ParsedMarkdown struct {
	FrontMatter string
	Preamble    string
	Sections    []Section
}

// ParseMarkdown splits a Markdown document into a preamble and a hierarchy of
// sections. It never fails on content: malformed constructs (unclosed fences,
// stray hashes) are preserved as body text rather than reported as errors.
func ParseMarkdown(input string) (*ParsedMarkdown, error) {
	if input == "" {
		return &ParsedMarkdown{}, nil
	}

	lines := strings.Split(normalizeLineEndings(input), "\n")
	// Drop the artifact of a trailing newline so it is not parsed as content.
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	doc := &ParsedMarkdown{}
	rest := consumeFrontMatter(lines, doc)

	var preamble []string
	var current *Section
	var contentLines []string
	usedIDs := make(map[string]int)
	var stack []*Section
	inFence := fenceState{}

	flush := func() {
		if current != nil {
			current.Content = strings.Join(stripTrailingBlanks(contentLines), "\n")
			contentLines = nil
			current = nil
		}
	}

	for _, line := range rest {
		if fence, ok := parseFenceDelimiter(line); ok {
			inFence = inFence.next(fence)
			if current == nil {
				preamble = append(preamble, line)
			} else {
				contentLines = append(contentLines, line)
			}
			continue
		}
		if !inFence.open {
			if level, title, ok := parseATXHeading(line); ok {
				flush()
				sec := &Section{
					ID:     uniqueSectionID(usedIDs, slugify(title)),
					Title:  title,
					Level:  level,
					Status: section.SectionStatusDraft,
				}
				for len(stack) > 0 && stack[len(stack)-1].Level >= level {
					stack = stack[:len(stack)-1]
				}
				if len(stack) > 0 {
					parentID := stack[len(stack)-1].ID
					sec.ParentID = &parentID
				}
				stack = append(stack, sec)
				sec.Position = len(doc.Sections)
				doc.Sections = append(doc.Sections, *sec)
				current = &doc.Sections[len(doc.Sections)-1]
				continue
			}
		}
		if current == nil {
			preamble = append(preamble, line)
		} else {
			contentLines = append(contentLines, line)
		}
	}
	flush()

	doc.Preamble = strings.Join(stripTrailingBlanks(preamble), "\n")
	return doc, nil
}

// ExportMarkdown renders sections to a canonical Markdown document.
func ExportMarkdown(sections []Section) (string, error) {
	return RenderMarkdown(&ParsedMarkdown{Sections: sections})
}

// ImportMarkdown parses uploaded Markdown into new sections on a booklet.
// Heading-less documents become a single top-level section named after the
// file. Leading front matter and preamble fold into the first section so no
// uploaded text is lost. Imported sections start as drafts with fresh IDs
// that cannot collide with existing sections.
func (s *Service) ImportMarkdown(bookletID, filename, content string) ([]Section, error) {
	b, err := s.repo.GetBooklet(bookletID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("nothing to import")
	}

	doc, err := ParseMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("parsing markdown: %w", err)
	}
	if len(doc.Sections) == 0 {
		name := strings.TrimSuffix(filename, ".md")
		if name == "" {
			name = "Imported"
		}
		doc.Sections = []Section{{
			Title:   name,
			Level:   1,
			Content: strings.TrimSpace(content),
		}}
	} else if leading := leadingText(doc); leading != "" {
		doc.Sections[0].Content = leading + "\n\n" + doc.Sections[0].Content
	}

	used := make(map[string]int, len(b.Sections))
	for i := range b.Sections {
		used[b.Sections[i].ID]++
	}
	now := timeNow()
	sections := make([]Section, 0, len(doc.Sections))
	for i := range doc.Sections {
		sec := &doc.Sections[i]
		sec.ID = uniqueSectionID(used, slugify(sec.Title))
		sec.Status = section.SectionStatusDraft
		sec.CreatedAt = now
		sec.UpdatedAt = now
		if err := sec.Validate(); err != nil {
			return nil, err
		}
		if err := s.repo.SaveSection(bookletID, sec); err != nil {
			return nil, fmt.Errorf("saving imported section %q: %w", sec.ID, err)
		}
		sections = append(sections, *sec)
	}
	return sections, nil
}

// leadingText joins a parsed document's front matter and preamble for
// folding into the first section on import.
func leadingText(doc *ParsedMarkdown) string {
	var parts []string
	if doc.FrontMatter != "" {
		parts = append(parts, doc.FrontMatter)
	}
	if doc.Preamble != "" {
		parts = append(parts, doc.Preamble)
	}
	return strings.Join(parts, "\n\n")
}

// RenderMarkdown renders parsed Markdown back to a document. The output is
// canonical: single blank lines separate blocks and the text ends with exactly
// one trailing newline.
func RenderMarkdown(doc *ParsedMarkdown) (string, error) {
	if doc == nil {
		return "", errors.New("parsed markdown is required")
	}

	var out strings.Builder
	if doc.FrontMatter != "" {
		out.WriteString(doc.FrontMatter)
		out.WriteString("\n")
		if doc.Preamble != "" || len(doc.Sections) > 0 {
			out.WriteString("\n")
		}
	}
	if doc.Preamble != "" {
		out.WriteString(doc.Preamble)
		out.WriteString("\n")
		if len(doc.Sections) > 0 {
			out.WriteString("\n")
		}
	}
	for i := range doc.Sections {
		s := &doc.Sections[i]
		if s.Level < 1 || s.Level > 6 {
			return "", fmt.Errorf("section %q level must be 1-6, got %d", s.ID, s.Level)
		}
		out.WriteString(strings.Repeat("#", s.Level))
		out.WriteString(" ")
		out.WriteString(s.Title)
		out.WriteString("\n")
		// Normalize stored content to canonical form so exports are stable
		// no matter how the content was saved (trailing newlines from
		// editors, leading blanks from parsing).
		if trimmed := strings.Trim(s.Content, "\n"); trimmed != "" {
			out.WriteString("\n")
			out.WriteString(trimmed)
			out.WriteString("\n")
		}
		if i < len(doc.Sections)-1 {
			out.WriteString("\n")
		}
	}
	return out.String(), nil
}

func normalizeLineEndings(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// consumeFrontMatter extracts a leading "---" ... ("---" | "...") block.
// It returns the remaining lines. An unclosed block is not front matter.
func consumeFrontMatter(lines []string, doc *ParsedMarkdown) []string {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return lines
	}
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" || trimmed == "..." {
			doc.FrontMatter = strings.Join(lines[:i+1], "\n")
			rest := lines[i+1:]
			if len(rest) > 0 && strings.TrimSpace(rest[0]) == "" {
				rest = rest[1:]
			}
			return rest
		}
	}
	return lines
}

// parseATXHeading parses an ATX heading line. It reports false for body text,
// including "#" without a following space, empty headings, and lines indented
// four or more spaces (indented code blocks).
func parseATXHeading(line string) (level int, title string, ok bool) {
	stripped := line
	indent := 0
	for indent < len(stripped) && stripped[indent] == ' ' {
		indent++
	}
	if indent > 3 {
		return 0, "", false
	}
	stripped = stripped[indent:]

	hashes := 0
	for hashes < len(stripped) && stripped[hashes] == '#' {
		hashes++
	}
	if hashes == 0 || hashes > 6 {
		return 0, "", false
	}
	rest := stripped[hashes:]
	if rest == "" {
		return 0, "", false
	}
	if rest[0] != ' ' && rest[0] != '\t' {
		return 0, "", false
	}
	rest = strings.Trim(rest, " \t")
	// Strip an optional closing hash run, which must be preceded by spacing.
	if end := len(rest); end > 0 && rest[end-1] == '#' {
		start := end
		for start > 0 && rest[start-1] == '#' {
			start--
		}
		if start > 0 && (rest[start-1] == ' ' || rest[start-1] == '\t') {
			rest = strings.TrimRight(rest[:start], " \t")
		}
	}
	if rest == "" {
		return 0, "", false
	}
	return hashes, rest, true
}

type fenceMark struct {
	char   byte
	length int
}

type fenceState struct {
	open   bool
	char   byte
	length int
}

func (f fenceState) next(d fenceMark) fenceState {
	if !f.open {
		return fenceState{open: true, char: d.char, length: d.length}
	}
	if d.char == f.char && d.length >= f.length {
		return fenceState{}
	}
	return f
}

// parseFenceDelimiter reports whether a line opens or closes a fenced code block.
func parseFenceDelimiter(line string) (fenceMark, bool) {
	stripped := line
	indent := 0
	for indent < len(stripped) && stripped[indent] == ' ' {
		indent++
	}
	if indent > 3 {
		return fenceMark{}, false
	}
	stripped = stripped[indent:]
	if stripped == "" || (stripped[0] != '`' && stripped[0] != '~') {
		return fenceMark{}, false
	}
	char := stripped[0]
	length := 0
	for length < len(stripped) && stripped[length] == char {
		length++
	}
	if length < 3 {
		return fenceMark{}, false
	}
	return fenceMark{char: char, length: length}, true
}

func stripTrailingBlanks(lines []string) []string {
	end := len(lines)
	for end > 0 && strings.Trim(lines[end-1], " \t") == "" {
		end--
	}
	if end == 0 {
		return nil
	}
	return lines[:end]
}

func slugify(title string) string {
	var out strings.Builder
	prevHyphen := true // avoid a leading hyphen
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)
			prevHyphen = false
		default:
			if !prevHyphen {
				out.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	slug := strings.Trim(out.String(), "-")
	if slug == "" {
		slug = "section"
	}
	return slug
}

func uniqueSectionID(used map[string]int, slug string) string {
	used[slug]++
	if used[slug] == 1 {
		return slug
	}
	return fmt.Sprintf("%s-%d", slug, used[slug])
}
