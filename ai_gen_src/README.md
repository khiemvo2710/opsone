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
- **`internal/api/chat_commands.go`** — `tryChatCommandReply`: duyệt/từ chối ngắn (`ok`/`không`), bảo trì chủ động, mở lại dịch vụ, đổi chế độ scope auto; session pending focus
- **`internal/api/chat_maintenance.go`** — `maintenanceForChat` + `tryChatMaintenanceReply` (§7.6.5.1): cùng query/lọc như Dashboard; **không** fall-through LLM
- **`internal/chatresolve`** — alias (`aliases.go`), intent metrics/BT (`intent.go`), SKU `10.000`→`10000` (`sku.go`)
- **`internal/tools/metrics_format.go`** — `FormatMetricsReply`, `BuildMetricsChatResult`
- **`internal/tools/maintenance_format.go`** — `FormatMaintenanceReply`, `FormatAllMaintenanceReply`, `EnrichMaintenanceOutput`
- **`internal/tools/maintenance_build.go`** — `MaintenanceInWindow`, `BuildMaintenanceOutput`, `FilterMaintenanceByProvider`
- **`internal/store/chat_intent_stats.go`** — `BumpChatIntentStat` — thống kê FAQ + route/success/fail (§7.6.5.3, P2)
- **`internal/store/chat_log.go`** — persist `chat_sessions`, `chat_messages`, `chat_interaction_log` (§7.6.5.5 P1)
- **`internal/api/chat_turn.go`** — `persistChatTurn`, route metadata mỗi `/chat` (§7.6.5.5 P2)
- **Chat học kinh nghiệm (§7.6.5.5):** P1–P2 ✅ · P3–P5 📋 — DDL `chat_command_patterns`, `chat_feedback`, `chat_few_shot_examples`, `chat_voice_corrections`, `chat_user_prefs` trong `db/schema.sql`
- Integration test: `$env:OPSONE_INTEGRATION=1; go test ./internal/api/... -v` — gồm `TestChatPostPersistsInteractionLog`, `TestProductScopeAutoPutOverview`, …

### Phase 5.1 — Chat học kinh nghiệm ✅ spec (§7.6.5.5)

| Phase | Việc | Trạng thái |
|-------|------|------------|
| P0 | Rule + `tryChatCommandReply` + `chat_intent_stats` | ✅ |
| P1 | Persist `chat_sessions` / `chat_messages` / `chat_interaction_log` | ✅ |
| P2 | Log route, slots, action_result mỗi `/chat` | ✅ |
| P3 | Job đào pattern + Admin promote UI | ✅ |
| P4 | `chat_few_shot_examples` → prompt LLM | ✅ |
| P5 | `chat_voice_corrections` + tín hiệu retry | ✅ |

**P3 — Store + API:**
- `internal/store/chat_patterns.go` — `MineCommandPatterns`, `ListCommandPatterns`, `PromoteCommandPattern`, `DeprecateCommandPattern`, `ListApprovedPatterns`, `MatchApprovedPattern`, `InsertChatFeedback`
- `internal/store/chat_patterns.go` — `MineFewShotExamples`, `GetApprovedFewShotExamples`, `ListFewShotExamples`, `PromoteFewShotExample`
- `internal/api/chat_admin.go` — Admin endpoints: `GET/POST /api/v1/admin/chat-patterns`, `POST .../mine`, `POST .../mine-few-shot`, `GET /api/v1/admin/few-shot`, `GET /api/v1/admin/voice-corrections`, `POST /api/v1/chat/feedback`
- Router P3+: approved pattern check **trước** rule code trong `chatAgentReply`

**P4 — Few-shot injection:**
- `chatFewShotHint(ctx, commandKey)` — query `GetApprovedFewShotExamples` theo intent, inject vào system prompt LLM

**P5 — Voice corrections + retry:**
- `internal/store/chat_voice.go` — `UpsertVoiceCorrection`, `GetVoiceCorrections`, `ApplyVoiceCorrections`, `MarkInteractionWrongRoute`, `FindRecentInteraction`, `DetectRetrySignal`
- `handleChatPost`: apply corrections trước routing (voice input); record cặp `stt_raw → message`; detect retry trong 60s → mark previous as `wrong_route`
- `voiceCorrectionsCache` — in-memory TTL 5 phút tránh DB query mỗi request

