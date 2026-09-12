// Package ollama implements provider.Provider over the Ollama HTTP API.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openbooklet/openbooklet/internal/provider"
)

// Client is an Ollama chat client.
type Client struct {
	endpoint   string
	http       *http.Client
	maxRetries int
}

// Config configures the client. Endpoint is the Ollama base URL
// (for example "http://localhost:11434").
type Config struct {
	Endpoint   string
	HTTPClient *http.Client
	MaxRetries int
}

// New creates a client from the given config. A zero Config talks to a local
// Ollama daemon.
func New(cfg Config) *Client {
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 300 * time.Second}
	}
	maxRetries := cfg.MaxRetries
	switch {
	case maxRetries < 0:
		maxRetries = 0
	case maxRetries == 0:
		maxRetries = 3
	}
	return &Client{endpoint: endpoint, http: httpClient, maxRetries: maxRetries}
}

var _ provider.Provider = (*Client)(nil)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type chatResponse struct {
	Model           string      `json:"model"`
	Message         chatMessage `json:"message"`
	Done            bool        `json:"done"`
	PromptEvalCount int         `json:"prompt_eval_count"`
	EvalCount       int         `json:"eval_count"`
}

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// Chat sends a single non-streaming chat request.
func (c *Client) Chat(ctx context.Context, req provider.Request) (provider.Response, error) {
	ctx, cancel := withTimeout(ctx, req.Timeout)
	defer cancel()

	body, err := json.Marshal(chatRequest{
		Model:    req.Model,
		Messages: toChatMessages(req.Messages),
		Options:  options(req),
	})
	if err != nil {
		return provider.Response{}, fmt.Errorf("encoding chat request: %w", err)
	}

	start := time.Now()
	var out chatResponse
	if err := c.doJSON(ctx, http.MethodPost, c.endpoint+"/api/chat", body, &out); err != nil {
		return provider.Response{}, err
	}
	return provider.Response{
		Content:     out.Message.Content,
		Model:       out.Model,
		UsageInput:  out.PromptEvalCount,
		UsageOutput: out.EvalCount,
		Duration:    time.Since(start),
		CreatedAt:   time.Now(),
	}, nil
}

// Stream opens a newline-delimited JSON chat stream. The channel closes when
// the stream completes, the context is cancelled, or a transport error occurs.
func (c *Client) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ctx, cancel := withTimeout(ctx, req.Timeout)

	body, err := json.Marshal(chatRequest{
		Model:    req.Model,
		Messages: toChatMessages(req.Messages),
		Stream:   true,
		Options:  options(req),
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("encoding stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("building stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("opening stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		defer cancel()
		return nil, readAPIError(resp)
	}

	ch := make(chan provider.Chunk)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		defer cancel()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var event chatResponse
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}
			if event.Message.Content == "" {
				continue
			}
			select {
			case ch <- provider.Chunk{Token: event.Message.Content, Model: event.Model, CreatedAt: time.Now()}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// Models lists the models installed in the Ollama daemon.
func (c *Client) Models(ctx context.Context) ([]provider.Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var out tagsResponse
	if err := c.doJSON(ctx, http.MethodGet, c.endpoint+"/api/tags", nil, &out); err != nil {
		return nil, err
	}
	models := make([]provider.Model, 0, len(out.Models))
	for _, m := range out.Models {
		models = append(models, provider.Model{ID: m.Name, Name: m.Name, Provider: "ollama"})
	}
	return models, nil
}

func (c *Client) doJSON(ctx context.Context, method, url string, body []byte, out any) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("request cancelled: %w", ctx.Err())
			case <-time.After(backoff(attempt)):
			}
		}

		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		httpReq, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response: %w", err)
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = parseAPIError(resp.StatusCode, respBody)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return parseAPIError(resp.StatusCode, respBody)
		}
		if out != nil {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
		}
		return nil
	}
	return lastErr
}

func backoff(attempt int) time.Duration {
	d := 100 * time.Millisecond
	for i := 1; i < attempt; i++ {
		d *= 2
	}
	if d > 2*time.Second {
		d = 2 * time.Second
	}
	return d
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

func toChatMessages(messages []provider.Message) []chatMessage {
	out := make([]chatMessage, 0, len(messages))
	for _, m := range messages {
		out = append(out, chatMessage{Role: m.Role, Content: m.Content})
	}
	return out
}

func options(req provider.Request) map[string]any {
	opts := make(map[string]any)
	if req.Temperature != nil {
		opts["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		opts["num_predict"] = *req.MaxTokens
	}
	if len(opts) == 0 {
		return nil
	}
	return opts
}

func readAPIError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
	if err != nil {
		return fmt.Errorf("provider error: status %d", resp.StatusCode)
	}
	return parseAPIError(resp.StatusCode, body)
}

func parseAPIError(status int, body []byte) error {
	var ollamaErr struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &ollamaErr); err == nil && ollamaErr.Error != "" {
		return fmt.Errorf("provider error: status %d: %s", status, ollamaErr.Error)
	}
	return fmt.Errorf("provider error: status %d", status)
}
