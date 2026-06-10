package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"opsone/internal/api"
	"opsone/internal/config"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := api.NewServer(db, cfg)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("api: %v", err)
	}
	log.Println("api stopped")
}
