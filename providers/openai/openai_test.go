package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openbooklet/openbooklet/internal/provider"
)

const (
	testModel = "test-model"
	testText  = "Hello, world!"
)

var testTokens = []string{"Hello", ", ", "world", "!"}

// fakeServer speaks the OpenAI chat completions protocol. The "never-responds"
// model sends headers plus one token then hangs; the "fail" model returns
// 500; the "flaky" model returns 429 on its first request to exercise retries.
func fakeServer(t *testing.T, hits *atomic.Int64) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		var req struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		switch req.Model {
		case provider.NeverRespondsModel:
			// Send headers plus one token, then hang: exercises
			// mid-stream cancellation and chat timeouts.
			w.Header().Set("Content-Type", "application/json")
			if req.Stream {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = fmt.Fprintf(w, "data: {\"model\":%q,\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n", testModel)
			} else {
				_, _ = fmt.Fprintf(w, `{"model":%q,"choices":[{"message":{"content":"partial"}}]}`, testModel)
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			<-r.Context().Done()
			return
		case "fail":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"boom","type":"server_error"}}`))
			return
		case "flaky":
			if hits == nil || hits.Load() == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if !req.Stream {
			_, _ = fmt.Fprintf(w, `{"model":%q,"choices":[{"message":{"content":%q}}],`+
				`"usage":{"prompt_tokens":5,"completion_tokens":7}}`, testModel, testText)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flush", http.StatusInternalServerError)
			return
		}
		for _, tok := range testTokens {
			_, _ = fmt.Fprintf(w, "data: {\"model\":%q,\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", testModel, tok)
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	})

	mux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":[{"id":%q}]}`, testModel)
	})

	return httptest.NewServer(mux)
}

func TestContract(t *testing.T) {
	server := fakeServer(t, nil)
	defer server.Close()

	provider.VerifyContract(t, provider.ContractHarness{
		Provider: New(Config{Endpoint: server.URL, MaxRetries: -1}),
		ChatText: testText,
		ModelID:  testModel,
	})
}

func TestChatServerError(t *testing.T) {
	server := fakeServer(t, nil)
	defer server.Close()

	client := New(Config{Endpoint: server.URL, MaxRetries: -1})
	_, err := client.Chat(t.Context(), provider.Request{
		Model:    "fail",
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("Chat should return an error on 500")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %q, want server message", err.Error())
	}
}

func TestStreamServerError(t *testing.T) {
	server := fakeServer(t, nil)
	defer server.Close()

	client := New(Config{Endpoint: server.URL, MaxRetries: -1})
	_, err := client.Stream(t.Context(), provider.Request{
		Model:    "fail",
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("Stream should return an error on 500")
	}
}

func TestChatRetriesRateLimit(t *testing.T) {
	var hits atomic.Int64
	server := fakeServer(t, &hits)
	defer server.Close()

	client := New(Config{Endpoint: server.URL})
	resp, err := client.Chat(t.Context(), provider.Request{
		Model:    "flaky",
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Content != testText {
		t.Errorf("content = %q, want %q", resp.Content, testText)
	}
	if hits.Load() != 2 {
		t.Errorf("requests = %d, want 2 (one retry)", hits.Load())
	}
}

func TestChatAuthorizationHeader(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"content":"x"}}]}`))
	}))
	defer server.Close()

	client := New(Config{Endpoint: server.URL, APIKey: "sk-test", MaxRetries: -1})
	if _, err := client.Chat(t.Context(), provider.Request{Model: "m"}); err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("Authorization = %q, want bearer token", gotAuth)
	}
}
