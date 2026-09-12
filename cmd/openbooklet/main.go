package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/openbooklet/openbooklet/internal/booklet"
	"github.com/openbooklet/openbooklet/internal/config"
	"github.com/openbooklet/openbooklet/internal/provider"
	"github.com/openbooklet/openbooklet/internal/section"
	"github.com/openbooklet/openbooklet/internal/storage"
	"github.com/openbooklet/openbooklet/providers/ollama"
	"github.com/openbooklet/openbooklet/providers/openai"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "generate" {
		if err := runGenerate(os.Args[2:]); err != nil {
			log.Fatalf("generate: %v", err)
		}
		return
	}

	cfg := config.LoadFromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	repo := booklet.NewInMemoryBookletRepository()

	app := NewApp(cfg, repo)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down OpenBooklet...")
		os.Exit(0)
	}()

	log.Printf("OpenBooklet starting on %s:%d", cfg.Host, cfg.Port)
	log.Println("Usage: openbooklet generate --prompt \"Describe the SOP to write.\"")

	// Verify services are wired correctly.
	bs := app.BookletService()
	ss := app.SectionService()
	if bs == nil || ss == nil {
		log.Fatal("Failed to initialize services")
	}
}

// App is the top-level application composition root.
type App struct {
	cfg  *config.Config
	repo *booklet.InMemoryBookletRepository
}

// NewApp creates a new application instance.
func NewApp(cfg *config.Config, repo *booklet.InMemoryBookletRepository) *App {
	return &App{
		cfg:  cfg,
		repo: repo,
	}
}

// BookletService returns a new booklet service using the application's repository.
func (a *App) BookletService() *booklet.Service {
	return booklet.NewService(a.repo)
}

// SectionService returns a new section service.
func (a *App) SectionService() *section.Service {
	return section.NewService(a.repo)
}

// runGenerate implements `openbooklet generate`: prompt → AI Markdown →
// parsed section cells → persisted booklet → .obk file on disk.
func runGenerate(args []string) error {
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	prompt := flags.String("prompt", "", "Chat-style description of the document to create (required).")
	title := flags.String("title", "Untitled Booklet", "Booklet title.")
	docType := flags.String("type", "general", "Booklet type (sop, article, runbook, ...).")
	out := flags.String("out", "", "Write the .obk file here (default \"<booklet-id>.obk\").")
	memory := flags.Bool("memory", false, "Use an in-memory database instead of the configured file.")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*prompt) == "" {
		flags.Usage()
		return fmt.Errorf("prompt is required")
	}

	cfg := config.LoadFromEnv()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	p, model, err := openProvider(cfg)
	if err != nil {
		return err
	}

	dbPath := cfg.DatabasePath
	if *memory {
		dbPath = ":memory:"
	} else if dir := filepath.Dir(dbPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating data directory: %w", err)
		}
	}
	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		return fmt.Errorf("opening storage: %w", err)
	}
	defer store.Close()

	svc := booklet.NewService(storage.NewSQLiteBookletRepository(store.DB()))

	id := uuid.NewString()
	if _, err := svc.CreateBooklet(id, *title, *docType, cfg.ProviderName, *prompt); err != nil {
		return fmt.Errorf("creating booklet: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sections, err := svc.GenerateSections(ctx, id, p, model, *prompt)
	if err != nil {
		return fmt.Errorf("generating sections: %w", err)
	}

	b, err := svc.GetBooklet(id)
	if err != nil {
		return fmt.Errorf("loading booklet: %w", err)
	}
	data, err := booklet.MarshalBooklet(b)
	if err != nil {
		return fmt.Errorf("serializing booklet: %w", err)
	}

	outPath := *out
	if outPath == "" {
		outPath = id + ".obk"
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing .obk file: %w", err)
	}

	fmt.Printf("Booklet %q (%s) with %d sections written to %s\n", b.Title, id, len(sections), outPath)
	for _, sec := range sections {
		fmt.Printf("  %s %s [%s]\n", strings.Repeat("#", sec.Level), sec.Title, sec.Status)
	}
	return nil
}

// openProvider builds the configured LLM provider and resolves the model.
func openProvider(cfg *config.Config) (provider.Provider, string, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.ProviderName)) {
	case "", "ollama":
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
