package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"code-executor-service/internal/api"
	"code-executor-service/internal/config"
	"code-executor-service/internal/executor"
)

func main() {
	// Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Starting code-executor service version=%s env=%s", cfg.App.Version, cfg.App.Env)

	// Initialize executor
	codeExecutor := executor.New(cfg)

	// Initialize handler
	handler := api.NewHandler(codeExecutor, cfg)

	// Setup routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server listening address=%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
