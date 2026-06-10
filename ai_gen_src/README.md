# OpsOne — Phase 0–7

Monorepo Go + MySQL cho **OpsOne** — spec: `../OpsOne.md`

## Phase deliverables

### Phase 0 — Setup ✅
- Monorepo, `db/schema.sql`, `db/seed.sql`, `internal/store`

### Phase 1 — Mock + Scheduler ✅
- `cmd/worker-mock` — sinh `mock_metrics` mỗi 1 phút
- `cmd/worker-agent` — dry-run scheduler 5 phút
- `internal/mock`, `internal/agent`, `internal/catalog`

### Phase 2 — Tools ✅
- **9 tools** trong `internal/tools/`
- `internal/threshold`, `internal/notify`, `internal/rollback`

### Phase 4 — Reasoning + Output ✅
- `internal/rules` — engine rules 1–9 (§7.3), deterministic trước LLM
- `internal/output` — template tiếng Việt: Incident, Routing Plan, Maintenance, Health
- `internal/agent/reasoning.go` — ghi `incidents`, `routing_plans`, `recommendations`
- LLM: fallback template khi `LLM_API_URL` trống (§7.6)

### Phase 6 — Frontend React ✅
- `web/` — Vite + React 18 + TypeScript + React Router + TanStack Query
- Routes §9: `/`, `/chat`, `/settings`, `/changes`, `/maintenance`, `/incidents/:id`
- Components: `HealthBadge`, `IncidentCard`, `RoutingPlanTable`, `MaintenanceCard`
- Chat + `useVoiceInput` (Web Speech API `vi-VN`); SSE `/events` + poll 30s fallback
- Dev: `VITE_DEV_AUTH_BYPASS=true` (header `X-OpsOne-Role`); production: MSAL.js §2.6.4

### Phase 5 — REST API + SSE ✅
- `cmd/api` — HTTP server `:8080` (§2.3)
- `internal/api` — handlers, CORS, `DEV_AUTH_BYPASS`, SSE poll 5s
- Endpoints: health-status, config, incidents, routing-plans, agent-changes, maintenance, notifications, products, metrics, mock, chat (stub), `/events`
- Integration test: `$env:OPSONE_INTEGRATION=1; go test ./internal/api/... -v`

### Phase 3 — Agent Core ✅
- `internal/agent/runner.go` — pipeline §3: loop theo `routing_mode`, gọi tools, EvaluateThresholds
- `CycleContext` / `ProductContext` / `ScopeContext` — gom context cho Reasoning (Phase 4)
- Ghi `agent_analysis_history`, `health_status_product`, `agent_state_history`
- `cmd/worker-agent` dùng `Runner` thay dry-run

### Phase 7 — E2E & Definition of Done ✅
- `internal/e2e/` — kịch bản §10/§12: routing seed (A), health API (H), rollback API (I), approve plan, audit log, dashboard
- `make test-e2e` / `Invoke-OpsOneE2E` — integration với MySQL; `-Full` chạy 3 agent cycles (`OPSONE_E2E_FULL=1`)
- `scripts/e2e.ps1` — seed UTF-8 + E2E smoke + product count
- Makefile: `test-unit`, `test-integration`, `test-e2e`, `verify-dod`; seed qua `docker cp` (UTF-8)

**Vận hành (demo):**
| Vai trò | Hành động |
|---------|-----------|
| **Ops admin** | Duyệt routing plan trên Dashboard hoặc `POST /routing-plans/{id}/approve` (header `X-OpsOne-Role: admin`) |
| **Ops admin** | Rollback trên `/changes` hoặc `POST /agent-changes/{id}/rollback` |
| **Agent auto** | Per SKU: `routing_scope_state.auto_action` = `auto` hoặc `time_window` (trong giờ) → agent apply; `recommend_only` (mặc định) → plan chờ duyệt |
| **On-call** | Worker crash → restart `worker-agent` / `worker-mock`; xem `agent_analysis_cycles.status=failed` |

## Quick start

```powershell
cd "C:\D\Projects\documents\AI Hackathon\ai_gen_src"

# Helpers (optional)
. .\scripts\dev.ps1

# Copy env
copy .env.example .env

# MySQL + migrate + seed
Invoke-OpsOneReset

# Tests (unit; integration cần MySQL + OPSONE_INTEGRATION=1)
go test ./...
$env:OPSONE_INTEGRATION="1"; go test ./... -v

# Phase 7 E2E (MySQL + seed)
.\scripts\e2e.ps1
# Hoặc: Invoke-OpsOneE2E
# 3 agent cycles (~10 phút): .\scripts\e2e.ps1 -Full
```

### Chạy workers (Phase 1)

Terminal 1 — mock data (1 phút):
```powershell
go run ./cmd/worker-mock
```

Terminal 2 — agent scheduler (5 phút):
```powershell
go run ./cmd/worker-agent
```

Terminal 3 — REST API + SSE (Phase 5):
```powershell
go run ./cmd/api
# http://localhost:8080/api/v1/health-status
```