DDL: `db/schema.sql` §13.9. Chi tiết: [`../OpsOne.md` §7.6.5.5](../OpsOne.md).

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
- Chat widget — panel ~800×780px; nhãn **Anh/Chị + tên** / **OpsOne**; onboarding client §9.2.1 (`chatUserProfile`, `chatOnboarding`); kéo dock 4 góc; resize; wake **alo** (`useOpsOneWake`)
- Voice `vi-VN` (`useVoiceInput`) — Mic toggle; im lặng **2s** → gửi; gửi `input_source: voice` + `stt_raw` tới `/chat` §7.6.5.5; *đóng chat* / *tắt mic* / *bye bye*; sửa ASR (`ăn`→`anh`); avatar sprite 7×4; cleanup recognition khi unmount (logout)
- Wake word `useOpsOneWake` — khẩu lệnh thuần (`mở chat`, `bật mic`) không pass remainder vào input; `matchesMicOnPhrase` check trước `stripAloWakePrefix`
- **Login mic** — checkbox "Cho phép microphone" gọi `getUserMedia` ngay khi tick (user gesture); hiện cảnh báo nếu browser block; **không** auto-prime khi mount (tránh popup quyền lúc load trang)
- **`RedSkuScrollNav`** render trong `ChatWidget` trước toggle button (DOM order = visual order với flex-column); bọc `stopPropagation` trên `onPointerDown`+`onPointerUp` để tránh mở chat khi click nav
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

## Gotchas kỹ thuật

### HTTP header + tên tiếng Việt (Error 1366)
**Fetch API spec** encode header values bằng **Latin-1** (ISO-8859-1), không phải UTF-8. Khi browser gửi `X-OpsOne-Actor: Khiêm`, byte nhận được là `\xEA\x6D` (Latin-1) thay vì `\xC3\xAA\x6D` (UTF-8). MySQL với column `utf8mb4` từ chối `\xEA\x6D` vì không phải UTF-8 hợp lệ.

**Fix:** `latin1HeaderToUTF8()` trong `internal/api/middleware.go` — convert mỗi Latin-1 byte → Unicode rune tương ứng (Latin-1 ↔ Unicode là ánh xạ 1-1) → Go string UTF-8 hợp lệ. ASCII-only name không bị ảnh hưởng.

```go
// internal/api/middleware.go
func latin1HeaderToUTF8(s string) string {
    for i := 0; i < len(s); i++ {
        if s[i] > 0x7F {
            runes := make([]rune, len(s))
            for j, b := range []byte(s) { runes[j] = rune(b) }
            return string(runes)
        }
    }
    return s
}
```

Áp dụng trong `actorFromRequest` — tất cả `updated_by`/`approved_by` lấy từ header đều được decode đúng.

### MySQL DSN charset
`ensureDSNCharset()` trong `internal/config/config.go` không chỉ append `charset=utf8mb4` khi thiếu, mà còn **replace** `charset=utf8` (hoặc charset khác) thành `charset=utf8mb4`. DSN từ `.env.greennode` có thể đã có `charset=utf8` nên cần replace, không chỉ append.

### Routing / Bảo trì sai provider
`redistributeAfterMaintenance` (`internal/api/maintenance_helpers.go`) kiểm tra `ListActiveMaintenance` để loại trừ các provider **đang** bảo trì trước khi phân bổ traffic. `chatSetMaintenanceScope` cancel pending routing plans ngay khi nhận lệnh bảo trì thủ công — tránh proposal cũ ghi đè.

### chatMaintenanceTargets — priority khi chọn provider để bảo trì
`chatMaintenanceTargets` (`internal/api/chat_commands.go`) chọn provider theo 3 bước:

