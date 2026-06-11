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
- LLM scheduler: fallback template khi chưa gọi LLM summary (§7.6); chat on-demand §7.6.5 tách riêng

### Phase 5 — REST API + SSE ✅
- `cmd/api` — HTTP server `:8080` (§2.3)
- `internal/api` — handlers, CORS, `DEV_AUTH_BYPASS`, SSE poll 5s
- `GET /dashboard/overview` — ẩn `pending_plan` / `pending_maintenance` khi `ShouldAutoApplyScope=true`; poll auto apply (`autoApplyScopeFromSnapshot`, tối đa 2 pass routing→bảo trì). Từ chối routing/bảo trì → poll tiếp có thể sinh đề xuất mới (không cooldown).
- `POST .../maintenance/extend` — **400** `no_change` khi gia hạn không đổi thời gian (`CountActiveMaintenanceForSKU`)
- Endpoints: health-status, config, incidents (phân trang), scopes auto/routing/maintenance, maintenance, **`POST /chat` LLM agent §7.6.5** (stub keyword khi không có `LLM_API_KEY`), `/events`
- **`internal/llm`** — OpenAI-compatible client (GreenNode AIP); **`internal/api/chat_agent.go`** — tool calling + session memory
- **`internal/api/chat_metrics.go`** — `metricsForChat` + `tryChatMetricsReply` (§7.6.5.2): GD pending/success/fail khớp `provider_metrics` Dashboard; **không** fall-through LLM
- **`internal/api/chat_maintenance.go`** — `maintenanceForChat` + `tryChatMaintenanceReply` (§7.6.5.1): cùng query/lọc như Dashboard; **không** fall-through LLM
- **`internal/chatresolve`** — alias (`aliases.go`), intent metrics/BT (`intent.go`), SKU `10.000`→`10000` (`sku.go`)
- **`internal/tools/metrics_format.go`** — `FormatMetricsReply`, `BuildMetricsChatResult`
- **`internal/tools/maintenance_format.go`** — `FormatMaintenanceReply`, `FormatAllMaintenanceReply`, `EnrichMaintenanceOutput`
- **`internal/tools/maintenance_build.go`** — `MaintenanceInWindow`, `BuildMaintenanceOutput`, `FilterMaintenanceByProvider`
- **`internal/store/chat_intent_stats.go`** — `BumpChatIntentStat` — thống kê câu hỏi thường gặp (§7.6.5.3)
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
- **`Layout`** — logo ZaloPay header + favicon tab (`web/public/favicon.png`)
- Chat widget — panel ~800×780px; nhãn **Bạn/OpsOne**; kéo dock 4 góc (`useChatDock`); resize 4 góc (`useChatResize`); auto-focus ô nhập khi mở
- Voice `vi-VN` (`useVoiceInput`) — Mic toggle liên tục; im lặng **2s** → gửi; restart STT sau gửi (clear transcript); watchdog chống đơ; lệnh thoại *tắt mic* / *kết thúc cuộc trò chuyện*
- Chat metric/BT §7.6.5.2–1 — *Mobifone 50.000 GD pending/banding*, *thẻ Garena*, *ngoài ra còn dịch vụ nào BT*; follow-up session; LLM + alias §7.6.5; Admin duyệt khi yêu cầu rõ
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

copy .env.example .env   # LLM_API_KEY, LLM_MODEL=minimax/minimax-m2.5, …
Invoke-OpsOneReset

go test ./...
$env:OPSONE_INTEGRATION="1"; go test ./... -v
.\scripts\e2e.ps1
```

**LLM chat:** `LLM_API_KEY` + `LLM_MODEL=minimax/minimax-m2.5` trong `.env`. API tự đọc `.env` khi chạy từ `ai_gen_src/`; khuyến nghị `.\scripts\run-api.ps1`.

**Chat alias, metric & bảo trì:** `NormalizeToolArgs` + `NormalizeSKU`. Câu pending/metric → `metricsForChat` (giống `provider_metrics` Dashboard, cửa sổ 15m). Câu BT → `maintenanceForChat` (giống `ListMaintenanceWindows` + `MaintenanceInWindow`). Intent metrics **ưu tiên** trước BT — tránh nhầm *quay đơn/banding* thành bảo trì. `chat_intent_stats` ghi hit count the FAQ. Deploy API sau đổi chat. Provider routing: `ESALE`, `IMEDIA`, `SHOPPAY`. Chi tiết: [`OpsOne.md` §7.6.5 / §7.6.5.1–3](../OpsOne.md).

### Chạy workers + API

```powershell
go run ./cmd/worker-mock
go run ./cmd/worker-agent
.\scripts\run-api.ps1          # khuyến nghị: nạp .env + kill port + API
# hoặc: go run ./cmd/api       # cần nạp env thủ công
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

### Deploy GreenNode AgentBase (4 runtimes)

```powershell
# IAM: opsone/.env  |  Container: ai_gen_src/.env.greennode
cd ai_gen_src
.\scripts\deploy-greennode-all.ps1
# Dashboard: endpoint opsone-web | API: endpoint opsone-api
```

Chi tiết §15.2.2 trong `OpsOne.md` (MYSQL_DSN, không `allowPublicKeyRetrieval` trong Go, `VITE_API_BASE_URL` khi build web).

## Cấu trúc (rút gọn)

```text
ai_gen_src/
├── cmd/api, worker-mock, worker-agent
├── internal/agent, api, chatresolve, llm, store, tools, …
│   ├── api/chat_metrics.go           # metricsForChat — GD pending khớp Dashboard
│   ├── api/chat_maintenance.go       # maintenanceForChat — khớp Dashboard
│   ├── chatresolve/intent.go, sku.go
│   ├── store/chat_intent_stats.go    # FAQ hit count
│   └── tools/metrics_format.go, maintenance_format.go, maintenance_build.go
├── db/schema.sql, seed.sql
├── Dockerfile, Dockerfile.worker-*
├── scripts/run-api.ps1, dev.ps1, deploy-greennode*.ps1
└── web/src/
    ├── components/   # ServiceOverviewTable, ScopeAutoEditor, ProductThresholdEditor, …
    ├── hooks/        # useChatDock, useSSE, useVoiceInput, useOverallHealth, useMaintenanceDefaultDurationMin
    ├── utils/        # scopeAuto, dashboardRowOrder (scopeDisplayLabel), maintenanceWindow, …
    └── pages/        # Dashboard, IncidentsPage, Settings, …
```

Spec đầy đủ: [`../OpsOne.md`](../OpsOne.md) — §7.6.5 Chat LLM, **§7.6.5.2 Metric/GD pending**, **§7.6.5.1 Tra cứu BT**, **§7.6.5.3 FAQ stats**, §9 Chat/Voice, §9.0 Dashboard, §15.2.2 GreenNode deploy, §9.5 Cấu hình, §9.5.2 Chế độ BT / Routing.
