package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opsone/internal/agent"
	"opsone/internal/config"
	"opsone/internal/healthserver"
	"opsone/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	healthserver.ListenAndServe(ctx, "")

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	db, err := store.OpenWithRetry(ctx, cfg.MySQLDSN, 2*time.Minute)
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	defer db.Close()

	settings, err := db.GetAgentSettings(context.Background())
	if err != nil {
		log.Fatalf("settings: %v", err)
	}

	interval := agent.IntervalFromSettings(settings)
	log.Printf("OpsOne worker-agent (core) starting interval=%s data_source=%s", interval, settings.DataSource)

	runner := agent.NewRunner(db)
	if err := runner.RunBlocking(ctx, interval); err != nil && err != context.Canceled {
		log.Fatalf("agent worker: %v", err)
	}
	log.Println("worker-agent stopped")
}
