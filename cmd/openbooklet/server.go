package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/openbooklet/openbooklet/internal/booklet"
	"github.com/openbooklet/openbooklet/internal/config"
	"github.com/openbooklet/openbooklet/internal/provider"
	"github.com/openbooklet/openbooklet/internal/storage"
	"github.com/openbooklet/openbooklet/providers/ollama"
	"github.com/openbooklet/openbooklet/providers/openai"
)

// runServer boots the HTTP API and blocks until SIGINT/SIGTERM.
func runServer(cfg *config.Config) error {
	if dir := filepath.Dir(cfg.DatabasePath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating data directory: %w", err)
		}
	}
	store, err := storage.NewSQLiteStorage(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("opening storage: %w", err)
	}
	defer store.Close()

	p, model, err := openProvider(cfg)
	if err != nil {
		return err
	}
	mux := buildMux(booklet.NewService(storage.NewSQLiteBookletRepository(store.DB())), p, model, cfg, "frontend/dist")

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("OpenBooklet listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-quit:
		log.Printf("Received %s, shutting down...", sig)
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	log.Println("OpenBooklet stopped.")
	return nil
}

// apiServer wires services to HTTP handlers. Provider and model are resolved
// once at startup; handlers share them across requests.
type apiServer struct {
	svc      *booklet.Service
	provider provider.Provider
	model    string
}

// buildMux assembles every route. Tests use it with fakes; runServer uses it
// with real dependencies.
func buildMux(svc *booklet.Service, p provider.Provider, model string, cfg *config.Config, dist string) *http.ServeMux {
	s := &apiServer{svc: svc, provider: p, model: model}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/api/v1/version", handleVersion(cfg))
	mux.HandleFunc("GET /api/v1/booklets", s.handleBooklets)
	mux.HandleFunc("POST /api/v1/booklets", s.handleCreateBooklet)
	mux.HandleFunc("GET /api/v1/booklets/{id}", s.handleBooklet)
	mux.HandleFunc("PUT /api/v1/booklets/{id}/sections/{sectionID}", s.handleUpdateSection)
	mux.HandleFunc("POST /api/v1/booklets/{id}/sections", s.handleCreateSection)
	mux.HandleFunc("POST /api/v1/booklets/{id}/generate", s.handleGenerate)
	mux.HandleFunc("POST /api/v1/booklets/{id}/sections/{sectionID}/regenerate", s.handleRegenerate)
	mux.HandleFunc("/", handleStatic(dist))
	return mux
}

// openProvider builds the configured LLM provider and resolves the model.
// An empty provider name means no AI features; generation endpoints then
// report no_provider while the editor keeps working offline.
func openProvider(cfg *config.Config) (provider.Provider, string, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.ProviderName)) {
	case "":
		return nil, "", nil
	case "ollama":
		model := cfg.ProviderModel
		if model == "" {
			model = "llama3"
		}
		return ollama.New(ollama.Config{Endpoint: cfg.ProviderEndpoint}), model, nil
	case "openai", "compatible":
		model := cfg.ProviderModel
		if model == "" {
			model = "gpt-4o-mini"
		}
		return openai.New(openai.Config{
			Endpoint: cfg.ProviderEndpoint,
			APIKey:   cfg.ProviderAPIKey,
		}), model, nil
	default:
		return nil, "", fmt.Errorf("unknown provider %q: want ollama, openai, or compatible", cfg.ProviderName)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleVersion(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]string{
				"version":  "0.1.0",
				"provider": cfg.ProviderName,
			},
		})
	}
}

type bookletSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Sections  int    `json:"sections"`
	UpdatedAt string `json:"updatedAt"`
}

