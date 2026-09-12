package provider

import (
	"context"
	"time"
)

// Model represents a available LLM model.
type Model struct {
	ID          string
	Name        string
	Provider    string
	MaxTokens   int
	Description string
}

// Request defines the parameters for an LLM call.
type Request struct {
	Messages    []Message
	Model       string
	Temperature *float64
	MaxTokens   *int
	Stream      bool
	ContextRefs []string
	Timeout     time.Duration
}

// Message represents a single message in the conversation.
type Message struct {
	Role    string // system, user, assistant
	Content string
}

// Chunk represents a streaming token from the LLM.
type Chunk struct {
	Token     string
	Model     string
	CreatedAt time.Time
}

// Response represents a completed LLM response.
type Response struct {
	Content     string
	Model       string
	UsageInput  int
	UsageOutput int
	Duration    time.Duration
	CreatedAt   time.Time
}

// Provider defines the interface for LLM providers.
// Application code depends on this abstraction, never on vendor-specific SDKs.
type Provider interface {
	Chat(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (<-chan Chunk, error)
	Models(ctx context.Context) ([]Model, error)
}