1. **DB recommendation** (`LatestPendingMaintenanceForScope`) → provider agent đề xuất.
2. **Pending routing plan** (`GetPendingRoutingPlanForScope`) → provider nào đang **active (>0%)** nhưng plan đề xuất về **0%** → đó là provider xấu cần bảo trì. Xử lý trường hợp dashboard hiển thị đề xuất routing nhưng chưa có DB recommendation.
3. **Routing-excluded** (TrafficPct==0, không đang BT) → agent đã de-route provider xấu trước đó.
4. **Active providers** (TrafficPct>0) — fallback cuối.

**Lỗi cũ:** Khi IMEDIA=100% (xấu), ESALE=0% và không có DB recommendation → Step 3 trả ESALE (sai). Fix: Step 2 kiểm tra pending plan → plan đề xuất IMEDIA→0% → bảo trì IMEDIA (đúng).

### chatSetMaintenanceScope — routing-only path
Khi user lệnh bảo trì provider có tên (vd: "bảo trì qua IMEDIA") và còn provider khác active → **routing-only**: chỉ UpdateRouting về 0%, **không** tạo maintenance window. Sau UpdateRouting, gọi thêm `CancelActiveMaintenanceForSKUProvider` để hủy cửa sổ BT cũ của provider đó — tránh cửa sổ BT orphan chặn "Mở lại dịch vụ".

### Mở lại dịch vụ — xử lý orphan windows
`chatReopenService` và `POST .../maintenance/reopen-service` không hard-error khi `CountActiveMaintenanceForSKU > 0` sau lần cancel đầu. Thay vào đó: sweep thêm qua `activeMaintenanceIDsForScope` + `CancelMaintenanceByIDs`, rồi tiếp tục restore routing bất kể còn window hay không.

### maintenanceOverview — bao gồm tất cả provider (kể cả 0%)
`maintenanceOverview` (`internal/api/maintenance_display.go`) không còn filter theo `activeSet` (providers có traffic>0). Maintenance window cho provider đang ở 0% routing vẫn được tính vào `row.maintenance` — đảm bảo nút "Mở lại dịch vụ" và chip BT hiển thị đúng trạng thái. Màu provider: chỉ tô cam (maintained) khi `routingPct > 0`; provider ở 0% luôn hiển thị xám/inactive.

### Product alias — "top up" (có dấu cách / gạch ngang)
`NormalizeKey` convert `-` thành space, nên "top-up" → "top up". Đã thêm alias `"top up mobi"`, `"top up mobifone"`, `"top up vina"`, `"top up vinaphone"`, `"top up viettel"` vào `staticProductAliases` trong `internal/chatresolve/aliases.go`.

### Carrier ambiguity — hỏi lại khi tên dịch vụ không rõ loại
Khi user nói "Mobifone" / "Viettel" không có qualifier (topup/thẻ/data), `IsAmbiguousCarrierProduct` (`internal/chatresolve/intent.go`) trả `true`. Các handler `tryChatSetMaintenanceReply`, `tryChatReopenReply`, `tryChatSetScopeAutoReply` sẽ hỏi lại thay vì đoán sai product. Qualifier nhận ra: `topup`, `top up`, `nap`, `thẻ`, `the `, `data`, `nạp`.

### Global chat commands — bảo trì/mở lại toàn bộ hoặc theo loại dịch vụ

| Lệnh chat | Action | Ghi chú |
|-----------|--------|---------|
| `bảo trì tất cả dịch vụ` | `UIActionSetAllMaintenance` | Tất cả products + SKUs |
| `mở lại tất cả dịch vụ` | `UIActionReopenAllServices` | Tất cả products + SKUs |
| `bảo trì tất cả thẻ` | `UIActionSetAllMaintenance` + filter `card` | Chỉ card products |
| `bảo trì tất cả topup` | `UIActionSetAllMaintenance` + filter `topup` | Chỉ topup products |
| `bảo trì tất cả data` | `UIActionSetAllMaintenance` + filter `topup_data` | Chỉ data products |
| `mở lại tất cả thẻ/topup/data` | `UIActionReopenAllServices` + filter | Tương tự |

`isAllServicesPhrase` (`commands.go`): nhận ra "tất cả thẻ/topup/data" bằng cách kiểm tra `serviceTypeTokens` (`"topup"`, `"top up"`, `" nap"`, `" data"`) và `strings.HasSuffix(key, " the")` ngoài `allObjectTokens` tiêu chuẩn.

