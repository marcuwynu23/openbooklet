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

// SectionEventType identifies the kind of a SectionEvent.
type SectionEventType string

const (
	// SectionEventStart opens a generation stream.
	SectionEventStart SectionEventType = "start"
	// SectionEventToken carries one generated token.
	SectionEventToken SectionEventType = "token"
	// SectionEventComplete ends the stream with the saved sections.
	SectionEventComplete SectionEventType = "complete"
	// SectionEventError ends the stream with a failure message.
	SectionEventError SectionEventType = "error"
)

// SectionEvent is a single booklet-generation event. The stream always ends
// with either a complete event (cells parsed and saved) or an error event.
type SectionEvent struct {
	Type         SectionEventType
	GenerationID string
	Token        string
	Sections     []Section
	Message      string
}

// StreamGeneration streams a booklet generation: start → token* → complete.
// On completion the Markdown is parsed into cells, stamped with generation
// metadata, and saved. Draining the channel to close is the caller's job;
// cancelling ctx aborts the provider stream and yields an error event.
func (s *Service) StreamGeneration(ctx context.Context, bookletID string, p provider.Provider, model, prompt string) (<-chan SectionEvent, error) {
	if _, err := s.repo.GetBooklet(bookletID); err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	tokenEvents, err := llm.Generate(ctx, p, llm.GenerateRequest{
		Model:        model,
		SystemPrompt: bookletSystemPrompt,
		UserPrompt:   prompt,
	})
	if err != nil {
		return nil, err
	}

	events := make(chan SectionEvent)
	go func() {
		defer close(events)
		var content strings.Builder
		var generationID string
		emitError := func(message string) {
			select {
			case events <- SectionEvent{Type: SectionEventError, GenerationID: generationID, Message: message}:
			case <-ctx.Done():
			}
		}
		for event := range tokenEvents {
			switch event.Type {
			case llm.EventStart:
				generationID = event.GenerationID
				select {
				case events <- SectionEvent{Type: SectionEventStart, GenerationID: generationID}:
				case <-ctx.Done():
					emitError(ctx.Err().Error())
					return
				}
			case llm.EventToken:
				content.WriteString(event.Token)
				select {
				case events <- SectionEvent{Type: SectionEventToken, GenerationID: generationID, Token: event.Token}:
				case <-ctx.Done():
					emitError(ctx.Err().Error())
					return
				}
			case llm.EventComplete:
				if event.Cancelled {
					emitError("generation cancelled")
					return
				}
				sections, err := s.saveGenerated(bookletID, p, model, prompt, content.String(), event)
				if err != nil {
					emitError(err.Error())
					return
				}
				select {
				case events <- SectionEvent{Type: SectionEventComplete, GenerationID: generationID, Sections: sections}:
				case <-ctx.Done():
				}
				return
			}
		}
		emitError("generation stream ended unexpectedly")
	}()
	return events, nil
}

// GenerateSections generates Markdown from a prompt, parses it into section
// cells, stamps generation metadata, and saves each cell. It returns the
// created sections in document order.
func (s *Service) GenerateSections(ctx context.Context, bookletID string, p provider.Provider, model, prompt string) ([]Section, error) {
	events, err := s.StreamGeneration(ctx, bookletID, p, model, prompt)
	if err != nil {
		return nil, err
	}
	for event := range events {
		switch event.Type {
		case SectionEventComplete:
			return event.Sections, nil
		case SectionEventError:
			return nil, fmt.Errorf("generation failed: %s", event.Message)
		}
	}
	return nil, fmt.Errorf("generation stream ended unexpectedly")
}

// saveGenerated parses generated Markdown into cells and saves them.
// A preamble before the first heading is folded into the first section's
// content so no generated text is lost; front matter is dropped because
// booklet metadata already lives on the Booklet.
func (s *Service) saveGenerated(bookletID string, p provider.Provider, model, prompt, content string, event llm.StreamEvent) ([]Section, error) {
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
			InputTokens:  event.InputTokens,
			OutputTokens: event.OutputTokens,
			Duration:     event.Duration,
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

// RegenerateMode selects how a cell is rewritten.
type RegenerateMode string

const (
	// RegenerateRewrite rewrites the cell content from its stored prompt.
	RegenerateRewrite RegenerateMode = "regenerate"
	// RegenerateExpand adds detail while keeping the structure.
	RegenerateExpand RegenerateMode = "expand"
	// RegenerateShorten condenses the cell without losing key points.
	RegenerateShorten RegenerateMode = "shorten"
	// RegenerateEdit rewrites the cell following a custom instruction.
	RegenerateEdit RegenerateMode = "edit"
)

// RegenerateSection rewrites a single cell with AI. The current content is
// snapshotted to history first so the edit stays revertible. The cell keeps
// its ID, level, and position; its status returns to generated.
func (s *Service) RegenerateSection(ctx context.Context, bookletID, sectionID string, p provider.Provider, model string, mode RegenerateMode, instruction string) (*Section, error) {
	sec, err := s.repo.GetSection(bookletID, sectionID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("provider is required")
	}

	system, user, err := regeneratePrompt(mode, instruction, sec)
	if err != nil {
		return nil, err
	}
	content, meta, err := llm.GenerateText(ctx, p, llm.GenerateRequest{
		Model:        model,
		SystemPrompt: system,
		UserPrompt:   user,
	})
	if err != nil {
		return nil, fmt.Errorf("regenerating section: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("regeneration returned empty content")
	}

	now := timeNow()
	sec.AddHistoryRecord(sec.Content, sec.Status, "before "+string(mode))
	sec.Content = strings.TrimSpace(content) + "\n"
	sec.Status = section.SectionStatusGenerated
	sec.Generation = &GenerationMetadata{
		Provider:     p.Name(),
		Model:        model,
		Prompt:       user,
		InputTokens:  meta.InputTokens,
		OutputTokens: meta.OutputTokens,
		Duration:     meta.Duration,
		CreatedAt:    now,
	}
	sec.UpdatedAt = now
	if err := sec.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.SaveSection(bookletID, sec); err != nil {
		return nil, fmt.Errorf("saving regenerated section: %w", err)
	}
	return sec, nil
}

func regeneratePrompt(mode RegenerateMode, instruction string, sec *Section) (system, user string, err error) {
	const shape = `Respond with only the rewritten Markdown for this one section. ` +
		`Keep the same heading level. Do not wrap it in code fences.`
	switch mode {
	case RegenerateRewrite:
		system = "You are a technical documentation writer. " + shape
		user = "Rewrite the following Markdown section"
		if strings.TrimSpace(sec.Prompt) != "" {
			user += " following this brief: " + sec.Prompt
		}
		user += ":\n\n" + sec.Content
		return system, user, nil
	case RegenerateExpand:
		system = "You are a technical documentation writer. " + shape
		user = "Expand the following Markdown section with more detail and concrete examples:\n\n" + sec.Content
		return system, user, nil
	case RegenerateShorten:
		system = "You are a technical documentation writer. " + shape
		user = "Condense the following Markdown section without losing key points:\n\n" + sec.Content
		return system, user, nil
	case RegenerateEdit:
		if strings.TrimSpace(instruction) == "" {
			return "", "", fmt.Errorf("edit mode requires an instruction")
		}
		system = "You are a technical documentation writer. " + shape
		user = "Apply this instruction to the following Markdown section: " + instruction + "\n\n" + sec.Content
		return system, user, nil
	default:
		return "", "", fmt.Errorf("unknown regenerate mode %q", mode)
	}
}
