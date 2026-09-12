package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openbooklet/openbooklet/internal/booklet"
	"github.com/openbooklet/openbooklet/internal/config"
	"github.com/openbooklet/openbooklet/internal/provider"
)

type fakeProvider struct {
	tokens []string
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Chat(_ context.Context, _ provider.Request) (provider.Response, error) {
	return provider.Response{}, nil
}

func (f *fakeProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
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
	}()
	return ch, nil
}

func (f *fakeProvider) Models(_ context.Context) ([]provider.Model, error) { return nil, nil }

func testMux(t *testing.T, p provider.Provider) (*httptest.Server, *booklet.Service) {
	t.Helper()
	repo := booklet.NewInMemoryBookletRepository()
	svc := booklet.NewService(repo)
	cfg := config.NewDefaultConfig()
	server := httptest.NewServer(buildMux(svc, p, "fake-model", cfg, t.TempDir()))
	t.Cleanup(server.Close)
	return server, svc
}

func TestGenerateEndpointStreamsCells(t *testing.T) {
	server, svc := testMux(t, &fakeProvider{tokens: []string{"# Guide\n\nBody.\n\n## Step\n\nDo it.\n"}})
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}

	resp, err := http.Post(server.URL+"/api/v1/booklets/b1/generate", "application/json",
		strings.NewReader(`{"prompt":"Write a guide."}`))
	if err != nil {
		t.Fatalf("POST generate failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	var sawStart, sawToken bool
	completeSections := -1
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	var eventType string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		switch eventType {
		case "start":
			sawStart = true
		case "token":
			sawToken = true
		case "complete":
			var payload struct {
				Sections []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"sections"`
			}
			if err := json.Unmarshal([]byte(data), &payload); err != nil {
				t.Fatalf("decoding complete event: %v", err)
			}
			completeSections = len(payload.Sections)
		case "error":
			t.Fatalf("stream error: %s", data)
		}
	}
	if !sawStart || !sawToken {
		t.Error("stream missing start or token events")
	}
	if completeSections != 2 {
		t.Errorf("complete carries %d sections, want 2", completeSections)
	}

	stored, err := svc.GetBooklet("b1")
	if err != nil {
		t.Fatalf("GetBooklet failed: %v", err)
	}
	if len(stored.Sections) != 2 {
		t.Errorf("stored sections = %d, want 2", len(stored.Sections))
	}
}

func TestGenerateEndpointMissingBooklet(t *testing.T) {
	server, _ := testMux(t, &fakeProvider{})
	resp, err := http.Post(server.URL+"/api/v1/booklets/missing/generate", "application/json",
		strings.NewReader(`{"prompt":"hi"}`))
	if err != nil {
		t.Fatalf("POST generate failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGenerateEndpointNoProvider(t *testing.T) {
	server, svc := testMux(t, nil)
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}
	resp, err := http.Post(server.URL+"/api/v1/booklets/b1/generate", "application/json",
		strings.NewReader(`{"prompt":"hi"}`))
	if err != nil {
		t.Fatalf("POST generate failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestRegenerateEndpoint(t *testing.T) {
	server, svc := testMux(t, &fakeProvider{tokens: []string{"Expanded body.\n"}})
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}
	if err := svc.AddSection("b1", &booklet.Section{ID: "s1", Title: "Step", Level: 2, Content: "Do it.\n", Status: "edited"}); err != nil {
		t.Fatalf("AddSection failed: %v", err)
	}

	resp, err := http.Post(server.URL+"/api/v1/booklets/b1/sections/s1/regenerate", "application/json",
		strings.NewReader(`{"mode":"expand"}`))
	if err != nil {
		t.Fatalf("POST regenerate failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Data struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !strings.Contains(body.Data.Content, "Expanded body.") {
		t.Errorf("content = %q, want regenerated text", body.Data.Content)
	}

	stored, err := svc.GetSection("b1", "s1")
	if err != nil {
		t.Fatalf("GetSection failed: %v", err)
	}
	if len(stored.History) != 1 {
		t.Errorf("history = %d records, want 1", len(stored.History))
	}
}

func TestCreateSectionEndpoint(t *testing.T) {
	server, svc := testMux(t, &fakeProvider{})
	if _, err := svc.CreateBooklet("b1", "Guide", "sop", "", ""); err != nil {
		t.Fatalf("CreateBooklet failed: %v", err)
	}

	resp, err := http.Post(server.URL+"/api/v1/booklets/b1/sections", "application/json",
		strings.NewReader(`{"title":"Manual","level":2,"content":"Hand-written.\n"}`))
	if err != nil {
		t.Fatalf("POST section failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var body struct {
		Data struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Status  string `json:"status"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.Data.ID == "" || body.Data.Title != "Manual" || body.Data.Status != "draft" {
		t.Errorf("section = %+v", body.Data)
	}

	stored, err := svc.GetSection("b1", body.Data.ID)
	if err != nil {
		t.Fatalf("GetSection failed: %v", err)
	}
	if stored.Content != "Hand-written.\n" {
		t.Errorf("content = %q", stored.Content)
	}

	bad, err := http.Post(server.URL+"/api/v1/booklets/b1/sections", "application/json",
		strings.NewReader(`{"title":"Bad","level":9}`))
	if err != nil {
		t.Fatalf("POST section failed: %v", err)
	}
	defer bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", bad.StatusCode)
	}

	missing, err := http.Post(server.URL+"/api/v1/booklets/nope/sections", "application/json",
		strings.NewReader(`{"title":"X"}`))
	if err != nil {
		t.Fatalf("POST section failed: %v", err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", missing.StatusCode)
	}
}
