// Package openai implements provider.Provider over any OpenAI-compatible
// HTTP API (OpenAI, LocalAI, vLLM, LM Studio, OpenRouter, ...).
package openai

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

	"github.com/openbooklet/openbooklet/internal/config"
	"github.com/openbooklet/openbooklet/internal/provider"
)

// Client is an OpenAI-compatible chat completions client.
type Client struct {
	endpoint   string
	apiKey     config.Secret
	http       *http.Client
	maxRetries int
}

// Config configures the client. Endpoint is the API base URL
// (for example "https://api.openai.com/v1"). MaxRetries caps retries on
// rate limiting and server errors; the zero value selects 3 retries,
// negative values disable retries.
type Config struct {
	Endpoint   string
	APIKey     config.Secret
	HTTPClient *http.Client
	MaxRetries int
}

// New creates a client from the given config. A zero Config talks to the
// OpenAI cloud endpoint without authentication.
func New(cfg Config) *Client {
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	maxRetries := cfg.MaxRetries
	switch {
	case maxRetries < 0:
		maxRetries = 0
	case maxRetries == 0:
		maxRetries = 3
	}
	return &Client{endpoint: endpoint, apiKey: cfg.APIKey, http: httpClient, maxRetries: maxRetries}
}

var _ provider.Provider = (*Client)(nil)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

type chatChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type chatResponse struct {
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type streamChoice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
}

type streamEvent struct {
	Model   string         `json:"model"`
	Choices []streamChoice `json:"choices"`
}

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Chat sends a single non-streaming completion request.
func (c *Client) Chat(ctx context.Context, req provider.Request) (provider.Response, error) {
	ctx, cancel := withTimeout(ctx, req.Timeout)
	defer cancel()

	body, err := json.Marshal(chatRequest{
		Model:       req.Model,
		Messages:    toChatMessages(req.Messages),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		return provider.Response{}, fmt.Errorf("encoding chat request: %w", err)
	}

	var out chatResponse
	if err := c.doJSON(ctx, http.MethodPost, c.endpoint+"/chat/completions", body, &out); err != nil {
		return provider.Response{}, err
	}
	content := ""
	if len(out.Choices) > 0 {
		content = out.Choices[0].Message.Content
	}
	return provider.Response{
		Content:     content,
		Model:       out.Model,
		UsageInput:  out.Usage.PromptTokens,
		UsageOutput: out.Usage.CompletionTokens,
		CreatedAt:   time.Now(),
	}, nil
}

// Stream opens a server-sent-events completion stream. The channel closes when
// the stream completes, the context is cancelled, or a transport error occurs.
func (c *Client) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ctx, cancel := withTimeout(ctx, req.Timeout)

	body, err := json.Marshal(chatRequest{
		Model:       req.Model,
		Messages:    toChatMessages(req.Messages),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("encoding stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("building stream request: %w", err)
	}
	c.setHeaders(httpReq)

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
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				return
			}
			var event streamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			for _, choice := range event.Choices {
				if choice.Delta.Content == "" {
					continue
				}
				select {
				case ch <- provider.Chunk{Token: choice.Delta.Content, Model: event.Model, CreatedAt: time.Now()}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

// Models lists the models advertised by the API.
func (c *Client) Models(ctx context.Context) ([]provider.Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var out modelsResponse
	if err := c.doJSON(ctx, http.MethodGet, c.endpoint+"/models", nil, &out); err != nil {
		return nil, err
	}
	models := make([]provider.Model, 0, len(out.Data))
	for _, m := range out.Data {
		models = append(models, provider.Model{ID: m.ID, Name: m.ID, Provider: "openai"})
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
		c.setHeaders(httpReq)

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

func (c *Client) setHeaders(r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		r.Header.Set("Authorization", "Bearer "+string(c.apiKey))
	}
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

func readAPIError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
	if err != nil {
		return fmt.Errorf("provider error: status %d", resp.StatusCode)
	}
	return parseAPIError(resp.StatusCode, body)
}

func parseAPIError(status int, body []byte) error {
	var apiErr apiError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return fmt.Errorf("provider error: status %d: %s", status, apiErr.Error.Message)
	}
	return fmt.Errorf("provider error: status %d", status)
}
