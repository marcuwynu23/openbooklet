package main

import (
	"log"

	"github.com/openbooklet/openbooklet/internal/config"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	if err := runServer(cfg); err != nil {
		log.Fatalf("server: %v", err)
	}
}
