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
- `internal/threshold`, `internal/notify`

### Phase 3 — Agent Core ✅
- `internal/agent/runner.go` — pipeline §3: loop theo `routing_mode`, gọi tools, EvaluateThresholds
- `CycleContext` / `ProductContext` / `ScopeContext` — gom context cho Reasoning (Phase 4)
- Ghi `agent_analysis_history`, `health_status_product`, `agent_state_history`
- `cmd/worker-agent` dùng `Runner` thay dry-run

### Phase 4 — Reasoning + Output ✅
- `internal/rules` — engine rules 1–9 (§7.3), deterministic trước LLM
- `internal/output` — template tiếng Việt: Incident, Routing Plan, Maintenance, Health
- `internal/agent/reasoning.go` — ghi `incidents`, `routing_plans`, `recommendations`; auto routing + auto maintenance khi `ShouldAutoApplyScope`; force bypass chu kỳ: `ShouldForceAutoRouting`, `ShouldForceAutoMaintenanceAllProviders`, `ShouldForceAutoMaintenance`
- LLM: fallback template khi `LLM_API_URL` trống (§7.6)

### Phase 5 — REST API + SSE ✅
- `cmd/api` — HTTP server `:8080` (§2.3)
- `internal/api` — handlers, CORS, `DEV_AUTH_BYPASS`, SSE poll 5s
- `GET /dashboard/overview` — ẩn `pending_plan` / `pending_maintenance` khi `ShouldAutoApplyScope=true`; poll auto apply (`autoApplyScopeFromSnapshot`, tối đa 2 pass routing→bảo trì). Từ chối routing/bảo trì → poll tiếp có thể sinh đề xuất mới (không cooldown).
- `POST .../maintenance/extend` — **400** `no_change` khi gia hạn không đổi thời gian (`CountActiveMaintenanceForSKU`)
- Endpoints: health-status, config, incidents (phân trang), scopes auto/routing/maintenance, maintenance, chat (stub), `/events`
- Integration test: `$env:OPSONE_INTEGRATION=1; go test ./internal/api/... -v` — gồm `TestProductScopeAutoPutOverview`, `TestConfigPutMaintenanceDefaultDuration`

