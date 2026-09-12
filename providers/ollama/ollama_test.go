package ollama

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openbooklet/openbooklet/internal/provider"
)

const (
	testModel = "test-model"
	testText  = "Hello, world!"
)

var testTokens = []string{"Hello", ", ", "world", "!"}

// fakeServer speaks the Ollama HTTP protocol. The "never-responds" model
// sends headers plus one token then hangs; the "fail" model returns 500.
func fakeServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Model == provider.NeverRespondsModel {
			// Send headers plus one token, then hang: exercises
			// mid-stream cancellation and chat timeouts.
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, "{\"model\":%q,\"message\":{\"role\":\"assistant\",\"content\":\"Hello\"},\"done\":false}\n",
				testModel)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			<-r.Context().Done()
			return
		}
		if req.Model == "fail" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if !req.Stream {
			_, _ = fmt.Fprintf(w, `{"model":%q,"message":{"role":"assistant","content":%q},`+
				`"done":true,"prompt_eval_count":5,"eval_count":7}`, testModel, testText)
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flush", http.StatusInternalServerError)
			return
		}
		for _, tok := range testTokens {
			_, _ = fmt.Fprintf(w, "{\"model\":%q,\"message\":{\"role\":\"assistant\",\"content\":%q},\"done\":false}\n",
				testModel, tok)
			flusher.Flush()
		}
		_, _ = fmt.Fprintf(w, "{\"model\":%q,\"message\":{\"role\":\"assistant\",\"content\":\"\"},"+
			"\"done\":true,\"prompt_eval_count\":5,\"eval_count\":7}\n", testModel)
	})

	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"models":[{"name":%q}]}`, testModel)
	})

	return httptest.NewServer(mux)
}

func TestContract(t *testing.T) {
	server := fakeServer(t)
	defer server.Close()

	provider.VerifyContract(t, provider.ContractHarness{
		Provider: New(Config{Endpoint: server.URL, MaxRetries: -1}),
		ChatText: testText,
		ModelID:  testModel,
	})
}

func TestChatUsage(t *testing.T) {
	server := fakeServer(t)
	defer server.Close()

	client := New(Config{Endpoint: server.URL, MaxRetries: -1})
	resp, err := client.Chat(t.Context(), provider.Request{
		Model:    testModel,
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.UsageInput != 5 || resp.UsageOutput != 7 {
		t.Errorf("usage = %d/%d, want 5/7", resp.UsageInput, resp.UsageOutput)
	}
}

func TestChatServerError(t *testing.T) {
	server := fakeServer(t)
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
	server := fakeServer(t)
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
