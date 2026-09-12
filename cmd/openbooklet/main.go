package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/openbooklet/openbooklet/internal/booklet"
	"github.com/openbooklet/openbooklet/internal/config"
	"github.com/openbooklet/openbooklet/internal/section"
)

func main() {
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
	log.Println("Phase 1 — Core Go Engine initialized. In-memory repository active.")

	// Verify services are wired correctly.
	bs := app.BookletService()
	ss := app.SectionService()
	if bs == nil || ss == nil {
		log.Fatal("Failed to initialize services")
	}
}

// App is the top-level application composition root for Phase 1.
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
