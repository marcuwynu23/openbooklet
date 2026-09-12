package booklet

import (
	"context"
	"fmt"
	"strings"

	"github.com/openbooklet/openbooklet/internal/llm"
	"github.com/openbooklet/openbooklet/internal/provider"
	"github.com/openbooklet/openbooklet/internal/section"
)

// bookletSystemPrompt instructs the model to respond with a single structured
// Markdown document that the parser can split into section cells.
const bookletSystemPrompt = `You are a technical documentation writer. ` +
	`Respond with a single Markdown document. ` +
	`Start with a level-1 heading for the document title, ` +
	`then structure the body with ## and ### headings. ` +
	`Do not wrap the document in code fences.`

// GenerateSections generates Markdown from a prompt, parses it into section
// cells, stamps generation metadata, and saves each cell. It returns the
// created sections in document order.
//
// A preamble before the first heading is folded into the first section's
// content so no generated text is lost; front matter is dropped because
// booklet metadata already lives on the Booklet.
func (s *Service) GenerateSections(ctx context.Context, bookletID string, p provider.Provider, model, prompt string) ([]Section, error) {
	if _, err := s.repo.GetBooklet(bookletID); err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	content, meta, err := llm.GenerateText(ctx, p, llm.GenerateRequest{
		Model:        model,
		SystemPrompt: bookletSystemPrompt,
		UserPrompt:   prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("generating booklet content: %w", err)
	}

	doc, err := ParseMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("parsing generated markdown: %w", err)
	}
	if len(doc.Sections) == 0 {
		return nil, fmt.Errorf("generated content contains no headings")
	}

	now := timeNow()
	sections := make([]Section, 0, len(doc.Sections))
	for i := range doc.Sections {
		sec := &doc.Sections[i]
		if i == 0 && doc.Preamble != "" {
			sec.Content = doc.Preamble + "\n\n" + sec.Content
		}
		sec.Prompt = prompt
		sec.Status = section.SectionStatusGenerated
		sec.Generation = &GenerationMetadata{
			Provider:     p.Name(),
			Model:        model,
			Prompt:       prompt,
			InputTokens:  meta.InputTokens,
			OutputTokens: meta.OutputTokens,
			Duration:     meta.Duration,
			CreatedAt:    now,
		}
		sec.CreatedAt = now
		sec.UpdatedAt = now
		if err := sec.Validate(); err != nil {
			return nil, err
		}
		if err := s.repo.SaveSection(bookletID, sec); err != nil {
			return nil, fmt.Errorf("saving generated section %q: %w", sec.ID, err)
		}
		sections = append(sections, *sec)
	}
	return sections, nil
}
