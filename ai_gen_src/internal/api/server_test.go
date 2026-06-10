package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"opsone/internal/api"
	"opsone/internal/config"
	"opsone/internal/store"
)

func testServer(t *testing.T) *api.Server {
	t.Helper()
	if os.Getenv("OPSONE_INTEGRATION") != "1" {
		t.Skip("set OPSONE_INTEGRATION=1 for API integration tests")
	}
	cfg := config.Load()
	cfg.DevAuthBypass = true
	db, err := store.Open(cfg.MySQLDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return api.NewServer(db, cfg)
}

func TestHealthStatus(t *testing.T) {
	srv := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health-status", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["health_status"]; !ok {
		t.Error("missing health_status")
	}
}

func TestHealthLiveness(t *testing.T) {
	srv := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestIncidentGetByID(t *testing.T) {
	srv := testServer(t)
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	listRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list incidents: %d %s", listRec.Code, listRec.Body.String())
	}
	var list struct {
		Items []struct {
			IncidentID string `json:"incident_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) == 0 {
		t.Skip("no incidents in DB")
	}
	id := list.Items[0].IncidentID
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+id, nil)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get incident %s: %d %s", id, getRec.Code, getRec.Body.String())
	}
}

func TestConfigPutAudit(t *testing.T) {
	srv := testServer(t)
	body := strings.NewReader(`{"scheduler_interval_min":5}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config", body)
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put config: %d %s", rec.Code, rec.Body.String())
	}
}
