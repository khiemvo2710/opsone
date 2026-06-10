package e2e_test

import (
	"context"
	"net/http"
	"testing"
)

// §12 — PUT /config ghi config_audit_log.
func TestE2E_DoD_ConfigAuditLog(t *testing.T) {
	db, cfg := requireE2E(t)
	ctx := context.Background()
	srv := testAPIServer(t, db, cfg)

	var before int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM config_audit_log`).Scan(&before)
	if err != nil {
		t.Fatal(err)
	}

	rec := apiPutJSON(t, srv, "/api/v1/config", `{"scheduler_interval_min":5}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT config: %d %s", rec.Code, rec.Body.String())
	}

	var after int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM config_audit_log`).Scan(&after)
	if err != nil {
		t.Fatal(err)
	}
	if after <= before {
		t.Errorf("config_audit_log count = %d, want > %d after PUT", after, before)
	}
}

// §12 — agent_change_log có bản ghi applied sau routing update (tools test covers write; verify list API).
func TestE2E_DoD_AgentChangesAPI(t *testing.T) {
	db, cfg := requireE2E(t)
	srv := testAPIServer(t, db, cfg)

	rec := apiGet(t, srv, "/api/v1/agent-changes?status=applied")
	if rec.Code != http.StatusOK {
		t.Fatalf("agent-changes: %d %s", rec.Code, rec.Body.String())
	}
}

// §12 — incidents list API (seed có ít nhất 0; endpoint phải 200).
func TestE2E_DoD_IncidentsAPI(t *testing.T) {
	db, cfg := requireE2E(t)
	srv := testAPIServer(t, db, cfg)

	rec := apiGet(t, srv, "/api/v1/incidents")
	if rec.Code != http.StatusOK {
		t.Fatalf("incidents: %d %s", rec.Code, rec.Body.String())
	}
}