### Phase 6 — Frontend React ✅
- `web/` — Vite + React 18 + TypeScript + React Router + TanStack Query
- Routes §9: `/`, `/incidents`, `/settings`, `/maintenance` (không `/incidents/:id`, không `/chat` riêng)
- **`Settings`** — card compact §9.5 (`max-width: 40rem`): scheduler + **thời lượng BT mặc định** + mock; select kịch bản full-width
- **`ServiceOverviewTable`** — tab Thẻ/Topup/Data; group SKU `rowspan`; cột provider 6 chỉ số; hàng con plan/bảo trì/**Mở lại provider**; `RedSkuScrollNav` ▲ 🔴 1/N ▼
- **`ProductThresholdEditor`** — hàng **Ngưỡng cảnh báo** đầu nhóm: `product_label` · label ngưỡng · 5 cụm %/GD thẳng cột Provider (§9.5.3)
- **`SkuMaintenanceTimeLabel`** — nhãn BT 4 dòng dưới SKU; **`ServiceMaintenanceButton`** / **`ActiveMaintenanceCell`**; cột SKU gộp **Bảo trì + Chế độ BT/Routing** (`scope-controls-cell`)
- **`ProductMaintenanceActions`** — **Mở lại dịch vụ** / **Gia hạn bảo trì** batch cột Dịch vụ — **chỉ khi mọi SKU** trong dịch vụ đang BT (§9.0.7)
- Toast scope: `{product_label} · {SKU}` (`scopeDisplayLabel`)
- **`ScopeAutoEditor`** — chế độ BT/routing cấp dịch vụ + SKU; compact + ⋯; **Lưu** riêng (`PUT /scopes/.../auto`, không dùng Lưu hàng Ngưỡng); `patchOverviewCache` + refetch; ẩn hàng đề xuất khi `ShouldAutoApplyScope` (`utils/scopeAuto.ts`, `ResolveEffectiveScopeAuto`)
- **Mở lại** provider — hàng *Mở lại provider*: **Trả lại** → baseline; **Lưu** → `restore-baseline` (baseline) hoặc `routing/apply` (tùy chỉnh)
- **Mở lại dịch vụ** — `POST .../maintenance/reopen-service` (atomic baseline + grace)
- Chat widget — kéo dock 4 góc (`useChatDock`); voice `vi-VN` (`useVoiceInput`)
- SSE `/events` + poll 60s fallback
- Dev: `VITE_DEV_AUTH_BYPASS=true` (header `X-OpsOne-Role`); production: MSAL.js §2.6.4

### Phase 7 — E2E & Definition of Done ✅
- `internal/e2e/` — kịch bản §10/§12
- `make test-e2e` / `Invoke-OpsOneE2E` — integration với MySQL
- `scripts/e2e.ps1` — seed UTF-8 + E2E smoke

**Vận hành (demo):**
| Vai trò | Hành vi |
|---------|---------|
| **Ops admin** | Duyệt routing/bảo trì khi SKU **Chỉ đề xuất** (hoặc **Tự động theo khung giờ** ngoài giờ) |
| **Agent auto** | `auto_action=auto` hoặc `time_window` (trong khung) → tự `UpdateRouting` / `SetMaintenance`; Dashboard không hiện đề xuất |
| **Agent auto — force** | Còn provider khỏe → `ShouldForceAutoRouting` (shift ngay); tất cả provider đỏ → `ShouldForceAutoMaintenanceAllProviders`; chỉ 1 provider routable + đỏ → `ShouldForceAutoMaintenance` |
| **UI TT** | Ẩn icon 🟢🟡🔴 khi scope đang bảo trì (`maintenanceDisplay.ts`) |
| **On-call** | Restart `worker-agent` / `worker-mock` khi crash |

**Chế độ BT / Routing (§9.5.2):** cấu hình **cấp dịch vụ** (`PUT /scopes/{product}/auto`, `sku_code=""`) ưu tiên trước **cấp SKU** khi đã lưu row dịch vụ (`ResolveEffectiveScopeAuto`). Overview trả ba lớp: `product_auto_action`, `scope_auto_action`, `auto_action` (hiệu lực). Backend: `ScopeAutoMapKey` trong `dashboard.go` — restart `cmd/api` sau đổi handler overview.

| `auto_action` | Nhãn UI | Hành vi |
|---------------|---------|---------|
| `recommend_only` | Chỉ đề xuất | Plan / đề xuất bảo trì chờ Duyệt |
| `auto` | Tự động | Agent tự apply; UI không hiện hàng đề xuất |
| `time_window` | Tự động theo khung giờ | Trong khung → như `auto` (force routing/bảo trì §9.5.2); **ngoài khung** → như `recommend_only` (hiện Duyệt/Từ chối) |

**Routing khôi phục:**
| Hành động UI | API | `trigger_type` |
|--------------|-----|----------------|
| **Mở lại** provider | `POST .../routing/restore-baseline` | `manual_baseline` + `recovery_apply_cycle_id` |
| **Mở lại dịch vụ** | `POST .../maintenance/reopen-service` | hủy BT + baseline + grace (atomic) |
| Routing tùy chỉnh (nếu cần) | `POST .../routing/apply` | `manual_temp` |

## Quick start

```powershell
cd ai_gen_src

# Helpers (optional)
. .\scripts\dev.ps1

copy .env.example .env
Invoke-OpsOneReset

go test ./...
$env:OPSONE_INTEGRATION="1"; go test ./... -v
.\scripts\e2e.ps1
```

### Chạy workers

```powershell
go run ./cmd/worker-mock
go run ./cmd/worker-agent
go run ./cmd/api
cd web; npm run dev
# http://localhost:5173
```

### Chạy React UI

```powershell
cd web
copy .env.example .env
npm install
npm run dev
```

## Cấu trúc (rút gọn)

```text
ai_gen_src/
├── cmd/api, worker-mock, worker-agent
├── internal/agent, api, store, tools, …
├── db/schema.sql, seed.sql
└── web/src/
    ├── components/   # ServiceOverviewTable, ScopeAutoEditor, ProductThresholdEditor, …
    ├── hooks/        # useChatDock, useSSE, useVoiceInput, useOverallHealth, useMaintenanceDefaultDurationMin
    ├── utils/        # scopeAuto, dashboardRowOrder (scopeDisplayLabel), maintenanceWindow, …
    └── pages/        # Dashboard, IncidentsPage, Settings, …
```

Spec đầy đủ: [`../OpsOne.md`](../OpsOne.md) — §9.0 Dashboard, §9.5 Cấu hình (compact), §9.5.2 Chế độ BT / Routing, §9.5.5 Thời lượng BT mặc định.
