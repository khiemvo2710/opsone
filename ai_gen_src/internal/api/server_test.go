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

func TestProductScopeAutoPutOverview(t *testing.T) {
	srv := testServer(t)
	putBody := strings.NewReader(`{"auto_action":"auto"}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/scopes/GARENA/auto", putBody)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("X-OpsOne-Role", "admin")
	putRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put product auto: %d %s", putRec.Code, putRec.Body.String())
	}

	ovReq := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/overview", nil)
	ovReq.Header.Set("X-OpsOne-Role", "admin")
	ovRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(ovRec, ovReq)
	if ovRec.Code != http.StatusOK {
		t.Fatalf("overview: %d %s", ovRec.Code, ovRec.Body.String())
	}
	var ov struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(ovRec.Body.Bytes(), &ov); err != nil {
		t.Fatal(err)
	}
	var garena map[string]any
	for _, row := range ov.Rows {
		if row["product_code"] == "GARENA" && row["sku_code"] == "10000" {
			garena = row
			break
		}
	}
	if garena == nil {
		t.Fatal("GARENA 10000 row not found")
	}
	if garena["product_auto_action"] != "auto" {
		t.Fatalf("product_auto_action=%v want auto", garena["product_auto_action"])
	}
	if garena["auto_action"] != "auto" {
		t.Fatalf("effective auto_action=%v want auto", garena["auto_action"])
	}
}

func TestConfigPutMaintenanceDefaultDuration(t *testing.T) {
	srv := testServer(t)
	putBody := strings.NewReader(`{"maintenance_default_duration_min":90}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/config", putBody)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("X-OpsOne-Role", "admin")
	putRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put config duration: %d %s", putRec.Code, putRec.Body.String())
	}
	var cfg map[string]any
	if err := json.Unmarshal(putRec.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["maintenance_default_duration_min"] != float64(90) {
		t.Fatalf("maintenance_default_duration_min=%v want 90", cfg["maintenance_default_duration_min"])
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
