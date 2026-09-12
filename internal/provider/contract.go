package provider

import (
	"context"
	"strings"
	"testing"
	"time"
)

// NeverRespondsModel is the model name a contract-test fake must treat as a
// request that sends response headers plus one token, then blocks until the
// client disconnects. It exercises mid-stream cancellation (StreamCancel) and
// deadline handling (ChatTimeout) in every Provider implementation.
const NeverRespondsModel = "never-responds"

// ContractHarness describes a Provider wired to a fake backend for the shared
// behavioral suite. The fake must return ChatText from both Chat and the
// concatenated Stream tokens, advertise ModelID from Models, and for
// NeverRespondsModel send response headers plus one token before hanging.
type ContractHarness struct {
	Provider Provider
	ChatText string
	ModelID  string
}

// VerifyContract runs the shared behavioral suite every Provider must pass:
// content delivery, streaming concatenation, model listing, stream
// cancellation, and chat timeout. Call it from each provider's tests with a
// fake backend speaking that provider's protocol.
func VerifyContract(t *testing.T, h ContractHarness) {
	t.Helper()
	if h.Provider == nil {
		t.Fatal("contract harness provider is required")
	}

	t.Run("Chat", func(t *testing.T) {
		resp, err := h.Provider.Chat(context.Background(), Request{
			Model:    h.ModelID,
			Messages: []Message{{Role: "user", Content: "Say hello."}},
		})
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}
		if resp.Content != h.ChatText {
			t.Errorf("Chat content = %q, want %q", resp.Content, h.ChatText)
		}
		if resp.UsageInput < 0 || resp.UsageOutput < 0 {
			t.Errorf("negative token usage: %+v", resp)
		}
	})

	t.Run("Stream", func(t *testing.T) {
		ch, err := h.Provider.Stream(context.Background(), Request{
			Model:    h.ModelID,
			Messages: []Message{{Role: "user", Content: "Say hello."}},
		})
		if err != nil {
			t.Fatalf("Stream failed: %v", err)
		}
		var out strings.Builder
		chunks := 0
		timeout := time.After(10 * time.Second)
		for {
			select {
			case chunk, ok := <-ch:
				if !ok {
					if out.String() != h.ChatText {
						t.Fatalf("stream concatenated = %q, want %q", out.String(), h.ChatText)
					}
					if chunks == 0 {
						t.Error("stream closed without delivering any chunks")
					}
					return
				}
				chunks++
				out.WriteString(chunk.Token)
			case <-timeout:
				t.Fatal("stream did not close within 10s")
			}
		}
	})

	t.Run("Models", func(t *testing.T) {
		models, err := h.Provider.Models(context.Background())
		if err != nil {
			t.Fatalf("Models failed: %v", err)
		}
		for _, m := range models {
			if m.ID == h.ModelID {
				return
			}
		}
		t.Errorf("model %q not advertised: %+v", h.ModelID, models)
	})

	t.Run("StreamCancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// The fake sends response headers plus one token for
		// NeverRespondsModel, then hangs: this mirrors cancelling an
		// established generation mid-stream.
		type streamResult struct {
			ch  <-chan Chunk
			err error
		}
		resCh := make(chan streamResult, 1)
		go func() {
			ch, err := h.Provider.Stream(ctx, Request{
				Model:    NeverRespondsModel,
				Messages: []Message{{Role: "user", Content: "Say hello."}},
			})
			resCh <- streamResult{ch: ch, err: err}
		}()

		var ch <-chan Chunk
		select {
		case res := <-resCh:
			if res.err != nil {
				t.Fatalf("Stream failed: %v", res.err)
			}
			ch = res.ch
		case <-time.After(10 * time.Second):
			t.Fatal("Stream did not return within 10s")
		}

		// Read one token to prove the stream is established.
		select {
		case _, ok := <-ch:
			if !ok {
				t.Fatal("stream closed before delivering a token")
			}
		case <-time.After(10 * time.Second):
			t.Fatal("stream delivered no token within 10s")
		}

		cancel()
		timeout := time.After(10 * time.Second)
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					return
				}
			case <-timeout:
				t.Fatal("stream did not close within 10s of cancellation")
			}
		}
	})

	t.Run("ChatTimeout", func(t *testing.T) {
		_, err := h.Provider.Chat(context.Background(), Request{
			Model:    NeverRespondsModel,
			Messages: []Message{{Role: "user", Content: "Say hello."}},
			Timeout:  100 * time.Millisecond,
		})
		if err == nil {
			t.Fatal("Chat against a hanging backend should time out")
		}
	})
}