Ví dụ curl (dev auth bypass mặc định `true`):
```powershell
curl http://localhost:8080/api/v1/health-status
curl http://localhost:8080/api/v1/config
curl -X PUT http://localhost:8080/api/v1/config -H "Content-Type: application/json" -H "X-OpsOne-Role: admin" -d "{\"scheduler_interval_min\":5}"
curl http://localhost:8080/api/v1/incidents
curl http://localhost:8080/api/v1/routing-plans/latest
curl -N http://localhost:8080/api/v1/events
```

### Chạy React UI (Phase 6)

Cần **Node.js 20 LTS** (`winget install OpenJS.NodeJS.LTS`).

```powershell
cd web
copy .env.example .env
npm install
npm run dev
# http://localhost:5173 — proxy /api → :8080
```

Chạy đủ stack (4 terminal):
1. `docker compose up -d`
2. `go run ./cmd/worker-mock`
3. `go run ./cmd/worker-agent`
4. `go run ./cmd/api`
5. `cd web && npm run dev`

Verify trong MySQL:
```sql
SELECT COUNT(*) FROM mock_metrics;
SELECT * FROM agent_analysis_cycles ORDER BY id DESC LIMIT 3;
SELECT * FROM agent_analysis_history WHERE cycle_id = (SELECT MAX(id) FROM agent_analysis_cycles) LIMIT 5;
```

> **Go:** Cần **1.21.13** (`winget install GoLang.Go --version 1.21.13`). Khớp spec OpsOne §2.1.

### Múi giờ (vi-VN, UTC+7)

DBeaver hiển thị `10:19` trong khi đồng hồ Windows `17:19` → MySQL/Go đang lưu **UTC** thay vì giờ VN.

Đã cấu hình:
- `docker-compose.yml`: `TZ=Asia/Ho_Chi_Minh`, `--default-time-zone=+07:00`
- DSN MySQL: `loc=Asia/Ho_Chi_Minh` (tự thêm nếu `.env` thiếu)

Áp dụng lại:
```powershell
docker compose down
docker compose up -d
copy .env.example .env   # hoặc thêm loc= vào MYSQL_DSN hiện tại
# restart worker-mock / worker-agent
```

Verify trong MySQL:
```sql
SELECT @@global.time_zone, @@session.time_zone, NOW();
-- kỳ vọng: +07:00 và NOW() ≈ giờ Windows
```

> Dữ liệu cũ (trước khi sửa) vẫn lệch 7h; chỉ bản ghi mới sau restart mới đúng.

## Cấu trúc

```text
ai_gen_src/
├── cmd/api/              # Phase 5 — REST + SSE
├── cmd/worker-mock/      # Phase 1 — mock 1 phút
├── cmd/worker-agent/     # Phase 1/3 — scheduler 5 phút
├── internal/
│   ├── config/           # env §2.1 (+ SMTP §8.9)
│   ├── domain/           # enums
│   ├── agent/            # Phase 3–4 — Core + Reasoning
│   ├── rules/            # Phase 4 — Rules §7.3
│   ├── output/           # Phase 4 — Output templates §8
│   ├── tools/            # Phase 2 — 9 tools §6
│   ├── threshold/        # Phase 2 — EvaluateThresholds §7.4
│   ├── notify/           # Phase 2 — email §8.9
│   ├── rollback/         # Phase 2 — rollback §8.7
│   ├── api/              # Phase 5 — REST + SSE §2.3
│   ├── e2e/              # Phase 7 — E2E §10/§12
│   └── store/            # MySQL §13
├── db/schema.sql
├── db/seed.sql
└── web/                  # Phase 6 — React (Vite)
    └── src/
        ├── api/          # REST client
        ├── auth/         # MSAL + dev bypass
        ├── components/   # HealthBadge, cards, Layout
        ├── hooks/        # useSSE, useVoiceInput
        └── pages/        # Dashboard, Chat, Settings, ...
```

## Verify Phase 0–7

| Check | Command |
|-------|---------|
| 11 products | `Invoke-OpsOneReset` → product_count = 11 |
| mock_metrics tăng | `go run ./cmd/worker-mock` |
| dry-run 1 cycle | `$env:OPSONE_INTEGRATION=1; go test ./internal/mock/... -run DryRun` |
| 9 tools + rollback | `$env:OPSONE_INTEGRATION=1; go test ./internal/tools/... -v` |
| threshold + notify | `$env:OPSONE_INTEGRATION=1; go test ./internal/threshold/... ./internal/tools/... -run Notify` |
| agent core cycle | `$env:OPSONE_INTEGRATION=1; go test ./internal/agent/... -v` |
| rules unit test | `go test ./internal/rules/... -v` |
| incidents sau cycle | `SELECT * FROM incidents ORDER BY id DESC LIMIT 5` |
| API health + config | `$env:OPSONE_INTEGRATION=1; go test ./internal/api/... -v` |
| **E2E §10/§12** | `.\scripts\e2e.ps1` hoặc `make test-e2e` |
| E2E 2 cycles (G) | `.\scripts\e2e.ps1 -Full` |
| API live | `go run ./cmd/api` → `curl http://localhost:8080/api/v1/health-status` |
| React dev | `cd web && npm run dev` → Dashboard + HealthBadge |
| Rollback UI | `/changes` → nút Rollback trên `agent_change_log` |

**Chưa trong scope Phase 7:** MSAL production, PUT routing/thresholds qua UI, email J/K live SMTP, toàn bộ §10 A–L (maintenance B, chat E/F, baseline L).