`ExtractAllServicesFilters(msg)` → `[]string` với các giá trị `"card"` / `"topup"` / `"topup_data"`. Empty slice = tất cả dịch vụ. Nhiều loại trong một câu ("data và top up") → `["topup_data","topup"]`; `allScopesForFilter` dùng `filterSet` OR-logic để include product nếu `ServiceType` khớp bất kỳ filter nào.

**Bug fix — topup không được bảo trì khi global command:** Topup products có `routing_mode = 'provider'` (sku_code = '') → `ListSKUsForProduct` trả empty → inner loop không chạy → topup bị bỏ qua. Fix: `allScopesForFilter` kiểm tra `p.RoutingMode == domain.RoutingProvider`; nếu đúng, append scope `{product, ""}` trực tiếp mà không cần ListSKUs.

**Bug fix — một số SKU không được bảo trì khi global command:** Hai nguyên nhân:
1. `chatSetMaintenanceScope(provider="")` gọi `chatMaintenanceTargets` (chọn provider xấu, không phải tất cả) → nếu providers đã trong `maintSet` hoặc routing-excluded → trả empty → scope bị skip.
2. `applyMaintenanceTargets` có rollback toàn scope khi bất kỳ provider nào fail (e.g. SHOPPAY disabled) → ESALE và IMEDIA bị cancel lại theo. Scope có active routing (V100K, VNP20) bị ảnh hưởng; scope đã 0% (V50K, VNP50) không bị rollback vì SHOPPAY cũng ở 0%.

Fix: `chatSetMaintenanceScopeAll` — bypass `chatMaintenanceTargets`, dùng **no-rollback loop** gọi `SetMaintenance` từng provider độc lập. Provider fail không ảnh hưởng đến các provider khác trong cùng scope. STT alias "cấp áp" → `"cap ap"` thêm vào `serviceTypeTokens` và `staticProductAliases`.

**Bug fix — "top up" (có space) và STT mishear "tốt áp" không được nhận là loại topup:** `NormalizeKey` không convert "top up" → "topup"; STT thường transcribe "top up" thành "tốt áp" → `foldCommandKey` → "tot ap". Fix: thêm `"top up"`, `"tot ap"`, `"top ap"` vào `serviceTypeTokens` và vào check trong `ExtractAllServicesFilters`. Thêm alias tương ứng vào `staticProductAliases` (e.g. `"tot ap mobifone"` → `TOPUP_MOBI`).

**Performance — concurrent scope processing:** `tryChatSetAllMaintenanceReply` và `tryChatReopenAllServicesReply` trước đây chạy tuần tự qua từng scope. Fix: `allScopesForFilter` collect tất cả scopes trước, sau đó chạy song song với goroutine pool (semaphore cap=8, `sync.WaitGroup` + `atomic.AddInt64`). Với ~30 scopes, thời gian giảm ~8×.

**Bug fix — maintenance_id collision khi concurrent:** `buildMaintenanceID` (`internal/tools/set_maintenance.go`) dùng millisecond (`now.Nanosecond()/1_000_000`) làm suffix → 2 goroutine gọi `SetMaintenance` cho cùng provider (ESALE) trong cùng millisecond → ID trùng → `InsertMaintenance` fail với duplicate key → SKU đó không có maintenance nhưng không báo lỗi (no-rollback). Fix: thay bằng **monotonic process-wide counter** (`atomic.AddInt64(&maintIDSeq, 1) % 10000`) — đảm bảo unique tuyệt đối trong cùng tiến trình. Format mới: `MT{yyMMddHHmmss}{PROV}{4-digit-counter}` (max 24 chars).

**Bug fix — "bảo trì tất cả thẻ Garena" bảo trì tất cả card thay vì chỉ Garena:** `isAllServicesPhrase` tìm thấy `" the "` (space hai bên) trong "tat ca cac the garena" → `IsSetAllMaintenanceCommand` = true → global all-card command. Fix: `IsSetAllMaintenanceCommand` và `IsReopenAllServicesCommand` gọi `ExtractProductFromText(msg)` trước khi return true; nếu có product cụ thể → return false, nhường cho `IsSetMaintenanceCommand` / `IsReopenServiceCommand` xử lý đúng product đó.

