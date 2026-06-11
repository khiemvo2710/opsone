package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opsone/internal/api"
	"opsone/internal/config"
	"opsone/internal/healthserver"
	"opsone/internal/store"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	cfg.LogLLMStartup()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// AgentBase probes GET /health on :8080 while the container starts.
	healthserver.ListenAndServe(ctx, cfg.APIAddr)

	db, err := store.OpenWithRetry(ctx, cfg.MySQLDSN, 2*time.Minute)
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	defer db.Close()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = healthserver.Shutdown(shutdownCtx)
	cancel()

	srv := api.NewServer(db, cfg)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("api: %v", err)
	}
	log.Println("api stopped")
}
