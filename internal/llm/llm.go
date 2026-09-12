// Package llm orchestrates generation requests: prompt assembly, streaming
// event handling, cancellation, and usage tracking on top of provider.Provider.
package llm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/openbooklet/openbooklet/internal/provider"
)

// EventType identifies the kind of a StreamEvent.
type EventType string

const (
	// EventStart is emitted first and carries the generation ID and model.
	EventStart EventType = "start"
	// EventToken carries one generated token.
	EventToken EventType = "token"
	// EventComplete terminates the stream with the full content and usage.
	// Cancelled reports whether the generation was cancelled mid-stream.
	EventComplete EventType = "complete"
)

// StreamEvent is a single generation event. Consumers range over the channel
// until it closes; a final EventComplete is always sent unless the provider
// fails before the stream opens (returned as an error instead).
type StreamEvent struct {
	Type         EventType
	GenerationID string
	Token        string
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
	Duration     time.Duration
	Cancelled    bool
}

// GenerateRequest describes a single generation.
type GenerateRequest struct {
	Model        string
	SystemPrompt string
	UserPrompt   string
	Temperature  *float64
	MaxTokens    *int
	Timeout      time.Duration
}

// ResultMetadata summarizes a completed generation for auditability.
type ResultMetadata struct {
	GenerationID string
	Model        string
	InputTokens  int
	OutputTokens int
	Duration     time.Duration
	Cancelled    bool
}

// Generate streams a generation as start → token* → complete events. The
// channel closes after the complete event, on cancellation (complete with
// Cancelled set), or when the provider stream ends early.
func Generate(ctx context.Context, p provider.Provider, req GenerateRequest) (<-chan StreamEvent, error) {
	if p == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(req.UserPrompt) == "" {
		return nil, fmt.Errorf("user prompt is required")
	}

	ctx, cancel := withTimeout(ctx, req.Timeout)

	messages := make([]provider.Message, 0, 2)
	if strings.TrimSpace(req.SystemPrompt) != "" {
		messages = append(messages, provider.Message{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, provider.Message{Role: "user", Content: req.UserPrompt})

	tokenCh, err := p.Stream(ctx, provider.Request{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		cancel()
		return nil, err
	}

	generationID := newGenerationID()
	events := make(chan StreamEvent)
	go func() {
		defer close(events)
		defer cancel()

		start := time.Now()
		select {
		case events <- StreamEvent{Type: EventStart, GenerationID: generationID, Model: req.Model}:
		case <-ctx.Done():
			events <- StreamEvent{Type: EventComplete, GenerationID: generationID, Model: req.Model, Cancelled: true, Duration: time.Since(start)}
			return
		}

		var content strings.Builder
		for {
			select {
			case <-ctx.Done():
				events <- StreamEvent{
					Type:         EventComplete,
					GenerationID: generationID,
					Model:        req.Model,
					Content:      content.String(),
					Duration:     time.Since(start),
					Cancelled:    true,
				}
				return
			case chunk, ok := <-tokenCh:
				if !ok {
					events <- StreamEvent{
						Type:         EventComplete,
						GenerationID: generationID,
						Model:        req.Model,
						Content:      content.String(),
						Duration:     time.Since(start),
						Cancelled:    ctx.Err() != nil,
					}
					return
				}
				content.WriteString(chunk.Token)
				select {
				case events <- StreamEvent{Type: EventToken, GenerationID: generationID, Model: req.Model, Token: chunk.Token}:
				case <-ctx.Done():
					events <- StreamEvent{
						Type:         EventComplete,
						GenerationID: generationID,
						Model:        req.Model,
						Content:      content.String(),
						Duration:     time.Since(start),
						Cancelled:    true,
					}
					return
				}
			}
		}
	}()
	return events, nil
}

// GenerateText drains Generate and returns the accumulated content.
func GenerateText(ctx context.Context, p provider.Provider, req GenerateRequest) (string, ResultMetadata, error) {
	events, err := Generate(ctx, p, req)
	if err != nil {
		return "", ResultMetadata{}, err
	}

	meta := ResultMetadata{Model: req.Model}
	var content strings.Builder
	start := time.Now()
	for event := range events {
		switch event.Type {
		case EventStart:
			meta.GenerationID = event.GenerationID
		case EventToken:
			content.WriteString(event.Token)
		case EventComplete:
			meta.InputTokens = event.InputTokens
			meta.OutputTokens = event.OutputTokens
			meta.Duration = event.Duration
			meta.Cancelled = event.Cancelled
			if event.Cancelled {
				return content.String(), meta, context.Cause(ctx)
			}
		}
	}
	if meta.Duration == 0 {
		meta.Duration = time.Since(start)
	}
	if err := ctx.Err(); err != nil {
		return content.String(), meta, err
	}
	return content.String(), meta, nil
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

func newGenerationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("gen-%d", time.Now().UnixNano())
	}
	return "gen-" + hex.EncodeToString(b[:])
}