**Bug fix — maintenance_id collision khi concurrent:** `buildMaintenanceID` (`internal/tools/set_maintenance.go`) dùng millisecond (`now.Nanosecond()/1_000_000`) làm suffix → 2 goroutine gọi `SetMaintenance` cho cùng provider code (ESALE) trong cùng millisecond → ID trùng → `InsertMaintenance` fail duplicate key → SKU bị bỏ qua không có maintenance nhưng không báo lỗi. Fix: thay bằng **monotonic process-wide counter** (`atomic.AddInt64(&maintIDSeq, 1) % 10000`). Format mới: `MT{yyMMddHHmmss}{PROV}{4-digit-counter}` (max 24 chars, tuyệt đối unique trong process).

**Fix — email templates thiếu dấu tiếng Việt:** Toàn bộ chuỗi tiếng Việt trong `internal/notify/service.go` đã được bổ sung dấu đầy đủ. Các tiêu đề email: `BẢO TRÌ BẮT ĐẦU`, `BẢO TRÌ KẾT THÚC`, `BẢO TRÌ ĐÃ HỦY`, `BẢO TRÌ LÊN LỊCH`, `ĐIỀU CHỈNH ROUTING`, `PHỤC HỒI THẤT BẠI`. Nội dung: `Sản phẩm`, `Lý do`, `Hành động`, `Người thực hiện`, `Lưu ý`, `Ngưỡng vượt`, `Lỗi`. Footer: `Thời điểm`, `Hệ thống giám sát thanh toán tự động`.

### Topup product — single-product maintenance/reopen cũng bị "không có SKU"

`chatSetMaintenance` và `chatReopenService` có nhánh `sku == ""` gọi `ListSKUsForProduct` để lấy danh sách SKU rồi loop. Topup products (`routing_mode='provider'`) không có SKU row → list trả empty → cả hai hàm trả lỗi `"không có SKU cho TOPUP_MOBI"`.

**Fix (`internal/api/chat_commands.go`):**
- `chatSetMaintenance`: khi `len(skus) == 0` → gọi `chatSetMaintenanceScopeAll(ctx, product, "", startsAt, endsAt, actor)` thay vì báo lỗi.
- `chatReopenService`: khi `len(skus) == 0` → gọi trực tiếp `CancelActiveMaintenanceForSKU(ctx, product, "", actor)` + `CancelPendingRoutingPlansForScope(ctx, product, "")` + `applyScopeReopenRouting(ctx, product, "", ...)` thay vì báo lỗi.

Áp dụng cho tất cả lệnh đơn sản phẩm có topup: "bảo trì topup mobifone", "mở lại topup vina", v.v.

### STT artifact "topup up" — nhận sai là thẻ

STT đôi khi transcribe "topup Mobifone" thành "topup up Mobifone" (lặp "up"). Khi đó `NormalizeKey` cho ra `"topup up mobifone"`:
- Không khớp alias `"topup mobifone"` (thiếu "up") cũng không khớp `"top up mobifone"` (không có space giữa "top" và "up")
- Partial match fallback tìm `"mobifone"` (length 8) → MOBIFONE (thẻ) ❌

**Fix:** thêm alias `"topup up mobi"` / `"topup up mobifone"` / `"topup up vina"` / `"topup up vinaphone"` / `"topup up viettel"` vào `staticProductAliases` (`internal/chatresolve/aliases.go`). Với longest-match, alias dài hơn (13–17 ký tự) thắng "mobifone" (8 ký tự).

### SSE bị buffer bởi GreenNode nginx proxy — chat không tự mở

ChatWidget dùng `EventSource` để nhận `pending_suggestions`. Trên GreenNode, nginx reverse proxy mặc định buffer HTTP response → SSE không stream tới client → `onerror` fire → `es = null` → auto-open chat không bao giờ kích hoạt. Dashboard vẫn refresh được vì `useSSE` (`hooks/useSSE.ts`) có fallback poll 30s riêng.

