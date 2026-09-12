package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/openbooklet/openbooklet/internal/provider"
)

// fakeProvider returns canned tokens, optionally hanging to test cancellation.
type fakeProvider struct {
	tokens []string
	hang   bool
	err    error
}

func (f *fakeProvider) Name() string {
	return "fake"
}

func (f *fakeProvider) Chat(_ context.Context, _ provider.Request) (provider.Response, error) {
	return provider.Response{Content: strings.Join(f.tokens, "")}, f.err
}

func (f *fakeProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	if f.err != nil {
		return nil, f.err
	}
	ch := make(chan provider.Chunk)
	go func() {
		defer close(ch)
		for _, tok := range f.tokens {
			select {
			case <-ctx.Done():
				return
			case ch <- provider.Chunk{Token: tok}:
			}
		}
		if f.hang {
			<-ctx.Done()
		}
	}()
	return ch, nil
}

func (f *fakeProvider) Models(_ context.Context) ([]provider.Model, error) {
	return []provider.Model{{ID: "fake", Name: "fake"}}, nil
}

func TestGenerateEventSequence(t *testing.T) {
	p := &fakeProvider{tokens: []string{"Hello", ", ", "world"}}
	events, err := Generate(context.Background(), p, GenerateRequest{
		Model:      "fake",
		UserPrompt: "Say hello.",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var got []StreamEvent
	timeout := time.After(5 * time.Second)
loop:
	for {
		select {
		case event, ok := <-events:
			if !ok {
				break loop
			}
			got = append(got, event)
		case <-timeout:
			t.Fatal("events channel did not close")
		}
	}

	if len(got) != 5 {
		t.Fatalf("got %d events, want 5 (start + 3 tokens + complete)", len(got))
	}
	if got[0].Type != EventStart || got[0].GenerationID == "" {
		t.Errorf("first event = %+v, want start with generation ID", got[0])
	}
	for i, want := range []string{"Hello", ", ", "world"} {
		if got[i+1].Type != EventToken || got[i+1].Token != want {
			t.Errorf("event %d = %+v, want token %q", i+1, got[i+1], want)
		}
	}
	last := got[len(got)-1]
	if last.Type != EventComplete || last.Content != "Hello, world" || last.Cancelled {
		t.Errorf("last event = %+v, want non-cancelled complete", last)
	}
}

func TestGenerateText(t *testing.T) {
	p := &fakeProvider{tokens: []string{"a", "b"}}
	content, meta, err := GenerateText(context.Background(), p, GenerateRequest{
		SystemPrompt: "Be brief.",
		UserPrompt:   "Go.",
	})
	if err != nil {
		t.Fatalf("GenerateText failed: %v", err)
	}
	if content != "ab" {
		t.Errorf("content = %q, want %q", content, "ab")
	}
	if meta.GenerationID == "" || meta.Cancelled {
		t.Errorf("meta = %+v", meta)
	}
}

func TestGenerateValidation(t *testing.T) {
	p := &fakeProvider{}
	if _, err := Generate(context.Background(), nil, GenerateRequest{UserPrompt: "hi"}); err == nil {
		t.Error("nil provider should return an error")
	}
	if _, err := Generate(context.Background(), p, GenerateRequest{}); err == nil {
		t.Error("empty prompt should return an error")
	}
	if _, err := Generate(context.Background(), &fakeProvider{err: errors.New("boom")},
		GenerateRequest{UserPrompt: "hi"}); err == nil {
		t.Error("provider error should propagate")
	}
}

func TestGenerateCancel(t *testing.T) {
	p := &fakeProvider{tokens: []string{"first"}, hang: true}
	ctx, cancel := context.WithCancel(context.Background())
	events, err := Generate(ctx, p, GenerateRequest{UserPrompt: "Go."})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Read the first token to prove the stream is established, then cancel.
	select {
	case event := <-events:
		if event.Type == EventComplete {
			t.Fatal("stream completed before cancellation")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no event within 5s")
	}
	cancel()

	timeout := time.After(5 * time.Second)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Type == EventComplete && !event.Cancelled {
				t.Error("complete after cancel should be marked cancelled")
				return
			}
			if event.Type == EventComplete {
				// Drain until close.
				for range events {
				}
				return
			}
		case <-timeout:
			t.Fatal("events did not close after cancel")
		}
	}
}

func TestGenerateTextCancel(t *testing.T) {
	p := &fakeProvider{hang: true}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	_, _, err := GenerateText(ctx, p, GenerateRequest{UserPrompt: "Go."})
	if err == nil {
		t.Error("cancelled generation should return an error")
	}
}
