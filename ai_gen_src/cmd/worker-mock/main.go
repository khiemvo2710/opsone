package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"opsone/internal/agent"
	"opsone/internal/config"
	"opsone/internal/mock"
	"opsone/internal/store"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	db, err := store.Open(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	defer db.Close()

	settings, err := db.GetAgentSettings(context.Background())
	if err != nil {
		log.Fatalf("settings: %v", err)
	}

	interval := agent.MockIntervalFromSettings(settings)
	log.Printf("OpsOne worker-mock starting interval=%s scenario=%s", interval, settings.MockScenario)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	gen := mock.NewGenerator(db, 0)
	if err := gen.RunBlocking(ctx, interval); err != nil && err != context.Canceled {
		log.Fatalf("mock worker: %v", err)
	}
	log.Println("worker-mock stopped")
}
