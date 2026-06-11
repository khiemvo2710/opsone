package healthserver

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	activeMu sync.Mutex
	active   *http.Server
)

// ListenAndServe starts GET /health on addr (default :8080) for GreenNode AgentBase runtime checks.
func ListenAndServe(ctx context.Context, addr string) {
	if addr == "" {
		addr = os.Getenv("WORKER_HEALTH_ADDR")
	}
	if addr == "" {
		addr = ":8080"
	}

	ok := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", ok)
	mux.HandleFunc("/api/v1/health", ok)

	srv := &http.Server{Addr: addr, Handler: mux}
	activeMu.Lock()
	active = srv
	activeMu.Unlock()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = Shutdown(shutdownCtx)
	}()
	go func() {
		log.Printf("health server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("health server: %v", err)
		}
	}()
}

// Shutdown stops the active health server so another listener can bind the same port.
func Shutdown(ctx context.Context) error {
	activeMu.Lock()
	srv := active
	active = nil
	activeMu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}