func (s *apiServer) handleBooklets(w http.ResponseWriter, _ *http.Request) {
	all, err := s.svc.GetAllBooklets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", "could not list booklets")
		return
	}
	summaries := make([]bookletSummary, 0, len(all))
	for _, b := range all {
		summaries = append(summaries, bookletSummary{
			ID:        b.ID,
			Title:     b.Title,
			Type:      b.Type,
			Status:    string(b.Status),
			Sections:  len(b.Sections),
			UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": summaries})
}

type sectionDTO struct {
	ID        string  `json:"id"`
	ParentID  *string `json:"parentId"`
	Title     string  `json:"title"`
	Level     int     `json:"level"`
	Prompt    string  `json:"prompt"`
	Content   string  `json:"content"`
	Status    string  `json:"status"`
	UpdatedAt string  `json:"updatedAt"`
}

type bookletDTO struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Type         string       `json:"type"`
	Status       string       `json:"status"`
	Audience     string       `json:"audience"`
	Instructions string       `json:"instructions"`
	Sections     []sectionDTO `json:"sections"`
	UpdatedAt    string       `json:"updatedAt"`
}

func toSectionDTO(s *booklet.Section) sectionDTO {
	return sectionDTO{
		ID:        s.ID,
		ParentID:  s.ParentID,
		Title:     s.Title,
		Level:     s.Level,
		Prompt:    s.Prompt,
		Content:   s.Content,
		Status:    string(s.Status),
		UpdatedAt: s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toBookletDTO(b *booklet.Booklet) bookletDTO {
	sections := make([]sectionDTO, 0, len(b.Sections))
	for i := range b.Sections {
		sections = append(sections, toSectionDTO(&b.Sections[i]))
	}
	return bookletDTO{
		ID:           b.ID,
		Title:        b.Title,
		Type:         b.Type,
		Status:       string(b.Status),
		Audience:     b.Audience,
		Instructions: b.Instructions,
		Sections:     sections,
		UpdatedAt:    b.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *apiServer) handleBooklet(w http.ResponseWriter, r *http.Request) {
	b, err := s.svc.GetBooklet(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "booklet not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toBookletDTO(b)})
}

func (s *apiServer) handleCreateBooklet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
		Type  string `json:"type"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "title is required")
		return
	}
	docType := req.Type
	if docType == "" {
		docType = "general"
	}
	b, err := s.svc.CreateBooklet(uuid.NewString(), req.Title, docType, "", "")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": toBookletDTO(b)})
}

func (s *apiServer) handleCreateSection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title    string  `json:"title"`
		Level    int     `json:"level"`
		ParentID *string `json:"parentId"`
		Prompt   string  `json:"prompt"`
		Content  string  `json:"content"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "title is required")
		return
	}
	level := req.Level
	if level == 0 {
		level = 2
	}
	now := time.Now().UTC()
	sec := &booklet.Section{
		ID:        uuid.NewString(),
		ParentID:  req.ParentID,
		Title:     req.Title,
		Level:     level,
		Prompt:    req.Prompt,
		Content:   req.Content,
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.svc.AddSection(r.PathValue("id"), sec); err != nil {
		if _, lookupErr := s.svc.GetBooklet(r.PathValue("id")); lookupErr != nil {
			writeError(w, http.StatusNotFound, "not_found", "booklet not found")
		} else {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": toSectionDTO(sec)})
}

func (s *apiServer) handleUpdateSection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   *string `json:"title"`
		Prompt  *string `json:"prompt"`
		Content *string `json:"content"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	sec, err := s.svc.UpdateSection(r.PathValue("id"), r.PathValue("sectionID"), req.Title, req.Prompt, req.Content)
	if err != nil {
		if _, lookupErr := s.svc.GetSection(r.PathValue("id"), r.PathValue("sectionID")); lookupErr != nil {
			writeError(w, http.StatusNotFound, "not_found", "booklet or section not found")
		} else {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toSectionDTO(sec)})
}

// handleGenerate streams a booklet generation as server-sent events:
// start → token* → complete | error. On complete the cells are already
// saved; the event carries their summaries.
func (s *apiServer) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt string `json:"prompt"`
		Model  string `json:"model"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "prompt is required")
		return
	}
	if s.provider == nil {
		writeError(w, http.StatusServiceUnavailable, "no_provider", "no LLM provider is configured")
		return
	}
	model := req.Model
	if model == "" {
		model = s.model
	}

	events, err := s.svc.StreamGeneration(r.Context(), r.PathValue("id"), s.provider, model, req.Prompt)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "booklet not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "no_stream", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	writeEvent := func(eventType string, payload any) {
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
		flusher.Flush()
	}
	for event := range events {
		switch event.Type {
		case booklet.SectionEventStart:
			writeEvent("start", map[string]string{"generationId": event.GenerationID})
		case booklet.SectionEventToken:
			writeEvent("token", map[string]string{"token": event.Token})
		case booklet.SectionEventComplete:
			summaries := make([]sectionDTO, 0, len(event.Sections))
			for i := range event.Sections {
				summaries = append(summaries, toSectionDTO(&event.Sections[i]))
			}
			writeEvent("complete", map[string]any{"sections": summaries})
		case booklet.SectionEventError:
			writeEvent("error", map[string]string{"message": event.Message})
		}
	}
}

// handleRegenerate rewrites one cell with AI. Mode is one of regenerate,
// expand, shorten, or edit (edit requires an instruction).
func (s *apiServer) handleRegenerate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode        string `json:"mode"`
		Instruction string `json:"instruction"`
		Prompt      string `json:"prompt"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if s.provider == nil {
		writeError(w, http.StatusServiceUnavailable, "no_provider", "no LLM provider is configured")
		return
	}
	mode := booklet.RegenerateMode(req.Mode)
	if mode == "" {
		mode = booklet.RegenerateRewrite
	}
	if strings.TrimSpace(req.Prompt) != "" {
		if _, err := s.svc.UpdateSection(r.PathValue("id"), r.PathValue("sectionID"), nil, &req.Prompt, nil); err != nil {
			writeError(w, http.StatusNotFound, "not_found", "booklet or section not found")
			return
		}
	}
	sec, err := s.svc.RegenerateSection(r.Context(), r.PathValue("id"), r.PathValue("sectionID"), s.provider, s.model, mode, req.Instruction)
	if err != nil {
		if _, lookupErr := s.svc.GetSection(r.PathValue("id"), r.PathValue("sectionID")); lookupErr != nil {
			writeError(w, http.StatusNotFound, "not_found", "booklet or section not found")
		} else {
			writeError(w, http.StatusBadGateway, "generation_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toSectionDTO(sec)})
}

func decodeBody(r *http.Request, out any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024))
	if err != nil {
		return fmt.Errorf("reading request body: %w", err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// handleStatic serves the built frontend with an SPA fallback to index.html.
func handleStatic(dist string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "not_found", "unknown API route")
			return
		}
		if _, err := os.Stat(filepath.Join(dist, "index.html")); err != nil {
			writeError(w, http.StatusNotFound, "frontend_missing", "frontend not built: run npm run build in frontend/")
			return
		}
		cleaned := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		target := filepath.Join(dist, filepath.FromSlash(cleaned))
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			http.ServeFile(w, r, target)
			return
		}
		http.ServeFile(w, r, filepath.Join(dist, "index.html"))
	}
}