**Fix 3 điểm:**
1. `internal/api/sse.go`: thêm `X-Accel-Buffering: no` header → báo nginx không buffer SSE.
2. `internal/api/sse.go` + `server.go`: thêm `GET /api/v1/suggestions` endpoint (REST fallback, gọi `getPendingSuggestionsForSSE`).
3. `web/src/components/ChatWidget.tsx`: khi SSE fail → tự switch sang poll `/suggestions` mỗi 30s. Khi mount, gọi `/suggestions` ngay lập tức — nếu `has_suggestions=true` thì mở chat và hiện đề xuất ngay (bỏ logic `initialLoadRef` suppress 10s cũ).

### Email cảnh báo vượt ngưỡng (breach) — gửi lại theo giờ

`SendIfNeeded` dùng dedup key `"product:provider:sku:breach"` vĩnh viễn → sau lần đầu không bao giờ gửi lại. Fix: key breach đổi thành `"product:provider:sku:breach:{YYYYMMDDHH}"` (bucket theo giờ UTC) → tối đa 1 email breach/scope/giờ. DeepLink trong email từ hardcode `localhost:5173` → thay bằng `DASHBOARD_URL` env var (config + `Runner.DashboardURL`).

### Email cảnh báo sự cố mới (incident_open)

Khi incident mới được tạo (`existingID == ""` trong `emitOutputs`, `internal/agent/reasoning.go`), gửi ngay 1 email với `TriggerEvent: "incident_open"`. Dedup key = `"incident:{incidentID}"` → đảm bảo gửi đúng 1 lần. Goroutine copy `thRes.BreachReasons` trước khi launch để tránh race condition.

`renderIncidentOpen` (`internal/notify/service.go`): subject `[OpsOne] 🔴 SỰ CỐ MỚI {id} | {scope}`, body gồm tỷ lệ metrics, ngưỡng vượt, tóm tắt, link tới `/incidents`.

`Reasoner` nhận thêm `DashboardURL string` (variadic arg); `Runner.DashboardURL` được truyền qua `NewReasoner`. `DASHBOARD_URL` env var được thêm vào `config.Config` và `.env.example` / `.env.greennode`.

### Email footer
`writeFooter` trong `internal/notify/service.go` dùng `\n` plain (không dùng `writeSep`) trước dòng thời điểm — tránh Outlook collapse dòng cuối bullet với footer.

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

**Chat alias, metric & bảo trì:** `NormalizeToolArgs` + `NormalizeSKU`. Câu pending/metric → `metricsForChat` (giống `provider_metrics` Dashboard, cửa sổ 15m). Câu BT → `maintenanceForChat` (giống `ListMaintenanceWindows` + `MaintenanceInWindow`). Intent metrics **ưu tiên** trước BT — tránh nhầm *quay đơn/banding* thành bảo trì. `chat_intent_stats` + `chat_interaction_log` ghi mỗi lượt §7.6.5.3–5. Deploy API sau đổi chat. Provider routing: `ESALE`, `IMEDIA`, `SHOPPAY`. Chi tiết: [`OpsOne.md` §7.6.5 / §7.6.5.1–5](../OpsOne.md).

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
│   ├── api/chat_turn.go              # persistChatTurn — log §7.6.5.5 P1–P2
│   ├── chatresolve/intent.go, sku.go, commands.go
│   ├── store/chat_intent_stats.go    # FAQ hit count (§7.6.5.3)
│   ├── store/chat_log.go             # sessions / messages / interaction_log (§7.6.5.5)
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

Spec đầy đủ: [`../OpsOne.md`](../OpsOne.md) — §7.6.5 Chat LLM, **§7.6.5.2 Metric/GD pending**, **§7.6.5.1 Tra cứu BT**, **§7.6.5.3 FAQ stats**, **§7.6.5.4 Lệnh trực tiếp**, **§7.6.5.5 Học kinh nghiệm**, §9 Chat/Voice, §9.0 Dashboard, §15.2.2 GreenNode deploy, §9.5 Cấu hình, §9.5.2 Chế độ BT / Routing.
