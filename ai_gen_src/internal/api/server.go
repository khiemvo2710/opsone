package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"opsone/internal/config"
	"opsone/internal/store"
	"opsone/internal/tools"
)

// Server is the OpsOne REST + SSE API (§2.3).
type Server struct {
	DB     *store.DB
	Tools  *tools.Registry
	Config config.Config
	mux    *http.ServeMux
}

// NewServer wires dependencies.
func NewServer(db *store.DB, cfg config.Config) *Server {
	s := &Server{
		DB:     db,
		Tools:  tools.NewRegistry(db),
		Config: cfg,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/health-status", s.handleHealthStatus)
	mux.HandleFunc("/api/v1/dashboard/overview", s.handleDashboardOverview)
	mux.HandleFunc("/api/v1/config", s.routeConfig)
	mux.HandleFunc("/api/v1/incidents", s.routeIncidents)
	mux.HandleFunc("/api/v1/incidents/", s.routeIncidents)
	mux.HandleFunc("/api/v1/routing-plans/", s.routeRoutingPlans)
	mux.HandleFunc("/api/v1/routing-plans/latest", s.handleRoutingPlansLatest)
	mux.HandleFunc("/api/v1/recommendations/", s.routeRecommendations)
	mux.HandleFunc("/api/v1/maintenance", s.routeMaintenance)
	mux.HandleFunc("/api/v1/maintenance/", s.routeMaintenance)
	mux.HandleFunc("/api/v1/notifications", s.handleNotificationsList)
	mux.HandleFunc("/api/v1/escalation-chat", s.handleEscalationList)
	mux.HandleFunc("/api/v1/products", s.handleProductsList)
	mux.HandleFunc("/api/v1/products/", s.routeProducts)
	mux.HandleFunc("/api/v1/scopes/", s.routeScopes)
	mux.HandleFunc("/api/v1/metrics", s.handleMetricsQuery)
	mux.HandleFunc("/api/v1/mock/status", s.handleMockStatus)
	mux.HandleFunc("/api/v1/mock/generate", s.handleMockGenerate)
	mux.HandleFunc("/api/v1/chat", s.routeChat)
	mux.HandleFunc("/api/v1/events", s.handleSSE)
	s.mux = mux
}

// Handler returns HTTP handler with CORS.
func (s *Server) Handler() http.Handler {
	return withCORS(s.Config.CORSOrigin, s.mux)
}

// Run starts listening on APIAddr until ctx cancelled.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.Config.APIAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	log.Printf("OpsOne API listening on %s (CORS %s)", s.Config.APIAddr, s.Config.CORSOrigin)
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) routeConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleConfigGet(w, r)
	case http.MethodPut:
		s.handleConfigPut(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
	}
}

func (s *Server) routeIncidents(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/incidents")
	path = strings.Trim(path, "/")
	if path == "" {
		if r.Method == http.MethodGet {
			s.handleIncidentsList(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
		return
	}
	if r.Method == http.MethodGet {
		s.handleIncidentGet(w, r, path)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
}

func (s *Server) routeRecommendations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/recommendations/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}
	id, ok := parseUintPath(parts[0])
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "ID không hợp lệ")
		return
	}
	switch parts[1] {
	case "approve":
		if r.Method == http.MethodPost {
			s.handleRecommendationApprove(w, r, id)
			return
		}
	case "reject":
		if r.Method == http.MethodPost {
			s.handleRecommendationReject(w, r, id)
			return
		}
	}
	writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
}

func (s *Server) routeRoutingPlans(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/routing-plans/")
	path = strings.Trim(path, "/")
	if path == "latest" {
		if r.Method == http.MethodGet {
			s.handleRoutingPlansLatest(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
		}
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}
	id, ok := parseUintPath(parts[0])
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "ID không hợp lệ")
		return
	}
	switch parts[1] {
	case "approve":
		if r.Method == http.MethodPost {
			s.handleRoutingPlanApprove(w, r, id)
			return
		}
	case "reject":
		if r.Method == http.MethodPost {
			s.handleRoutingPlanReject(w, r, id)
			return
		}
	}
	writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
}

func (s *Server) routeMaintenance(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/maintenance")
	path = strings.Trim(path, "/")
	if path == "" {
		if r.Method == http.MethodGet {
			s.handleMaintenanceList(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
		return
	}
	if r.Method == http.MethodGet {
		s.handleMaintenanceGet(w, r, path)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
}

func (s *Server) routeProducts(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/products/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}
	code, sub := parts[0], parts[1]
	switch sub {
	case "routing":
		if r.Method == http.MethodGet {
			s.handleProductRoutingGet(w, r, code)
			return
		}
	case "thresholds":
		if r.Method == http.MethodGet {
			s.handleProductThresholdsGet(w, r, code)
			return
		}
		if r.Method == http.MethodPut {
			s.handleProductThresholdsPut(w, r, code)
			return
		}
	}
	writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
}

func (s *Server) routeScopes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/scopes/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}
	product := parts[0]
	action := parts[len(parts)-1]

	if action == "auto" && r.Method == http.MethodPut {
		sku := ""
		if len(parts) == 3 {
			sku = parts[1]
		}
		s.handleScopeAutoPut(w, r, product, sku)
		return
	}

	if r.Method != http.MethodPost || len(parts) != 4 {
		writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
		return
	}
	sku := parts[1]
	switch parts[2] {
	case "routing":
		switch action {
		case "approve":
			s.handleScopeRoutingApprove(w, r, product, sku)
		case "apply":
			s.handleScopeRoutingApply(w, r, product, sku)
		case "restore-baseline":
			s.handleScopeRoutingRestoreBaseline(w, r, product, sku)
		case "reject":
			s.handleScopeRoutingReject(w, r, product, sku)
		default:
			writeError(w, http.StatusNotFound, "not_found", "Not found")
		}
	case "maintenance":
		switch action {
		case "approve":
			s.handleScopeMaintenanceApprove(w, r, product, sku)
		case "reject":
			s.handleScopeMaintenanceReject(w, r, product, sku)
		case "cancel":
			s.handleScopeMaintenanceCancel(w, r, product, sku)
		case "reopen-service":
			s.handleScopeMaintenanceReopenService(w, r, product, sku)
		case "extend":
			s.handleScopeMaintenanceExtend(w, r, product, sku)
		default:
			writeError(w, http.StatusNotFound, "not_found", "Not found")
		}
	default:
		writeError(w, http.StatusNotFound, "not_found", "Not found")
	}
}

func (s *Server) routeChat(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleChatPost(w, r)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
}
