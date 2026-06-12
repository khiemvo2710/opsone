# OpsOne — Spec triển khai (Business + Code)

> **OpsOne** = **Ops** (vận hành) + **One-man team** — một AI Agent thay vai trò cả đội ops: chủ động phân tích traffic, phát hiện sự cố, đề xuất/điều phối routing.  
> **Mục đích:** Một file duy nhất để **con người** và **AI coding engine** đọc → hiểu business → build **OpsOne** chạy thật, test được.  
> **Stack mặc định:** **MySQL 8** · **Go 1.21** · **React 18** (Vite).

## Các bước phát triển chính

> **Thứ tự bắt buộc:** làm tuần tự từ bước 1 → 8. Chi tiết từng bước: **§2.4** (19 bước verify) · **§11** (checklist phase) · **§12** (Definition of Done).


| #     | Bước                      | Nội dung chính                                                                            | Deliverable                                               | Tham chiếu     | Verify nhanh                                      |
| ----- | ------------------------- | ----------------------------------------------------------------------------------------- | --------------------------------------------------------- | -------------- | ------------------------------------------------- |
| **1** | **Chuẩn bị**              | Monorepo Go + React; MySQL DDL & seed catalog 11 product                                  | `db/schema.sql`, `db/seed.sql`, cấu trúc §2.2             | §2.2, §13      | `SELECT COUNT(*) FROM products` = 11              |
| **2** | **Mock Data + Scheduler** | Sinh data giả mỗi 1 phút; scheduler Agent 5 phút; ghi lịch sử phân tích                   | `cmd/worker-mock`, `agent_analysis_history`               | §4, §4.5       | `mock_metrics` tăng mỗi phút; dry-run 1 cycle     |
| **3** | **Tools**                 | 9 tool nội bộ: GetMetrics → UpdateRouting / SetMaintenance; ngưỡng per product; email ops | `internal/tools`, `internal/threshold`, `internal/notify` | §6, §7.4, §8.9 | Unit test từng tool + fixture §10                 |
| **4** | **Agent core**            | Loop catalog theo `routing_mode`; pipeline thu thập → suy luận → hành động                | `cmd/worker-agent`, `internal/agent`                      | §3, §5         | Incident + `health_status` xuất hiện sau 1 chu kỳ |
| **5** | **Reasoning + Output**    | Rules 1–9; sinh Health 🟢🟡🔴, Incident, Routing Plan, Maintenance; `agent_change_log`    | `internal/rules`, output §8                               | §7, §8         | Kịch bản A–D, H (routing SKU & provider)          |
| **6** | **REST API**              | REST + `dashboard/overview` (snapshot metric, refresh pending plan, auto đề xuất), incidents phân trang, SSE     | `cmd/api`                                                 | §2.3           | `curl /dashboard/overview` + `/incidents?page=1` |
| **7** | **Frontend React**        | Dashboard §9.0 (5 metric live, plan/bảo trì con, incidents phân trang), Settings, Chat + Voice | `web/` (Vite + React)                                     | §9             | F5 Dashboard — SKU đỏ có hàng đề xuất; incidents Trước/Sau |
| **8** | **E2E & DoD**             | Approve routing, email khi vượt ngưỡng; demo end-to-end không production        | Toàn hệ thống                                             | §10, §12       | `make test` pass; đủ checklist §12                |


**Ba binary chạy nền khi demo:** `worker-mock` (1 phút) · `worker-agent` (5 phút) · `api` (REST + feed).

---

## Cách đọc tài liệu


| Đối tượng                | Đọc theo thứ tự                                                                     |
| ------------------------ | ----------------------------------------------------------------------------------- |
| **Product / Ops**        | §1 Business → §8 Output (icon 🟢🟡🔴) → §10 Kịch bản test |
| **Developer Go**         | §2.1–2.4 Blueprint → **§13 MySQL** → §4–7 Agent → §6 Tools                          |
| **Developer React**      | §2.3 API → **§2.3.2 health Dashboard** → §9 UI → §10 kịch bản E/F |
| **AI Agent ( codegen )** | Khối `PROJECT` (build_phases) → §11 checklist → §2.2 repo → §13 DDL → §14 Tool Contracts → §7.3 Rules YAML → §7.6 Prompt → §3.1 State Machine → §15 Deployment |


### Bản đồ nội dung (Parts)


| Part                      | Mục      | Nội dung                                                           |
| ------------------------- | -------- | ------------------------------------------------------------------ |
| **I — Business**          | §1       | Bài toán, routing rules, catalog, **ngưỡng per product**           |
| **II — Architecture**     | §2, §3   | Kiến trúc 6 lớp, pipeline runtime, **state machine Agent**         |
| **II-b — Implementation** | §2.1–2.6 | **Go + React + MySQL**, repo, API, build order, vi-VN, **O365 auth** |
| **III — Agent backend**   | §4–8     | Scheduler, mock, agent, tools, output, email ops, **prompt LLM** |
| **IV — Frontend**         | §9       | React UI chat/voice/config/responsive                              |
| **V — Verification**      | §10–12   | E2E scenarios, phases, Definition of Done                          |
| **VI — Data**             | §13      | MySQL DDL, seed, query mẫu                                         |
| **VII — Tool Contracts**  | §14      | JSON Schema 9 tools (input/output) — AI codegen / MCP              |
| **VIII — Deployment**     | §15      | Docker, Vercel/Railway, Nginx, CI/CD GitHub Actions                |


### Metadata (machine-readable)

```yaml
project:
  name: opsone
  display_name: OpsOne
  tagline: "Ops + One-man team"
  version: "1.0"
stack:
  database: mysql-8.0
  backend: go-1.21
  frontend: react-18-vite
  locale: vi-VN
  llm: openai-compatible  # optional; rules engine chạy trước LLM
binaries:
  - cmd/api          # REST + WebSocket feed
  - cmd/worker-mock  # mock data mỗi 1 phút
  - cmd/worker-agent # phân tích theo scheduler_interval
build_phases:        # khớp §11 + bảng "Các bước phát triển chính"
  - id: 0
    name: setup
    deliverable: [db/schema.sql, db/seed.sql, internal/store]
  - id: 1
    name: mock_scheduler
    deliverable: [cmd/worker-mock, agent_analysis_history]
  - id: 2
    name: tools
    deliverable: [internal/tools, internal/threshold, internal/notify]
  - id: 3
    name: agent_core
    deliverable: [cmd/worker-agent, internal/agent]
  - id: 4
    name: reasoning_output
    deliverable: [internal/rules, internal/health, agent_change_log]
  - id: 5
    name: rest_api
    deliverable: [cmd/api, SSE /events, internal/auth]
  - id: 6
    name: frontend
    deliverable: [web/]
  - id: 7
    name: e2e_dod
    deliverable: [scenarios §10, make test, §12]
domain:
  products: 11
  providers: [ESALE, IMEDIA, SHOPPAY]
  routing:
    sku_mode: [card, topup_data]
    provider_mode: [topup]
  topup_data_skus: [VNP20, VNP50, V50K, V100K]
jobs:
  mock_interval: 1m
  agent_interval_default: 5m
  maintenance_default_duration: 60m
outputs: [health_status, incident, recommendation, maintenance, routing_plan, agent_change_log, ops_email]
notifications:
  channels: [email]
  trigger_after: [threshold_breach, routing_applied, maintenance_active]
  alert_when: [health_red, high_pending, high_fail, fail_txn_count, open_incident]
health_status: [green, yellow, red]
data_source: [mock, production]
alert_thresholds:
  scope: per_product
  metrics: [success_rate_min_pct, pending_rate_max_pct, fail_rate_max_pct, fail_txn_count_max, error_event_count_max]
```

---

# Part I — Business & Domain

## 1. Bài toán cần giải

Dashboard vận hành thường chỉ cho biết **điều gì đang xảy ra** (snapshot).

**OpsOne** phải làm thêm:


| Khả năng   | Mô tả                                                                                                        |
| ---------- | ------------------------------------------------------------------------------------------------------------ |
| Phát hiện  | Tự nhận biết bất thường theo chu kỳ                                                                          |
| Phân tích  | Nguyên nhân (metric, lỗi, routing)                                                                           |
| Trạng thái | **3 icon màu** — xanh (OK) / vàng (nghi vấn) / đỏ (có vấn đề) — sau mỗi lần phân tích                        |
| Đánh giá   | Tác động kinh doanh (revenue)                                                                                |
| Hành động  | Đề xuất hoặc thực thi điều phối traffic                                                                      |
| Giao tiếp  | UI **chat** hoặc **giọng nói (micro)** — responsive trên **PC và điện thoại**                                |
| Thông báo  | **Email** team vận hành sau routing/bảo trì khi 🔴 / pending / lỗi — kèm hướng dẫn nhóm chat provider (§8.9) |
| Chạy thử   | **Mock Data Generator** — tự sinh data giả **mỗi phút** để Agent phân tích khi chưa nối production           |


**Quy tắc chia tải theo loại dịch vụ (bắt buộc):**


| Loại dịch vụ   | `service_type` | Sản phẩm                                   | Cách chia tải (UpdateRouting)                                      | Không áp dụng                        |
| -------------- | -------------- | ------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------ |
| **Thẻ**        | `card`         | ZING, GARENA, VINAPHONE, MOBIFONE, VIETTEL | **Chỉ theo SKU / mệnh giá** (10.000đ, 20.000đ, …)                  | ~~Chia tải aggregate theo provider~~ |
| **Topup data** | `topup_data`   | DATA_VINA, DATA_MOBI, DATA_VIETTEL         | **Chỉ theo SKU / gói** (VNP20, VNP50, V50K, V100K) — **giống thẻ** | ~~Chia tải aggregate theo provider~~ |
| **Topup**      | `topup`        | TOPUP_VINA, TOPUP_MOBI, TOPUP_VIETTEL      | **Chỉ theo provider** (ESALE, IMEDIA, SHOPPAY)                     | ~~Chia tải theo SKU~~                |


**Gom nhóm triển khai (`routing_mode`):**


| `routing_mode` | `service_type`       | UpdateRouting scope    |
| -------------- | -------------------- | ---------------------- |
| **sku**        | `card`, `topup_data` | `sku` hoặc `sku_batch` |
| **provider**   | `topup`              | `provider`             |


**Giải thích:**

- **Thẻ / Topup data:** Mỗi SKU (mệnh giá hoặc gói data) là một luồng routing riêng — VD SKU 20.000đ hoặc gói `VNP50`: ESALE 10%  IMEDIA 90%. Agent vẫn **đọc metric theo provider** trong từng SKU, nhưng **không** gọi UpdateRouting ở mức cả product.
- **Topup (nạp tiền):** Một bảng routing cho cả product — VD TOPUP_VINA: ESALE 70%, IMEDIA 20%, SHOPPAY 10%. Không dùng GetSkus / routing theo gói.

### 1.1 Danh sách loại dịch vụ & sản phẩm (catalog)

Agent quét **toàn bộ product** trong catalog mỗi chu kỳ (hoặc subset pilot nếu rollout từng phần).


| `service_type`   | `routing_mode` | Sản phẩm (`product`) | Ghi chú                                               |
| ---------------- | -------------- | -------------------- | ----------------------------------------------------- |
| `**card`**       | `sku`          | **ZING**             | Thẻ game                                              |
|                  |                | **GARENA**           | Thẻ game                                              |
|                  |                | **VINAPHONE**        | Thẻ telco                                             |
|                  |                | **MOBIFONE**         | Thẻ telco                                             |
|                  |                | **VIETTEL**          | Thẻ telco                                             |
| `**topup_data`** | `sku`          | **DATA_VINA**        | Topup data Vinaphone — SKU: VNP20, VNP50, V50K, V100K |
|                  |                | **DATA_MOBI**        | Topup data Mobifone — SKU: VNP20, VNP50, V50K, V100K  |
|                  |                | **DATA_VIETTEL**     | Topup data Viettel — SKU: VNP20, VNP50, V50K, V100K   |
| `**topup`**      | `provider`     | **TOPUP_VINA**       | Nạp tiền Vinaphone                                    |
|                  |                | **TOPUP_MOBI**       | Nạp tiền Mobifone                                     |
|                  |                | **TOPUP_VIETTEL**    | Nạp tiền Viettel                                      |


**Tổng:** **11 product** trong catalog. **Kịch bản bảo trì (§10.2, §10.2b):** **ZING** chỉ **1 provider active** (ESALE) — dù catalog có thể còn provider inactive → nhánh maintenance, không chuyển traffic sang provider khác.

**Config catalog mẫu (backend / UI):**

```json
{
  "products": [
    { "product": "ZING",       "service_type": "card",       "routing_mode": "sku",      "label": "Thẻ Zing" },
    { "product": "GARENA",     "service_type": "card",       "routing_mode": "sku",      "label": "Thẻ Garena" },
    { "product": "VINAPHONE",  "service_type": "card",       "routing_mode": "sku",      "label": "Thẻ Vinaphone" },
    { "product": "MOBIFONE",   "service_type": "card",       "routing_mode": "sku",      "label": "Thẻ Mobifone" },
    { "product": "VIETTEL",    "service_type": "card",       "routing_mode": "sku",      "label": "Thẻ Viettel" },
    { "product": "DATA_VINA",    "service_type": "topup_data", "routing_mode": "sku",      "label": "Data Vinaphone" },
    { "product": "DATA_MOBI",    "service_type": "topup_data", "routing_mode": "sku",      "label": "Data Mobifone" },
    { "product": "DATA_VIETTEL", "service_type": "topup_data", "routing_mode": "sku",      "label": "Data Viettel" },
    { "product": "TOPUP_VINA",    "service_type": "topup",      "routing_mode": "provider", "label": "Topup Vinaphone" },
    { "product": "TOPUP_MOBI",    "service_type": "topup",      "routing_mode": "provider", "label": "Topup Mobifone" },
    { "product": "TOPUP_VIETTEL", "service_type": "topup",      "routing_mode": "provider", "label": "Topup Viettel" }
  ]
}
```

**SKU gói topup data** (dùng chung cho DATA_VINA / DATA_MOBI / DATA_VIETTEL): `VNP20`, `VNP50`, `V50K`, `V100K`.

**Checklist catalog:**

- Seed 11 product vào bảng `products` + SKU + provider (§13.2, §13.10).
- UI (tuỳ chọn): bảng danh sách product — bật/tắt từng sản phẩm trong pilot.
- Scheduler loop: `foreach product in catalog WHERE enabled`.

**Single-provider active (maintenance):** Product có **chỉ 1 provider active** (`product_providers.enabled = 1`) — VD **ZING → chỉ ESALE active** (kịch bản B) hoặc **2 provider trong catalog nhưng 1 inactive** → nhánh **maintenance**, không `UpdateRouting`. Các kịch bản routing SKU (§10.3) dùng ZING đủ 3 provider **active** (seed mặc định).

### 1.2 Ngưỡng cảnh báo & hành động theo dịch vụ

Mỗi **product** (dịch vụ) có bộ **ngưỡng riêng** trong `product_alert_thresholds` (§13.2.1). Khi metric **vượt ngưỡng** → OpsOne **đồng thời**:

1. **Quyết định hành động** — routing (≥ 2 provider **active**) hoặc bảo trì (chỉ 1 provider **active**) theo rules + `routing_scope_state.auto_action` **hiệu lực** per scope (§9.5.2 — ưu tiên cấu hình **dịch vụ** `sku_code=""` trước, fallback **SKU**) — xem §7.4 `suggested_action`.
2. **Gửi email** team vận hành (§8.9) — nếu `alert_email_enabled` và vượt ngưỡng.

**Hai loại ngưỡng (OR — chỉ cần một vượt):**


| Loại            | Trường config           | Ví dụ   | Ý nghĩa                                   |
| --------------- | ----------------------- | ------- | ----------------------------------------- |
| **Tỷ lệ %**     | `success_rate_min_pct`  | `< 80%` | Success quá thấp                          |
|                 | `pending_rate_max_pct`  | `> 15%` | Pending cao                               |
|                 | `fail_rate_max_pct`     | `> 10%` | Fail cao                                  |
| **Số lượng GD** | `fail_txn_count_max`    | `> 50` (mặc định) | Số giao dịch **lỗi** trong cửa sổ         |
|                 | `error_event_count_max` | `> 50`  | Tổng sự kiện lỗi tại **cùng snapshot** metric (§2.3.2) |
|                 | `pending_txn_count_max` | `> 5` (mặc định) | Số GD **pending** trong cửa sổ            |

> **Dashboard UI (§2.3.2, §9.0):** So sánh **per provider** (`provider_metrics`) với toán tử `≤` / `≥` — bất kỳ provider active vi phạm 1/5 chỉ số → đỏ + đề xuất routing/bảo trì. Agent chu kỳ (`ScopeBreached`, `ScopeBreachedFromRates`) dùng `<` / `>` strict; Dashboard dùng `ScopeBreachedFromSnapshot` (≤/≥) khớp màu chữ UI.

**Chế độ cảnh báo:** Khi **bất kỳ** điều kiện trên vượt ngưỡng (OR) → `EvaluateThresholds` trả `should_alert_mode=true` (health 🟡/🔴, UI tab/icon). Nếu `alert_email_enabled=1` → thêm `should_alert_email=true` và gửi mail §8.9.


**Cửa sổ đo:** `metrics_window_min` (mặc định **15 phút**) — `GetMetrics` lấy **dòng mới nhất** trong cửa sổ; mọi đếm GD lỗi / pending / error event phải **cùng timestamp** snapshot đó (không cộng dồn cả cửa sổ khi metric là snapshot 1 phút — §2.3.2).

**Fallback:** cột `NULL` trên product → dùng **ngưỡng mặc định toàn hệ thống** trong `agent_settings` (§13.4).

**Seed mẫu (11 product):**


| Product    | Success min | Pending max | Fail max | Fail GD max | Pending GD max | Chu kỳ liên tiếp |
| ---------- | ----------- | ----------- | -------- | ----------- | -------------- | ---------------- |
| (mặc định) | 80%         | 15%         | 10%      | **50**      | **5**          | 2                |
| TOPUP_VINA | 80%         | 15%         | 10%      | 120         | 5              | 2                |
| ZING       | 85%         | 12%         | 8%       | 80          | 10             | 2                |
| DATA_VINA  | 82%         | 14%         | 10%      | 60          | 8              | 1                |


**Luồng đánh giá (mỗi chu kỳ, per product):**

```text
Load product_alert_thresholds
  → GetProviders(product) — đếm provider **active** (§6.3)
  → GetMetrics (latest row in window) + fail/pending txn + error_event **cùng recorded_at**
  → EvaluateThresholds() — internal/threshold §7.4
  → IF breached AND consecutive_cycles >= required
       → Rules §7.3 + health 🔴
       → IF active_provider_count > 1 AND có backup healthy → Routing Plan / UpdateRouting
       → IF active_provider_count == 1 (hoặc không có backup healthy) → Maintenance / SetMaintenance
       → IF alert_email_enabled → email §8.9 (kèm ngưỡng nào bị vượt)
```

---

# Part II — Architecture & Runtime

## 2. Kiến trúc 6 lớp

```text
0. Mock Data Generator (1 min)          ← demo / chạy thử (bật khi data_source=mock)
        ↓ ghi bảng mock_metrics
1. Scheduler (5 min)                    ← chủ động, không cần user
        ↓
2. OpsOne (Agent core)
        ↓
3. Tools  ← đọc MySQL (mock / production) hoặc API ngoài
   ├─ GetMetrics
   ├─ GetTopErrors
   ├─ GetProviders
   ├─ GetSkus
   ├─ GetRouting
   ├─ GetRevenue
   ├─ UpdateRouting
   ├─ GetMaintenance          ← cửa sổ bảo trì đang/ sắp diễn ra
   └─ SetMaintenance          ← lên lịch bảo trì có starts_at / ends_at
        ↓
4. Reasoning Engine
        ↓
5. Output
   ├─ Health Status (🟢 / 🟡 / 🔴)
   ├─ Incident
   ├─ Recommendation
   ├─ Maintenance
   └─ Routing Plan
        ↓
6. UI giao tiếp (responsive)
   ├─ Chat (nhập text)
   ├─ Voice (micro → speech-to-text)
   └─ Hiển thị feed Incident / Routing Plan / **icon trạng thái**
        ↔  (song song) User hỏi / ra lệnh bất kỳ lúc nào
```

> **Lưu ý:** Scheduler chạy theo **chu kỳ cấu hình UI** (mặc định 5 phút). **Mock Generator** sinh data **1 phút/lần** khi `data_source=mock`. UI Cấu hình quyết định **auto routing**.

**Tầm nhìn sản phẩm:**

> Hầu hết dashboard hiện nay chỉ cho biết "điều gì đang xảy ra". Agent tiến thêm một bước: tự phát hiện bất thường, tự phân tích nguyên nhân, đánh giá tác động kinh doanh và chủ động đề xuất hoặc thực thi phương án điều phối traffic tối ưu theo thời gian thực.

**Luồng triển khai:**

```text
[0] Mock Data Generator (1 phút/lần)     ← chỉ khi chế độ mock
         ↓
     mock_metrics / mock_error_stats (MySQL)
         ↓
[1] Scheduler (chu kỳ cấu hình UI, mặc định 5 phút)
         ↓
[2] OpsOne (Agent core)
         ↓
[3] Tools — data_source=mock → MySQL §13.5; production → metrics_snapshot / API
         ↓
[4] Reasoning Engine (LLM + rules)
         ↓
[5] Output
         ↓
     (tùy chọn) UpdateRouting — đề xuất hoặc thực thi
         ↓
     Lưu agent_analysis_history → chu kỳ sau
         ↓
     Đẩy Output lên UI (feed / push) — user xem trên PC hoặc điện thoại
```

**Luồng UI (song song với scheduler):**

```text
User mở UI (PC / mobile)
    ↓
Chat text  hoặc  Bật Mic (toggle) / nói **alo** (wake) → nói lệnh → im lặng 2s → tự gửi
              ↓
Sau đó `POST /chat` (Agent tự cập nhật tên/avatar nếu user nói "Tên tôi là Khiêm")
              ↓
Speech-to-text (nếu voice) → *đóng chat* (thu gọn) · *tắt mic* / *bye bye* (thoát)
    ↓
Agent trả lời / gọi tool on-demand (VD: giải thích incident, xem routing)
    ↓
Hiển thị trong khung chat + cập nhật panel Incident / Routing Plan
```

---

## 2.1 Tech stack


| Layer           | Công nghệ                     | Ghi chú                                                       |
| --------------- | ----------------------------- | ------------------------------------------------------------- |
| **Database**    | MySQL 8, InnoDB, utf8mb4      | Schema **§13**; dùng `database/sql` + prepared statements     |
| **Backend**     | Go 1.21                       | Monorepo; goroutine + `time.Ticker` cho scheduler             |
| **HTTP API**    | `net/http` hoặc `chi` / `gin` | REST JSON; CORS cho React dev                                 |
| **Workers**     | Go binaries riêng             | `worker-mock` (1m), `worker-agent` (5m configurable)          |
| **Reasoning**   | Go rules engine + LLM API     | Rules §7 hard-code trước; LLM tóm tắt + chat                  |
| **Frontend**    | React 18 + Vite + TypeScript  | Responsive; Web Speech API cho voice                          |
| **Realtime UI** | SSE hoặc WebSocket            | Push incident / health_status sau mỗi chu kỳ                  |
| **Ngôn ngữ**    | **Tiếng Việt** (`vi-VN`)      | Mọi thông báo Agent: UI, chat, email, incident — xem **§2.5** |


**Biến môi trường (`.env`):**

```bash
MYSQL_DSN=app:secret@tcp(localhost:3306)/opsone?parseTime=true&charset=utf8mb4&loc=Asia%2FHo_Chi_Minh
DATA_SOURCE=mock                    # mock | production
LLM_API_URL=                        # optional; mặc định GreenNode AIP §7.6.7
LLM_API_KEY=                        # bật chat LLM khi set (alias AIP_API_KEY)
LLM_MODEL=minimax/minimax-m2.5      # id từ GET .../v1/models (phân biệt hoa/thường)
LLM_TIMEOUT_SECONDS=30
API_ADDR=:8080
CORS_ORIGIN=http://localhost:5173
# Email (§8.9) — hackathon: MailHog localhost:1025
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USER=
SMTP_PASS=
SMTP_FROM=opsone@company.local
NOTIFICATION_MOCK=false              # true = chỉ ghi notification_log, không gửi SMTP
AGENT_LOCALE=vi-VN                   # ngôn ngữ output Agent §2.5
```

---

## 2.5 Ngôn ngữ Agent — tiếng Việt (bắt buộc)

**OpsOne** giao tiếp với team vận hành bằng **tiếng Việt**. Code/DB enum giữ tiếng Anh (`high`, `open`); **mọi text hiển thị hoặc gửi cho người** phải là tiếng Việt.


| Kênh                                        | Yêu cầu                                                                      |
| ------------------------------------------- | ---------------------------------------------------------------------------- |
| **UI**                                      | Nhãn, feed, incident card, routing plan, settings — tiếng Việt               |
| **Chat / Voice**                            | Agent trả lời tiếng Việt; STT ưu tiên `vi-VN`                                |
| **Email** (§8.9)                            | Subject + body tiếng Việt                                                    |
| **Incident / Recommendation / Maintenance** | `summary`, `action_detail` tiếng Việt                                        |
| **LLM**                                     | System prompt: *"Luôn trả lời bằng tiếng Việt, văn phong ngắn gọn cho ops."* |


**Bảng dịch enum → nhãn UI / email** (`internal/i18n` hoặc `web/src/i18n/vi.ts`):


| Enum (DB/API)                   | Nhãn tiếng Việt       |
| ------------------------------- | --------------------- |
| `health_status: green`          | Hệ thống OK           |
| `health_status: yellow`         | Đang theo dõi / xử lý |
| `health_status: red`            | Đang có vấn đề        |
| `severity: low`                 | Thấp                  |
| `severity: medium`              | Trung bình            |
| `severity: high`                | Cao                   |
| `incident.status: open`         | Đang mở               |
| `incident.status: acknowledged` | Đã từ chối (admin từ chối routing plan) |
| `incident.status: resolved`     | Đã xử lý (duyệt routing hoặc auto) |
| `incident.resolution_action: admin_approve` | Duyệt routing |
| `incident.resolution_action: admin_reject`  | Từ chối routing |
| `incident.resolution_action: auto`          | Tự động (agent apply routing) |
| `maintenance: pending_approve`  | Chờ duyệt bảo trì     |
| `maintenance: scheduled`        | Đã lên lịch           |
| `maintenance: active`           | Đang bảo trì          |
| `maintenance: completed`        | Hoàn tất bảo trì      |


**Trường JSON user-facing** (luôn tiếng Việt):

- `health_summary`, `health_label`
- `incident.summary`, `recommendations.action_detail`
- `notification_log.subject`, `action_summary`
- `breach_reasons[]` — VD: `"Tỷ lệ lỗi 12% vượt ngưỡng 10%"`

**Không dùng** trong text gửi ops: `Maintenance Recommended`, `Shift traffic`, `Monitor for`, `Severity: High`, `Routing Plan pending`, `degrading` — thay bằng bản Việt tương ứng (VD: *Đề xuất bảo trì*, *Chuyển traffic*, *Theo dõi thêm*, *Mức Cao*, *Kế hoạch routing chờ duyệt*, *suy giảm*).

**Checklist:**

- `agent_settings.agent_locale` = `vi-VN` (§13.4).
- Template email `internal/notify/template_vi.go` — không hard-code English.
- Unit test: snapshot subject/body email chỉ chứa tiếng Việt (trừ mã kỹ thuật: product_code, ESALE).

---

## 2.2 Cấu trúc repository (monorepo)

```text
opsone/
├── cmd/
│   ├── api/                 # HTTP server — REST + SSE
│   │   └── main.go
│   ├── worker-mock/         # Cron 1 phút — §4.5
│   │   └── main.go
│   └── worker-agent/        # Cron N phút — §3 pipeline
│       └── main.go
├── internal/
│   ├── config/              # env, agent_settings từ MySQL
│   ├── domain/              # Product, SKU, RoutingMode enums
│   ├── store/               # MySQL repositories (§13)
│   │   ├── products.go
│   │   ├── metrics.go       # mock_metrics | metrics_snapshot
│   │   ├── routing.go
│   │   ├── cycles.go        # agent_analysis_*
│   │   ├── routing_update.go # UpdateRouting + agent_change_log audit §8.7
│   │   └── output.go        # incidents, routing_plans
│   ├── mock/                # Generator + scenarios §4.5.3
│   ├── tools/               # GetMetrics, GetRouting, … §6
│   ├── maintenance/         # Lifecycle cửa sổ bảo trì §8.5
│   ├── notify/              # Email ops + template tiếng Việt §8.9
│   ├── i18n/                # Dịch enum → nhãn vi-VN §2.5
│   ├── agent/               # Orchestrator loop §5
│   ├── rules/               # Rules 1–9 §7.3
│   ├── threshold/           # Đánh giá ngưỡng per product §1.2 / §7.5
│   ├── reasoning/           # LLM client + prompt builder
│   ├── health/              # health_status green/yellow/red §8.2
│   ├── api/                 # REST + SSE §2.3; chat §7.6.5 — chat_agent, chat_metrics, chat_maintenance, chat_commands, chat_turn, …
│   ├── chatresolve/         # Alias + intent metrics/BT/SKU §7.6.5 (`intent.go`, `sku.go`, `aliases.go`)
│   ├── healthserver/        # GET /health :8080 — probe AgentBase trước khi nối MySQL (api/workers)
│   └── llm/                 # OpenAI-compatible MaaS client §7.6.7 (GreenNode AIP)
├── Dockerfile               # api — GreenNode AgentBase
├── Dockerfile.worker-mock
├── Dockerfile.worker-agent
├── db/
│   ├── schema.sql           # copy từ §13
│   └── seed.sql             # UTF-8 — nhãn tiếng Việt (Thẻ, đ); xem §13.11
├── scripts/
│   ├── dev.ps1              # Invoke-OpsOneReset, Start-OpsOneAPI/Web, … (Windows)
│   ├── run-api.ps1          # nạp `.env`, giải phóng port `API_ADDR`, `go run ./cmd/api`
│   ├── prepare-greennode-env.ps1
│   ├── deploy-greennode.ps1   # TARGET: api | worker-mock | worker-agent | web
│   └── deploy-greennode-all.ps1
├── .env.greennode           # env container runtime (MYSQL_DSN, LLM_*, …) — không commit secret
├── web/                     # React app (Phase 6 ✅)
│   ├── Dockerfile           # nginx :8080 — GreenNode `opsone-web`
│   ├── nginx.conf
│   ├── public/              # favicon ZaloPay (favicon.png, apple-touch-icon.png)
│   ├── dev.ps1              # npm run dev — fix PATH Node trên Windows
│   ├── src/
│   │   ├── api/client.ts    # fetch + DEV_AUTH_BYPASS / MSAL
│   │   ├── auth/msalConfig.ts
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx       # bảng routing §9.0
│   │   │   ├── IncidentsPage.tsx   # tab Sự cố — incidents phân trang
│   │   │   ├── Settings.tsx        # §9.5 scheduler + bảo trì mặc định + mock (compact card; ngưỡng/auto trên Dashboard)
│   │   │   └── MaintenancePage.tsx
│   │   ├── components/
│   │   │   ├── Layout.tsx              # top-nav + logo ZaloPay + OpsOne + SSE + ChatWidget
│   │   │   ├── ChatWidget.tsx          # chat nổi — onboarding local + dock 4 góc (useChatDock)
│   │   │   ├── ChatAvatar.tsx          # avatar user (sprite 7×4) + OpsOne (favicon)
│   │   │   ├── HealthBadge.tsx         # 🟢🟡🔴 (+ compact icon-only)
│   │   │   ├── ServiceOverviewTable.tsx # routing + bảo trì + cột provider (6 chỉ số) + hàng plan con + ngưỡng đầu nhóm
│   │   │   ├── ProductThresholdEditor.tsx # ngưỡng per product — hàng Ngưỡng cảnh báo (§9.5.3)
│   │   │   ├── ProviderMetricCell.tsx   # 6 dòng metric/provider; inactive (routing 0%) + Mở lại; maintained → disabled + metric 0
│   │   │   ├── RoutingPctEditor.tsx    # nhập % khi duyệt plan (hàng con, căn cột provider); sync draft khi API đổi
│   │   │   ├── IncidentsTable.tsx      # bảng sự cố full-width + phân trang (§9.0)
│   │   │   ├── ScopeAutoEditor.tsx     # Chế độ BT / Routing — compact + ⋯ (cột SKU + cột Dịch vụ §9.5.2)
│   │   │   ├── ActiveMaintenanceCell.tsx # BT active — Mở lại dịch vụ + Gia hạn (Từ/Đến, §9.0.5)
│   │   │   ├── ServiceMaintenanceButton.tsx # BT thủ công — nút「Bảo trì dịch vụ」cột Bảo trì + cột Dịch vụ (§9.0.6–7)
│   │   │   ├── ProductMaintenanceActions.tsx # BT active cấp dịch vụ — Mở lại / Gia hạn batch khi mọi SKU BT (§9.0.7)
│   │   │   ├── SkuMaintenanceTimeLabel.tsx   # nhãn BT active 4 dòng dưới tên SKU (§9.0)
│   │   │   ├── RedSkuScrollNav.tsx           # ▲ 🔴 1/N ▼ — điều hướng nhanh tới SKU đỏ
│   │   │   └── MaintenanceCard.tsx
│   │   ├── utils/
│   │   │   ├── assistantIdentity.ts    # ASSISTANT_NAME = OpsOne
│   │   │   ├── chatIntro.ts            # CHAT_INTRO_MESSAGE — làm quen
│   │   │   ├── chatOnboarding.ts       # profileKnown, buildOnboardingReply, profile update
│   │   │   ├── chatUserProfile.ts      # parse tên/xưng hô/tuổi, avatar sprite, userBubbleLabel
│   │   │   ├── dashboardHealth.ts      # effectiveRowHealth + worst health + format legacy summary
│   │   │   ├── dashboardRowOrder.ts    # groupRowsByProduct, scopeDisplayLabel (toast)
│   │   │   ├── scopeAuto.ts            # ShouldAutoApplyScope — ẩn đề xuất khi auto
│   │   │   ├── maintenanceDisplay.ts   # isProviderUnderMaintenance — cột provider disabled khi BT active
│   │   │   ├── maintenanceWindow.ts    # formatDatetimeVi BT, cửa sổ mặc định 60p, maintenanceWindowUnchanged
│   │   │   ├── scopeMetrics.ts         # breach 5 ngưỡng + format S/P/F (%, GD)
│   │   │   └── incidentStatus.ts       # nhãn TT sự cố + format handled
│   │   └── hooks/
│   │       ├── useChatDock.ts          # chat dock 4 góc + localStorage
│   │       ├── useChatResize.ts        # resize panel 4 góc
│   │       ├── useOpsOneWake.ts        # wake "alo" → mở chat + mic (background STT)
│   │       ├── useSSE.ts               # /events + poll fallback
│   │       ├── useOverallHealth.ts     # header/logo badge 🟢🟡🔴 + health_summary
│   │       ├── useProductThresholds.ts # cache GET /products/{code}/thresholds
│   │       └── useVoiceInput.ts        # Web Speech vi-VN; mic liên tục; 2s silence → gửi; alo/đóng chat/tắt mic
│   └── package.json
├── docker-compose.yml       # mysql (+ TZ Asia/Ho_Chi_Minh)
└── Makefile                 # migrate, seed, run-all
```

**Package mapping (Go ↔ Spec):**


| Package                | Spec       | Trách nhiệm                                |
| ---------------------- | ---------- | ------------------------------------------ |
| `internal/store`       | §13        | CRUD MySQL; `history.go` (`ScopeBreachedFromSnapshot`, `SnapshotBreachReasons`, `FindOpenIncidentForScope`); `recommendations.go`; `ListIncidents` + `CountIncidents` |
| `internal/tools`       | §6         | 9 tool functions (7 cốt lõi + maintenance); `GetMetrics` trả `recorded_at` |
| `internal/agent`       | §3, §5     | Pipeline 7 bước mỗi chu kỳ                 |
| `internal/mock`        | §4.5       | Sinh data 1 phút → `mock_metrics`; skip scope đang bảo trì |
| `internal/rules`       | §7.3       | Rule engine deterministic                  |
| `internal/threshold`   | §1.2, §7.4 | So sánh metric vs ngưỡng per product       |
| `internal/catalog`     | §1.1, §8.2 | `scopes.go`, `display.go` — `ProductDisplayLabel`, `FormatCycleHealthSummary` |
| `internal/chatresolve` | §7.6.5     | Alias product, assistant identity (OpsOne / Zalopay cùng trợ lý) |
| `internal/i18n`        | §2.5       | Nhãn tiếng Việt cho enum API               |
| `internal/health`      | §8.2       | Tính `health_status`                       |
| `internal/maintenance` | §8.5       | Cửa sổ bảo trì starts_at / ends_at         |
| `internal/notify`      | §8.9       | Email team ops + leo thang provider chat   |
| `internal/store/routing_update` | §8.7 | Ghi `agent_change_log` khi UpdateRouting (audit DB) |
| `internal/api`         | §2.3       | HTTP handlers, SSE; `dashboard.go` + `dashboard_snapshot.go` + `dashboard_suggest.go` (live health, snapshot metric, refresh pending plan, đề xuất routing/bảo trì) |
| `cmd/api`              | §2.3, §9   | REST cho React; `DEV_AUTH_BYPASS` dev (§2.3.1)     |


---

## 2.3 REST API (contract cho React & test)

Base URL: `http://localhost:8080/api/v1`


| Method | Path                           | Handler Go             | Mô tả                                             |
| ------ | ------------------------------ | ---------------------- | ------------------------------------------------- |
| GET    | `/health`                      | `HealthLiveness`       | `{ "status": "ok" }` — probe                        |
| GET    | `/health-status`               | `HealthHandler`        | Global + per-product 🟢🟡🔴 + `product_label`     |
| GET    | `/dashboard/overview`          | `DashboardOverview`    | Bảng routing tất cả product/SKU + bảo trì + plan chờ §9.0 |
| GET    | `/config`                      | `ConfigGet`            | `agent_settings` — scheduler, mock (không auto routing global) |
| PUT    | `/config`                      | `ConfigPut`            | Admin; ghi `config_audit_log`                     |
| PUT    | `/scopes/{product}/auto`       | `ScopeAutoPut`         | Cấp **dịch vụ** (`sku_code=""`) — `auto_action`, `window_*`; áp dụng mọi SKU; nếu auto → `CancelPendingRoutingPlansForProduct` (§9.5.2) |
| PUT    | `/scopes/{product}/{sku}/auto`| `ScopeAutoPut`        | Cấp **SKU** (fallback khi chưa cấu hình dịch vụ) — `auto_action`, `window_start`, `window_end` |
| POST   | `/scopes/{product}/{sku}/routing/approve` | `ScopeRoutingApprove` | Duyệt đề xuất routing (synthetic `id=0` hoặc admin confirm) — body `{ "proposed_pct": {...}, "plan": {...} }` → `UpdateRouting` |
| POST   | `/scopes/{product}/{sku}/routing/apply`   | `ScopeRoutingApply`   | Admin cập nhật routing thủ công tùy chỉnh — body `{ "proposed_pct": {...} }` → `UpdateRouting`, `trigger_type=manual_temp` (bảng `agent_change_log`; **không** dùng `admin_manual` — enum đó chỉ cho `maintenance_windows`) |
| POST   | `/scopes/{product}/{sku}/routing/restore-baseline` | `ScopeRoutingRestoreBaseline` | **Mở lại** provider — áp `baseline_pct`; metric tham chiếu từ **chu kỳ Agent gần nhất** (`agent_analysis_history`), không snapshot live (provider `routing_pct=0` → metric 0); ghi `recovery_apply_cycle_id` — poll/agent auto **chờ chu kỳ tiếp theo** |
| POST   | `/scopes/{product}/{sku}/maintenance/reopen-service` | `ScopeMaintenanceReopenService` | **Mở lại dịch vụ** — hủy BT active + `restore-baseline` **một request** (tránh race poll); cùng grace + metric chu kỳ Agent |
| POST   | `/scopes/{product}/{sku}/routing/reject`  | `ScopeRoutingReject`  | Từ chối đề xuất synthetic → ghi `routing_plans.status=rejected`; cho phép tạo đề xuất mới khi metric vẫn vi phạm |
| POST   | `/scopes/{product}/{sku}/maintenance/approve` | `ScopeMaintenanceApprove` | Duyệt đề xuất bảo trì synthetic — `SetMaintenance`, `trigger_type=admin_manual` |
| POST   | `/scopes/{product}/{sku}/maintenance/reject`  | `ScopeMaintenanceReject`  | Từ chối đề xuất synthetic → ghi `recommendations` `DISMISSED:…` (audit); poll tiếp theo có thể trả `pending_maintenance` mới nếu metric vẫn vi phạm |
| POST   | `/scopes/{product}/{sku}/maintenance/extend`  | `ScopeMaintenanceExtend`  | **Gia hạn bảo trì** — body `{ starts_at, ends_at }`; cập nhật mọi cửa sổ `scheduled`/`active` còn hiệu lực (`UpdateActiveMaintenanceTimesForSKU`). **400** `no_change` nếu thời gian không đổi nhưng vẫn còn BT (`CountActiveMaintenanceForSKU`); **404** nếu không còn BT active |
| GET    | `/incidents?page=&page_size=&since=` | `IncidentsList`        | Feed UI phân trang (mặc định `page=1`, `page_size=10`, max 50) |
| GET    | `/incidents/{incident_id}`     | `IncidentsGet`         | Chi tiết — **đăng ký cả** `/incidents/` (Go 1.21 ServeMux) |
| GET    | `/routing-plans/latest`        | `RoutingPlansLatest`   |                                                   |
| POST   | `/routing-plans/{id}/approve`  | `RoutingPlansApprove`  | Body `{ "proposed_pct": {...} }` → UpdateRouting; **đóng sự cố** liên quan → `resolved` + `handled_*` §8.3.1 |
| POST   | `/routing-plans/{id}/reject`   | `RoutingPlansReject`   | Plan `rejected`; **đóng sự cố** → `acknowledged` + `resolution_action=admin_reject` |
| POST   | `/recommendations/{id}/approve`| `RecommendationApprove`| Duyệt đề xuất bảo trì Agent — `SetMaintenance`, `trigger_type=admin_manual` |
| POST   | `/recommendations/{id}/reject` | `RecommendationReject` | Xóa recommendation; đóng sự cố `acknowledged` |
| GET    | `/maintenance`                 | `MaintenanceList`      | Danh sách cửa sổ bảo trì (filter status, product) |
| GET    | `/maintenance/{id}`            | `MaintenanceGet`       | Chi tiết starts_at / ends_at                      |
| POST   | `/maintenance`                 | `MaintenanceCreate`    | Admin lên lịch thủ công                           |
| POST   | `/maintenance/{id}/approve`    | `MaintenanceApprove`   | Duyệt đề xuất OpsOne → scheduled/active           |
| POST   | `/maintenance/{id}/cancel`     | `MaintenanceCancel`    | Hủy trước khi/khi đang bảo trì                    |
| GET    | `/notifications`               | `NotificationsList`    | Lịch sử email đã gửi §8.9                         |
| POST   | `/notifications/test`          | `NotificationsTest`    | Admin gửi mail thử                                |
| GET    | `/escalation-chat`             | `EscalationChatList`   | Cấu hình app/nhóm/tag theo provider               |
| PUT    | `/escalation-chat`             | `EscalationChatPut`    | Admin cập nhật leo thang chat                     |
| POST   | `/chat`                        | `ChatPost`             | `{ "message", "session_id?", "user_display_name?", "input_source?", "stt_raw?" }` → `{ "reply", "session_id" }`; LLM agent §7.6.5; log `chat_interaction_log` §7.6.5.5; **502** `llm_error` nếu MaaS lỗi |
| GET    | `/chat/sessions/{id}`          | `ChatHistory`          |                                                   |
| GET    | `/products`                    | `ProductsList`         | Catalog §1.1 + ngưỡng tóm tắt                     |
| GET    | `/products/{code}/routing`     | `ProductRoutingGet`    | Baseline + traffic + `pending_restore` per scope/SKU §8.6.5 |
| PUT    | `/products/{code}/routing`     | `ProductRoutingPut`    | Admin; body `set_as_baseline` + `routing` §8.6.5              |
| GET    | `/products/{code}/thresholds`  | `ProductThresholdsGet` | Ngưỡng cảnh báo §1.2                              |
| PUT    | `/products/{code}/thresholds`  | `ProductThresholdsPut` | Admin cập nhật ngưỡng                             |
| GET    | `/alert-thresholds`            | `AlertThresholdsList`  | Bảng ngưỡng toàn catalog                          |
| GET    | `/metrics`                     | `MetricsQuery`         | Query params: product, provider, sku, window      |
| GET    | `/mock/status`                 | `MockStatus`           |                                                   |
| POST   | `/mock/generate`               | `MockGenerateOnce`     | Debug                                             |
| GET    | `/events`                      | SSE                    | Stream cycle finished + health                    |


**Request / Response `POST /scopes/{product}/{sku}/routing/apply` (mở lại provider):**

```json
// Request
{ "proposed_pct": { "ESALE": 30, "IMEDIA": 70, "SHOPPAY": 0 } }

// Response 200
{ "applied": true, "change_log_ids": [4] }
```

- Validate giống approve: mỗi provider **0–100%**, **Σ = 100%**; scope khớp `routing_mode` (SKU hoặc topup provider).
- Ghi `agent_change_log` với `trigger_type=manual_temp`, `executed_by=admin`; **không** đóng incident (chỉ `admin_approve` / `auto` mới resolve §8.3.1).
- Handler: `handleScopeRoutingApply` (`handlers_scope_actions.go`); route `POST` action `apply` trong `routeScopes`.

**Response `GET /health-status` (mẫu):**

```json
{
  "cycle_started": "2026-06-04T10:10:00+07:00",
  "health_status": "yellow",
  "health_label": "Đang theo dõi / xử lý",
  "health_summary": "TOPUP_VINA ESALE suy giảm — Kế hoạch routing chờ duyệt",
  "products": [
    { "product_code": "TOPUP_VINA", "product_label": "Topup Vinaphone", "health_status": "yellow", "health_summary": "..." },
    { "product_code": "ZING", "product_label": "Thẻ Zing", "health_status": "green", "health_summary": "Ổn định" }
  ]
}
```

**Response `GET /dashboard/overview` (mẫu — §9.0):**

```json
{
  "updated_at": "2026-06-09T10:49:30+07:00",
  "providers": ["ESALE", "IMEDIA", "SHOPPAY"],
  "rows": [
    {
      "product_code": "TOPUP_VINA",
      "product_label": "Topup Vinaphone",
      "service_type": "topup",
      "sku_code": "",
      "health_status": "green",
      "routing_pct": { "ESALE": 70, "IMEDIA": 20, "SHOPPAY": 10 }
    },
    {
      "product_code": "DATA_VINA",
      "product_label": "Data Vinaphone",
      "service_type": "topup_data",
      "sku_code": "V100K",
      "health_status": "red",
      "routing_pct": { "ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20 },
      "live_metrics": {
        "success_pct": 97.2,
        "pending_pct": 0.8,
        "fail_pct": 2.0,
        "pending_txn": 32,
        "fail_txn": 63
      },
      "pending_plan": {
        "id": 12,
        "status": "pending_approve",
        "plan": { "proposed_pct": { "ESALE": 45.9, "IMEDIA": 36.1, "SHOPPAY": 18 }, "reason_vi": "..." }
      }
    },
    {
      "product_code": "DATA_VINA",
      "sku_code": "V50K",
      "health_status": "green",
      "routing_pct": { "ESALE": 0, "IMEDIA": 100, "SHOPPAY": 0 },
      "provider_metrics": {
        "ESALE": { "routing_pct": 0, "success_pct": 0, "pending_pct": 0, "fail_pct": 0, "pending_txn": 0, "fail_txn": 0 },
        "IMEDIA": { "routing_pct": 100, "success_pct": 97.2, "pending_pct": 0.3, "fail_pct": 2.5, "pending_txn": 0, "fail_txn": 25 },
        "SHOPPAY": { "routing_pct": 0, "success_pct": 0, "pending_pct": 0, "fail_pct": 0, "pending_txn": 0, "fail_txn": 0 }
      }
    }
  ]
}
```

(Giá trị `health_status` = kết quả snapshot lúc `updated_at`; UI hiển thị 🔴 cho `V100K` vì `pending_plan` **hoặc** metric vi phạm — xem §2.3.2.)

**Ví dụ scope đang bảo trì** (cùng endpoint, row rút gọn):

```json
{
  "sku_code": "V100K",
  "routing_pct": { "ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20 },
  "provider_metrics": {
    "ESALE": { "routing_pct": 0, "success_pct": 0, "pending_pct": 0, "fail_pct": 0, "pending_txn": 0, "fail_txn": 0, "under_maintenance": true },
    "IMEDIA": { "routing_pct": 0, "success_pct": 0, "pending_pct": 0, "fail_pct": 0, "pending_txn": 0, "fail_txn": 0, "under_maintenance": true },
    "SHOPPAY": { "routing_pct": 0, "success_pct": 0, "pending_pct": 0, "fail_pct": 0, "pending_txn": 0, "fail_txn": 0, "under_maintenance": true }
  },
  "maintenance": { "scope_level": true, "starts_at": "...", "ends_at": "...", "remaining_min": 42 }
}
```

(Khi `maintenance` active: **không** trả `pending_plan`; cột provider UI **disabled**, mọi chỉ số hiển thị **0**.)

- `service_type`: `card` | `topup` | `topup_data` — từ bảng `products`; UI dùng để tab lọc loại dịch vụ.
- `provider_metrics` (optional): map per provider — `% Routing`, `% Success`, `% Pending`, `% Fail`, `pending_txn`, `fail_txn`, `under_maintenance?` — dùng cho cột ESALE/IMEDIA/SHOPPAY §9.0. Provider `routing_pct=0` (hard cut) vẫn trả object metric **0**. Provider **đang bảo trì** (theo provider hoặc BT toàn SKU khi mọi provider active đều BT): snapshot trả metric **0**, `routing_pct` hiển thị **0**, `under_maintenance: true` — UI disabled (§9.0).
- `live_metrics` (optional): snapshot gom scope — **legacy/audit**; UI dùng `provider_metrics`.
- `health_status` **theo từng scope** (product × SKU): từ `buildScopeSnapshot()` — đánh giá consecutive breach trên **từng provider active** (§2.3.2). **Không** lấy `health_status_product` product-level.
- `auto_action`, `window_start`, `window_end` (optional): cấu hình auto **hiệu lực** cho scope — từ `ResolveEffectiveScopeAuto` (§9.5.2): nếu có row `routing_scope_state` với `sku_code=""` (cấu hình **dịch vụ**) → dùng giá trị đó cho mọi SKU thuộc product; ngược lại fallback row `(product_code, sku_code)`.
- `product_auto_action`, `product_window_start`, `product_window_end` (optional): cấu hình **gốc** cấp dịch vụ (`sku_code=""`) — dùng cho `ScopeAutoEditor` cột **Dịch vụ**.
- `scope_auto_action`, `scope_window_start`, `scope_window_end` (optional): cấu hình **gốc** cấp SKU — dùng cho `ScopeAutoEditor` cột **Bảo trì + Chế độ BT / Routing** (hiển thị giá trị đã lưu per SKU; **không** thay thế bởi cấu hình dịch vụ — xem §9.5.2).
- `pending_plan`: tối đa **1 bản ghi DB** / scope (`status ∈ {pending_approve, draft}`) **hoặc** đề xuất **synthetic** (`suggested: true`, `id: 0`). **Không** trả khi `ShouldAutoApplyScope=true` (§9.5.2) hoặc scope có cửa sổ `maintenance` active. UI **Duyệt/Từ chối** cả hai loại (plan có `id` → `POST /routing-plans/{id}/…`; synthetic → `POST /scopes/{product}/{sku}/routing/…`). **Refresh trên poll:** mỗi `GET /dashboard/overview` — nếu còn plan DB và metric live vẫn vi phạm → `refreshPendingRoutingFromSnapshot` + `UPDATE routing_plans.plan_json`; hết vi phạm → `CancelPendingRoutingPlansForScope`. Agent (`reasoning.go`) auto apply hoặc `UpdatePendingRoutingPlanForScope` tùy `ShouldAutoApplyScope`.
- `pending_maintenance` (optional): khi breach nhưng `suggested_action=maintenance` — **không** trả khi `ShouldAutoApplyScope=true`. Object `{ id?, provider_code?, scope_level?, reason, action_type, suggested? }`. Bảo trì **cả SKU** → `scope_level=true`. Duyệt gửi `starts_at` + `ends_at` (ISO); mặc định UI = now → now+60p. Sau **Từ chối** → poll tiếp theo có thể trả đề xuất synthetic mới (không ẩn theo thời gian).
- `maintenance`: chỉ trả khi `starts_at ≤ now < ends_at` (tránh hiển thị *còn 0 phút* khi DB vẫn `active` nhưng đã hết hạn).
- **UI icon:** `pending_plan` chờ duyệt → cột TT dòng 🔴 (override); **hoặc** bất kỳ metric nào vượt ngưỡng → 🔴 (`effectiveRowHealth`); tab chứa SKU đó → icon tab 🔴 (§9.0).
- **UI sort:** SKU số (`10000`, `20000`, …) sort **tăng dần theo giá trị số**; SKU chữ sort `localeCompare` numeric.

### 2.3.2 Health hiển thị Dashboard vs Agent (triển khai `dashboard.go`)

Hai nguồn trạng thái — **không trộn lẫn**:

| Nguồn | API / lưu trữ | Mục đích |
| --- | --- | --- |
| **Agent (chu kỳ)** | `agent_state_history`, `health_status_product`, `GET /health-status` | State machine §3.1, consecutive breach, recovery timeline, SSE feed, audit |
| **Dashboard (live)** | `GET /dashboard/overview` → `health_status` + `provider_metrics` mỗi row | Cột **TT**, cột **provider** (6 chỉ số/provider), tab loại dịch vụ, `useOverallHealth` — phản ánh **metric lúc gọi API** |

**Thuật toán `buildScopeSnapshot` + `scopeSuggestionFromSnapshot`** (Go `internal/api/dashboard_snapshot.go`, `dashboard.go`, `dashboard_suggest.go`):

1. Load `product_alert_thresholds` + metric window (mặc định 15 phút) + agent history (consecutive breach).
2. Với **mỗi provider** trong `routing_pct` của scope:
   - **Đang bảo trì** (`maintainedActiveProviders` hoặc BT toàn SKU khi mọi provider active đều BT): metric **0**, `routing_pct` hiển thị **0**, `under_maintenance: true` — **không** tính breach / worst / gom `live_metrics`.
   - `routing_pct > 0` (không BT): metric thực từ `LoadLatestMetricsSince`.
   - `routing_pct = 0` (hard cut): object metric **0** — UI inactive + nút **Mở lại**.
3. **`ScopeBreachedFromSnapshot`** — **per provider**; chỉ provider **`routing_pct > 0` và không BT** tham gia breach; **5 điều kiện OR**:

| Chỉ số UI | Vi phạm (→ 🔴) | Dấu hàng Ngưỡng |
| --- | --- | --- |
| % Success | `success_pct ≤ success_rate_min_pct` | `≤` |
| % Pending | `pending_pct ≥ pending_rate_max_pct` | `≥` |
| % Fail | `fail_pct ≥ fail_rate_max_pct` | `≥` |
| Pending (GD) | `pending_txn ≥ pending_txn_count_max` | `≥` |
| Fail (GD) | `fail_txn ≥ fail_txn_count_max` | `≥` |

4. **Bất kỳ provider active** vi phạm ≥ 1 điều kiện → scope **đang theo dõi**; màu TT phụ thuộc `consecutive_cycles_required` (§1.2):
   - **Chu kỳ 1** vi phạm (hoặc live trước khi đủ chu kỳ liên tiếp): `health_status=yellow` — **không** đề xuất routing/bảo trì mới khi `ShouldAutoApplyScope=false`; **không** auto khi chưa đủ gate (trừ ngoại lệ force §9.5.2).
   - **Chu kỳ liên tiếp thứ 2+** (`consecutive >= required`, mặc định 2): `health_status=red` — sinh đề xuất / hành động theo `auto_action` §9.5.2.
   - **Ngoại lệ force (bypass gate chu kỳ)** — chỉ khi `ShouldAutoApplyScope=true` (`auto` hoặc `time_window` **trong** khung); poll Dashboard + Agent §9.5.2:
     - **`ShouldForceAutoRouting`:** `ActiveRoutingCount ≥ 2`, `SKURoutingDecision=routing` (còn ≥1 provider khỏe) → **tự `UpdateRouting` ngay** (shift khỏi provider vi phạm), không chờ `consecutive_cycles_required`.
     - **`ShouldForceAutoMaintenanceAllProviders`:** `ActiveRoutingCount ≥ 2`, `SKURoutingDecision=maintenance` (mọi provider active vi phạm) → **tự `SetMaintenance` ngay**.
     - **`ShouldForceAutoMaintenance`:** `ActiveRoutingCount == 1`, provider đó vi phạm → **tự `SetMaintenance` ngay** — không còn provider khác để shift.
   - **Chu kỳ kế tiếp không vi phạm**: `health_status=green` — reset chuỗi consecutive.
5. **`ScopeConsecutiveBreachFromHistory`**: đếm chu kỳ Agent liên tiếp vi phạm (history + live); `ShouldAct = consecutive >= required` trên **bất kỳ** provider vi phạm (không chỉ worst), **hoặc** một trong ba hàm force trên (mục 4) khi `ShouldAutoApplyScope=true`.
6. Không fallback `health_status_product`; không đọc `agent_state_history` trực tiếp cho màu TT.
7. **`error_event_count_max`** — Agent §7.4 vẫn dùng; **Dashboard không hiển thị** error event trên cột provider.
8. `live_metrics` (gom scope) vẫn trả API — **UI không dùng**; cột provider tô đỏ từng metric vượt ngưỡng (`ProviderMetricCell`), trừ khi `under_maintenance`.

**Frontend (`effectiveRowHealth` + `ProviderMetricCell` + `maintenanceDisplay.ts` + `SkuMaintenanceTimeLabel`):** Cột **TT** và tab dùng `health_status` API — **ẩn icon** khi scope **đang bảo trì** (`row.maintenance` / `isSkuUnderActiveMaintenance`: mọi provider active đều BT); **nhãn thời gian BT** hiển thị **dưới tên SKU** (ô gộp SKU+TT, §9.0), cột **Bảo trì** chỉ còn nút hành động. Cột **provider**: đỏ/xanh theo metric; `routing_pct=0` → muted + **Mở lại** → mở hàng *Mở lại provider* (§9.0); **Lưu** = baseline → `POST .../routing/restore-baseline`, khác baseline → `POST .../routing/apply`; `under_maintenance` / `maintenance.scope_level` → **disabled**, metric **0**, không **Mở lại**. Hàng *Kế hoạch routing*: `ServiceOverviewTable` sync `draftRouting` khi `proposed_pct` hoặc `updated_at` đổi (poll 60s); **ẩn** khi `shouldShowManualApproval(row)=false` (§9.5.2); sau **Từ chối** routing → hủy plan pending DB + UI không treo hàng 0%/100% không nút.

**Map màu UI (Dashboard row — cột TT):**

| `health_status` API | Cột TT | Ghi chú |
| --- | --- | --- |
| *(đang bảo trì)* | *(gộp SKU)* | Không hiển thị 🟢🟡🔴 — nhãn thời gian BT dưới tên SKU (§9.0.1); cột **Bảo trì** = nút Mở lại/Gia hạn |
| `green` | 🟢 | Không provider active vi phạm snapshot **hoặc** đã hết chuỗi vi phạm |
| `yellow` | 🟡 | Vi phạm lần đầu / chưa đủ `consecutive_cycles_required` — theo dõi; **chưa** hàng đề xuất mới (trừ plan DB cũ khi `ShouldAutoApplyScope=false`) |
| `red` | 🔴 | Đủ chu kỳ liên tiếp vi phạm **hoặc** có `pending_plan` / `pending_maintenance` chờ duyệt (`ShouldAutoApplyScope=false`) |

**`GET /health-status`** vẫn aggregate **theo chu kỳ Agent** (product-level + global cycle) — dùng header badge phụ; logo badge ưu tiên worst từ overview (`useOverallHealth`: `max(overview rows, health-status API)`).

**Đề xuất routing/bảo trì khi gọi `GET /dashboard/overview`** (`dashboard_suggest.go` + `dashboard.go`):

1. **Đề xuất mới (synthetic):** chỉ khi `snap.ShouldAct == true` (`consecutive >= required`) → `scopeSuggestionFromSnapshot`. Routing → `routingPlanResponse` (nếu đã có plan DB thì `UPDATE plan_json` rồi trả row DB; chưa có → synthetic `id: 0`, `suggested: true`).
2. **Refresh plan DB (poll ~60s):** nếu scope còn plan `pending_approve`/`draft`, metric live **vẫn vi phạm** (`AnyBreached`) nhưng chưa đủ `ShouldAct` hoặc agent chưa chạy → `refreshPendingRoutingFromSnapshot` (không yêu cầu `ShouldAct`) → `UPDATE routing_plans.plan_json` → trả plan DB mới.
3. **Hết vi phạm:** `CancelPendingRoutingPlansForScope` — ẩn hàng *Kế hoạch routing*.
4. **Cửa sổ bảo trì active:** **không** trả `pending_plan` (ưu tiên cột Bảo trì); cột provider disabled (mục 2 trên).
5. **Chu kỳ vi phạm đầu** (`consecutive < required`): cột TT 🟡, **không** hàng đề xuất mới — trừ khi đã có plan DB từ chu kỳ trước (refresh mục 2).
6. **`SKURoutingDecision`**: tất cả provider `routing_pct>0` vi phạm → `maintenance` (ưu tiên đề xuất/bảo trì, **hủy** plan routing pending khi refresh); một phần vi phạm → `routing` (hard cut §8.6.3).
7. **Auto apply trên poll** (`ShouldAutoApplyScope=true`): `scopeAutoApplyAllowed` = `ShouldAct` **hoặc** `ShouldForceAutoRouting` **hoặc** `ShouldForceAutoMaintenanceAllProviders` **hoặc** `ShouldForceAutoMaintenance` → `autoApplyScopeFromSnapshot` — routing trước; nếu `proposed == current` hoặc `SKURoutingDecision=maintenance` → `SetMaintenance`; tối đa **2 pass**/request (routing → bảo trì). Ba hàm **force** bypass gate `consecutive_cycles_required` (§9.5.2).
8. Persist chính: **worker-agent** (`reasoning.go`, cùng bộ force + `ShouldAct`), admin **Duyệt**, **auto** per SKU — **và** refresh plan pending trên GET overview (mục 2) khi `ShouldAutoApplyScope=false`.
9. **Từ chối / Duyệt routing:** mỗi SKU chỉ có **một đề xuất chờ** (`pending_approve` / `draft` hoặc synthetic live). Sau **Duyệt** hoặc **Từ chối** → `CancelPendingRoutingPlansForScope` (plan DB) + có thể tạo đề xuất mới nếu metric vẫn vi phạm (`HasPendingRoutingPlan` / không còn plan chờ).
10. **Từ chối / Duyệt bảo trì:** sau **Từ chối** (synthetic hoặc recommendation Agent) → **không** ẩn theo thời gian; poll tiếp theo sinh lại `pending_maintenance` synthetic nếu metric vẫn vi phạm và `suggested_action=maintenance`. Ghi `DISMISSED:…` trong `recommendations` chỉ để audit.
11. **Không** tạo incident trên GET — incident do Agent §8.3 khi `ShouldAct` hoặc một trong ba hàm force §9.5.2.

**Response `GET /incidents` (phân trang):**

```json
{
  "items": [ { "incident_id": "20260609-001", "status": "open", ... } ],
  "total": 42,
  "page": 1,
  "page_size": 10
}
```

Query: `page` (1-based, mặc định 1), `page_size` (mặc định 10, max 50), `since` (RFC3339, optional).

### 2.3.1 Dev auth & CORS (triển khai `ai_gen_src`)

| Biến env (Go) | Mặc định dev | Mô tả |
| ------------- | ------------ | ----- |
| `DEV_AUTH_BYPASS` | `true` | Bỏ JWT; đọc `X-OpsOne-Role` / `X-OpsOne-Actor` |
| `CORS_ORIGIN` | `http://localhost:5173` | Vite dev proxy `/api` → `:8080` |
| `API_ADDR` | `:8080` | `cmd/api` |

| Biến env (React `web/.env`) | Mặc định dev | Mô tả |
| --------------------------- | ------------ | ----- |
| `VITE_API_BASE_URL` | (trống) | Local: proxy `/api` → `:8080`. GreenNode web image: `https://<opsone-api>/api/v1` (build-arg) |
| `VITE_DEV_AUTH_BYPASS` | `true` | Gửi header admin tới API |
| `VITE_AAD_*` | (trống) | Bật MSAL khi có tenant/client (§2.6.4) |

**Chạy local (4 terminal):** `docker compose up -d` · `go run ./cmd/worker-mock` · `go run ./cmd/worker-agent` · `.\scripts\run-api.ps1` (Windows; hoặc `go run ./cmd/api` + nạp `.env`) · `cd web && .\dev.ps1`

**Seed Windows:** `scripts/dev.ps1` → `Invoke-OpsOneReset` — `Wait-OpsOneMysqlReady` (poll tối đa 120s) rồi **`docker cp` + `mysql source`** + `--default-character-set=utf8mb4` (giữ UTF-8 trong `seed.sql` — không pipe stdin PowerShell).

---

## 2.4 Lộ trình build & test (thứ tự bắt buộc)


| #   | Việc                                                            | Verify                               |
| --- | --------------------------------------------------------------- | ------------------------------------ |
| 1   | `docker compose up mysql` + chạy `db/schema.sql`, `db/seed.sql` | `SELECT COUNT(*) FROM products` = 11 |
| 2   | `internal/store` — integration test MySQL                       | CRUD products, routing_config        |
| 3   | `cmd/worker-mock` — tick 1 phút                                 | Rows trong `mock_metrics` tăng       |
| 4   | `internal/tools` — unit test từng tool                          | GetMetrics mock window 15m           |
| 5   | `internal/threshold` — unit test ngưỡng % + fail_txn            | Fixture §10.11                       |
| 6   | `internal/rules` — unit test rules 1–9                          | Fixture §10                          |
| 7   | `internal/agent` — dry-run 1 cycle                              | Ghi `agent_analysis_cycles`          |
| 8   | `cmd/worker-agent` — chạy nền                                   | Incident + health_status xuất hiện   |
| 9   | `cmd/api` — REST                                                | `curl /health-status`                |
| 10  | React Dashboard — bảng routing §9.0 + incidents + HealthBadge   | F5 sau `run-api.ps1` / `go run ./cmd/api` |
| 11  | React Dashboard — ngưỡng per product §9.5.3 + Auto per dịch vụ/SKU §9.5.2 + metric live | §10.11                               |
| 12  | Scenario A–D — routing output                                   | §10.1–10.4                           |
| 13  | React Chat + voice                                              | §10.5                                |
| 14  | Scenario G mock + H icons                                       | §10.7–10.8                           |
| 15  | Approve routing → `agent_change_log`                            |                                      |
| 16  | Mở lại provider / Mở lại dịch vụ → routing baseline biz          | §8.7 / Dashboard                     |
| 17  | Vượt ngưỡng → routing/bảo trì + email                           | §10.10, §10.11                       |
| 18  | `make test` — Go + (optional) React                             |                                      |
| 19  | DoD §12                                                         | Demo end-to-end không production     |


**Makefile gợi ý:**

```makefile
migrate:  mysql -u root -p opsone < db/schema.sql
seed:     mysql -u root -p opsone < db/seed.sql
run-api:  pwsh -File scripts/run-api.ps1   # Windows: nạp .env + kill port trước khi start
run-mock: go run ./cmd/worker-mock
run-agent: go run ./cmd/worker-agent
run-web:  cd web && npm run dev
test:     go test ./...
```

**PowerShell (`scripts/dev.ps1`):** `Invoke-OpsOneReset` (chờ MySQL ready → DROP + CREATE `schema.sql` + `seed.sql`) · `Start-OpsOneAPI` (`run-api.ps1` — nạp `.env`, kill port `:8080`, chạy API) · `Invoke-OpsOneClearRuntime` · `Invoke-OpsOneE2E` · …

---

## 2.6 Authentication & Authorization — Microsoft 365 (Entra ID)

OpsOne dùng **tài khoản Microsoft 365 doanh nghiệp** (Microsoft Entra ID / Azure AD) làm SSO duy nhất cho ops + admin. Chuẩn: **OpenID Connect (OIDC)** trên nền **OAuth 2.0 Authorization Code Flow with PKCE** — phù hợp cho SPA React + Go API.

> **Không** tạo bảng password riêng. Mọi danh tính do Microsoft Identity Platform cấp; OpsOne chỉ **verify token** và **map role**.

### 2.6.1 Luồng xác thực tổng quan

```text
[React SPA]
   │ 1. Click "Đăng nhập với Microsoft"
   ▼
[login.microsoftonline.com/{tenant_id}]
   │ 2. User nhập tài khoản O365 + MFA
   │ 3. Trả authorization code (PKCE)
   ▼
[React SPA]
   │ 4. Đổi code → ID Token (OIDC) + Access Token (scope api://opsone-api/access_as_user)
   │ 5. Lưu trong sessionStorage (MSAL.js)
   │
   │ 6. Gọi API: Authorization: Bearer <access_token>
   ▼
[Go API /api/v1/*]
   │ 7. Middleware: Verify JWT (RS256) qua JWKS public keys của tenant
   │ 8. Đọc claims: oid, preferred_username (UPN), roles[]
   │ 9. Map roles → admin | ops; gắn vào request context
   ▼
[Handler] — thực thi nghiệp vụ + ghi audit log = UPN
```

### 2.6.2 Azure AD App Registration (IT setup, 1 lần)

Đăng ký 2 app trên **Azure Portal → Microsoft Entra ID → App registrations**:

| Mục | App 1 — Frontend SPA | App 2 — Backend API |
|-----|----------------------|---------------------|
| **Name** | `OpsOne Web` | `OpsOne API` |
| **Account type** | Single tenant | Single tenant |
| **Platform** | Single-page application | — (Web API) |
| **Redirect URI** | `http://localhost:5173/auth/callback` (dev), `https://opsone.company.com/auth/callback` (prod) | — |
| **API permissions** | `Microsoft Graph` → `User.Read` (delegated) + scope của App 2 | — |
| **Expose an API** | — | Application ID URI: `api://opsone-api`; Scope: `access_as_user` |
| **App roles** | — | Tạo 2 role: `Admin`, `Ops` (Allowed member types: Users/Groups) |

Sau khi tạo, gán user vào role qua **Enterprise applications → OpsOne API → Users and groups → Add assignment**.

Ghi nhận các giá trị: `TENANT_ID`, `WEB_CLIENT_ID` (App 1), `API_CLIENT_ID` (App 2).

### 2.6.3 Biến môi trường (bổ sung `.env` §2.1)

```bash
AAD_TENANT_ID=00000000-0000-0000-0000-000000000000
AAD_WEB_CLIENT_ID=11111111-1111-1111-1111-111111111111
AAD_API_CLIENT_ID=22222222-2222-2222-2222-222222222222
AAD_API_SCOPE=api://opsone-api/access_as_user
AAD_AUTHORITY=https://login.microsoftonline.com/${AAD_TENANT_ID}
AAD_ISSUER=https://login.microsoftonline.com/${AAD_TENANT_ID}/v2.0
AAD_JWKS_URL=https://login.microsoftonline.com/${AAD_TENANT_ID}/discovery/v2.0/keys
AAD_JWKS_CACHE_MIN=60
DEV_AUTH_BYPASS=false
```

Frontend Vite cần biến `VITE_AAD_TENANT_ID`, `VITE_AAD_WEB_CLIENT_ID`, `VITE_AAD_API_SCOPE`.

### 2.6.4 Frontend — MSAL.js (React)

```bash
npm install @azure/msal-browser@3.10.0 @azure/msal-react@2.0.21
```

`web/src/auth/msalConfig.ts`:

```ts
import { Configuration, PublicClientApplication } from '@azure/msal-browser';

export const msalConfig: Configuration = {
  auth: {
    clientId: import.meta.env.VITE_AAD_WEB_CLIENT_ID,
    authority: `https://login.microsoftonline.com/${import.meta.env.VITE_AAD_TENANT_ID}`,
    redirectUri: window.location.origin + '/auth/callback',
    postLogoutRedirectUri: window.location.origin + '/',
  },
  cache: { cacheLocation: 'sessionStorage', storeAuthStateInCookie: false },
};

export const loginRequest = {
  scopes: ['openid', 'profile', 'User.Read', import.meta.env.VITE_AAD_API_SCOPE],
};

export const pca = new PublicClientApplication(msalConfig);
```

`web/src/main.tsx`:

```tsx
import { MsalProvider } from '@azure/msal-react';
<MsalProvider instance={pca}><App /></MsalProvider>
```

`web/src/api/client.ts` — gắn token vào mỗi request:

```ts
async function authHeader(): Promise<Record<string, string>> {
  const account = pca.getActiveAccount() ?? pca.getAllAccounts()[0];
  if (!account) throw new Error('chưa đăng nhập');
  const res = await pca.acquireTokenSilent({ ...loginRequest, account });
  return { Authorization: `Bearer ${res.accessToken}` };
}

export async function api(path: string, init: RequestInit = {}) {
  const headers = { 'Content-Type': 'application/json', ...(await authHeader()), ...init.headers };
  return fetch(`/api/v1${path}`, { ...init, headers });
}
```

**SSE `/events`:** dùng `EventSource` không gửi được header → backend nhận token qua `?access_token=...` (query); chỉ chấp nhận scope SSE hẹp, hoặc dùng cookie session sau exchange.

**Logout:** `pca.logoutRedirect({ postLogoutRedirectUri: '/' })` — Microsoft sẽ xoá session phía O365.

### 2.6.5 Backend — Go middleware verify JWT

```bash
go get github.com/coreos/go-oidc/v3/oidc@v3.10.0
go get golang.org/x/oauth2@v0.22.0
```

`internal/auth/middleware.go`:

```go
package auth

import (
    "context"
    "net/http"
    "strings"
    "github.com/coreos/go-oidc/v3/oidc"
)

type ctxKey string
const CtxUser ctxKey = "opsone.user"

type AzureClaims struct {
    Oid   string   `json:"oid"`
    UPN   string   `json:"preferred_username"`
    Name  string   `json:"name"`
    Roles []string `json:"roles"`
    Aud   string   `json:"aud"`
    Iss   string   `json:"iss"`
    Exp   int64    `json:"exp"`
}

func (c *AzureClaims) HasRole(role string) bool {
    for _, r := range c.Roles { if r == role { return true } }
    return false
}

func RequireAuth(verifier *oidc.IDTokenVerifier, devBypass bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if devBypass {
                ctx := context.WithValue(r.Context(), CtxUser, &AzureClaims{UPN: "dev@local", Roles: []string{"Admin"}})
                next.ServeHTTP(w, r.WithContext(ctx)); return
            }
            raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
            if raw == "" {
                http.Error(w, "missing bearer token", http.StatusUnauthorized); return
            }
            tok, err := verifier.Verify(r.Context(), raw)
            if err != nil { http.Error(w, "invalid token", http.StatusUnauthorized); return }
            var c AzureClaims
            if err := tok.Claims(&c); err != nil { http.Error(w, "claims parse", http.StatusUnauthorized); return }
            next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), CtxUser, &c)))
        })
    }
}

func RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user, _ := r.Context().Value(CtxUser).(*AzureClaims)
            if user == nil || !user.HasRole(role) {
                http.Error(w, "forbidden", http.StatusForbidden); return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

`cmd/api/main.go` (khởi tạo):

```go
provider, err := oidc.NewProvider(ctx, os.Getenv("AAD_ISSUER"))
if err != nil { log.Fatal(err) }
verifier := provider.Verifier(&oidc.Config{ ClientID: os.Getenv("AAD_API_CLIENT_ID") })

r.Use(auth.RequireAuth(verifier, os.Getenv("DEV_AUTH_BYPASS") == "true"))
r.With(auth.RequireRole("Admin")).Put("/api/v1/config", ConfigPut)
r.With(auth.RequireRole("Admin")).Put("/api/v1/products/{code}/routing", ProductRoutingPut)
r.With(auth.RequireRole("Admin")).Post("/api/v1/routing-plans/{id}/approve", RoutingPlansApprove)
r.With(auth.RequireRole("Admin")).Post("/api/v1/routing-plans/{id}/approve", RoutingPlansApprove)
```

### 2.6.6 Map role Azure AD → OpsOne

| App Role (Azure AD) | OpsOne role | Quyền |
|---------------------|-------------|-------|
| `Admin` | admin | PUT `/config`, PUT thresholds, **PUT routing (`set_as_baseline`)**, approve routing/maintenance, gửi mail thử |
| `Ops` | ops | GET tất cả, chat/voice, xem incident, xem lịch sử |
| (không có role) | — | 403 (đã login O365 nhưng chưa được gán app role) |

Quy ước: claim `roles` (mảng) là tiêu chuẩn của Azure AD App Roles — không dùng `groups` (groups trả nhiều, có thể >200 và phải gọi Graph để resolve).

### 2.6.7 Bảng phân quyền endpoint (cập nhật §2.3)

| Method | Path | Role tối thiểu |
|--------|------|----------------|
| GET    | `/health-status`, `/incidents*`, `/routing-plans*`, `/maintenance*`, `/notifications`, `/escalation-chat`, `/products*`, `/alert-thresholds`, `/metrics`, `/mock/status`, `/config` | `Ops` |
| PUT    | `/config`, `/products/{code}/thresholds`, `/products/{code}/routing`, `/escalation-chat` | `Admin` |
| POST   | `/routing-plans/{id}/approve`, `/routing-plans/{id}/reject` | `Admin` |
| POST   | `/maintenance`, `/maintenance/{id}/approve`, `/maintenance/{id}/cancel` | `Admin` |
| POST   | `/notifications/test`, `/mock/generate` | `Admin` |
| POST   | `/chat`, `GET /chat/sessions/{id}` | `Ops` |
| GET    | `/events` (SSE) | `Ops` — token qua query `?access_token=` hoặc cookie session |

### 2.6.8 Audit log lấy danh tính từ JWT

Mọi cột `executed_by`, `changed_by`, `approved_by`, `cancelled_by`, `rolled_back_by`, `updated_by` (trừ Agent system) đều **lấy từ JWT claim `preferred_username` (UPN)** — handler **không tin** body request gửi.

```go
user := r.Context().Value(auth.CtxUser).(*auth.AzureClaims)
executedBy := user.UPN   // nguyen.van.a@company.com
```

Agent system tự chạy: `executed_by = 'opsone-agent'` (constant).

### 2.6.9 Bảng `users` — cache profile (optional nhưng nên có)

```sql
CREATE TABLE users (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  azure_oid     VARCHAR(64)  NOT NULL COMMENT 'claim oid — stable user id',
  upn           VARCHAR(128) NOT NULL COMMENT 'preferred_username — user@company.com',
  display_name  VARCHAR(128) NULL,
  role_cached   ENUM('admin','ops') NOT NULL DEFAULT 'ops' COMMENT 'role gần nhất (cache hiển thị, không thay verify JWT)',
  last_login_at DATETIME     NULL,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_users_oid (azure_oid),
  UNIQUE KEY uk_users_upn (upn),
  KEY idx_users_role (role_cached)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

Upsert trong middleware sau khi verify thành công — phục vụ UI hiển thị "Ai đã làm gì" và dropdown chọn approver.

### 2.6.10 Dev mode bypass (chỉ local)

`DEV_AUTH_BYPASS=true` chỉ cho phép ở môi trường dev:

- Middleware skip verify; gán user mặc định `dev@local` role `Admin`.
- Startup **assert** `ENV != prod` — nếu prod mà bật → log fatal + exit.
- CI/CD test E2E (§10) chạy với bypass = true.

### 2.6.11 UI — Login / Logout / Phân quyền hiển thị

```tsx
// AuthGate.tsx — chặn vào app nếu chưa login
const { accounts, instance } = useMsal();
if (accounts.length === 0) return <button onClick={() => instance.loginRedirect(loginRequest)}>Đăng nhập với Microsoft</button>;

// useRole.ts — đọc role từ idTokenClaims để ẩn/hiện nút admin
export function useRole(): 'admin' | 'ops' {
  const account = useAccount();
  const roles = (account?.idTokenClaims as any)?.roles ?? [];
  return roles.includes('Admin') ? 'admin' : 'ops';
}

// Trong Settings.tsx
{role === 'admin' ? <SaveButton /> : <Hint>Chỉ admin được sửa</Hint>}
```

### 2.6.12 Checklist triển khai Auth

- [ ] IT đăng ký 2 app Azure AD (Web SPA + Backend API), gán App roles `Admin`/`Ops`.
- [ ] IT cấp `TENANT_ID`, 2 `CLIENT_ID`, scope `api://opsone-api/access_as_user`.
- [ ] FE: cài MSAL.js, login button, protected routes, auto refresh token, logout về Microsoft.
- [ ] BE: middleware verify JWT qua JWKS (cache 1h); validate `aud` = `AAD_API_CLIENT_ID`, `iss` = `AAD_ISSUER`.
- [ ] BE: `RequireRole("Admin")` cho mọi endpoint mutate; trả 403 không lộ thông tin nội bộ.
- [ ] Audit: mọi `executed_by`/`changed_by` lấy từ JWT, **không** từ body.
- [ ] SSE `/events`: nhận token query với scope hẹp hoặc cookie session ngắn hạn.
- [ ] Bảng `users` upsert mỗi login; truy vấn ai-đã-làm-gì trên UI.
- [ ] `DEV_AUTH_BYPASS` chỉ bật ở dev; startup assert chặn prod.
- [ ] Test: token hết hạn → FE silent refresh; refresh fail → redirect login.
- [ ] Test: user chưa được gán App Role → 403 + UI hiển thị *"Tài khoản chưa được cấp quyền — liên hệ IT"*.
- [ ] Tài liệu IT-onboarding: cách gán user vào `Admin` hoặc `Ops`.

---

## 3. Một chu kỳ Agent (kịch bản runtime)

Mỗi **5 phút**, Agent chạy **một pipeline** cố định:

1. **Kích hoạt** — Scheduler trigger (không cần user chat).
2. **Thu thập** — `GetMaintenance` (§6.8): product đang bảo trì → skip metric. Theo `routing_mode`:
  - **Thẻ / Topup data** (`routing_mode=sku`): product → **SKU** → provider.
  - **Topup tiền** (`routing_mode=provider`): product → provider (không loop SKU).
3. **So sánh lịch sử** — Đọc `agent_analysis_history` (xu hướng N chu kỳ gần nhất).
4. **Đánh giá ngưỡng** — `EvaluateThresholds(product)` (§1.2, §7.4): so % và **số GD lỗi** với config per product.
5. **Suy luận** — Reasoning Engine áp dụng rules + LLM (chỉ khi đã/quasi vượt ngưỡng).
6. **Sinh output** — **Health Status** + Incident / Recommendation / Maintenance / Routing Plan.
7. **Hành động** — Nếu vượt ngưỡng + `auto_action` của scope cho phép (§9.5.2) → `UpdateRouting` hoặc `SetMaintenance`.
8. **Ghi log thay đổi** — `agent_change_log` khi routing thực thi (§8.7).
9. **Ghi log phân tích** — `agent_analysis_history`.
10. **Email ops** — Khi vượt ngưỡng **và/hoặc** vừa routing/bảo trì → mail §8.9 (kèm ngưỡng bị vượt).

---

## 3.1 State Machine của Agent (per product × scope)

> Mỗi `(product_code, sku_code)` có **state** riêng, được Agent tính lại **mỗi chu kỳ** và lưu vào `agent_state_history` (§13.7). State quyết định output type, color UI, và hành vi chu kỳ kế tiếp.

### 3.1.1 Sơ đồ trạng thái

```text
              ┌──────────────────────────────────────────────┐
              │                                              │
              ▼                                              │
        ┌──────────┐   metric vượt ngưỡng 1 chu kỳ    ┌──────────┐
        │  NORMAL  ├──────────────────────────────────►│ WARNING  │
        │   🟢     │◄──────metric hồi────────────────  │   🟡     │
        └──────────┘                                  └────┬─────┘
              ▲                                            │  vượt đủ
              │                                            │  consecutive_cycles
              │ recovery_cycles                            ▼
              │ liên tiếp ổn định                    ┌──────────┐
        ┌─────┴──────┐                              │ INCIDENT │
        │ RECOVERING │                              │   🔴     │
        │    🟡       │                              └────┬─────┘
        └─────▲──────┘                                    │ EvaluateThresholds
              │                                           │ + suggested_action
              │                                           │
              │ apply OK                                  ▼
              │                              ┌────────────────────────┐
              │                              │   ROUTING_PROPOSED     │
              │                              │  /MAINTENANCE_PROPOSED │
              │                              │         🟡             │
              │                              └────┬───────────────────┘
              │                                   │
              │     auto / approve                │       reject / timeout
              │                                   ▼                ▼
              │                          ┌─────────────┐   ┌────────────┐
              │                          │  ROUTING_   │   │  INCIDENT  │
              └──────────────────────────┤  APPLIED /  │   │  (giữ)     │
                                         │ MAINT_ACTIVE│   └────────────┘
                                         │     🔴       │
                                         └─────────────┘
```

### 3.1.2 Bảng trạng thái

| State | Color UI | Khi nào | Output sinh ra mỗi chu kỳ | Hành vi Agent |
|-------|----------|---------|---------------------------|---------------|
| `NORMAL` | 🟢 green | Không breached; metric trong baseline | `health_status=green`; không Incident | Chỉ ghi `agent_analysis_history` |
| `WARNING` | 🟡 yellow | Breached 1 chu kỳ nhưng chưa đủ `consecutive_cycles` | `health_status=yellow`; Recommendation `monitor` | Tăng `consecutive_breach_cycles`; chưa hành động |
| `INCIDENT` | 🔴 red | Breached + đủ `consecutive_cycles` | `health_status=red`; Incident `open`; gọi rules 1–9 | Quyết định `suggested_action` (§7.4) |
| `ROUTING_PROPOSED` | 🟡 yellow | INCIDENT + `suggested_action=routing` + chưa apply | Routing Plan `pending_approve` (chỉ khi `ShouldAutoApplyScope=false`) | Chờ approve (`recommend_only` / `time_window` ngoài giờ) hoặc tự `UpdateRouting` (`auto` / `time_window` trong giờ) |
| `MAINTENANCE_PROPOSED` | 🟡 yellow | INCIDENT + `suggested_action=maintenance` | Recommendation bảo trì (§8.5) — chỉ UI khi `ShouldAutoApplyScope=false` | Chờ admin approve hoặc tự `SetMaintenance` khi `ShouldAutoApplyScope=true` |
| `ROUTING_APPLIED` | 🔴 red | Vừa apply routing (chu kỳ apply) | `agent_change_log` + đóng sự cố §8.3.1 | Chờ **1 chu kỳ** agent → RECOVERING |
| `MAINTENANCE_ACTIVE` | 🔴 red | Maintenance window đang chạy | Card 🔧 + countdown | Skip cảnh báo metric; chờ `ends_at` |
| `RECOVERING` | 🟡 yellow | **+1 chu kỳ** sau apply routing | `health_status=yellow`; summary *đang hồi phục* | Theo dõi metric |
| `NORMAL` (post-recovery) | 🟢 green | **+2 chu kỳ** sau apply **và** metric không breach | `health_status=green` | Xóa `recovery_apply_cycle_id`; tuỳ chọn restore baseline §8.6.3 Bước 4 |

### 3.1.3 Transition matrix (machine-readable cho AI codegen)

```yaml
state_machine:
  initial: NORMAL
  scope: per_product_sku   # (product_code, sku_code)

  transitions:
    - from: NORMAL
      to: WARNING
      when: EvaluateThresholds.breached == true AND consecutive_breach_cycles < required

    - from: WARNING
      to: INCIDENT
      when: EvaluateThresholds.breached == true AND consecutive_breach_cycles >= required

    - from: WARNING
      to: NORMAL
      when: EvaluateThresholds.breached == false

    - from: INCIDENT
      to: ROUTING_PROPOSED
      when: suggested_action == "routing"

    - from: INCIDENT
      to: MAINTENANCE_PROPOSED
      when: suggested_action == "maintenance"

    - from: ROUTING_PROPOSED
      to: ROUTING_APPLIED
      when: scope.auto_action allows auto apply (§9.5.2) OR routing_plan.status == "approved"
      side_effects: [UpdateRouting, agent_change_log INSERT, email §8.9]

    - from: ROUTING_PROPOSED
      to: INCIDENT
      when: routing_plan.status == "rejected"

    - from: MAINTENANCE_PROPOSED
      to: MAINTENANCE_ACTIVE
      when: maintenance.status == "scheduled" AND starts_at <= now
      side_effects: [SetMaintenance commit, email §8.9]

    - from: ROUTING_APPLIED
      to: RECOVERING
      when: analysis_cycles_since_apply >= 1

    - from: RECOVERING
      to: NORMAL
      when: analysis_cycles_since_apply >= 2 AND EvaluateThresholds.breached == false
      side_effects: [clear recovery_apply_cycle_id on routing_scope_state]

    - from: RECOVERING
      to: INCIDENT
      when: analysis_cycles_since_apply >= 2 AND EvaluateThresholds.breached == true

    - from: ROUTING_APPLIED
      to: INCIDENT
      when: vẫn breached sau 1 chu kỳ apply (routing không cải thiện)
```

### 3.1.4 Persistence — bảng `agent_state_history`

Mỗi chu kỳ ghi 1 dòng per `(product, sku)` để vẽ timeline + audit:

```sql
-- DDL chi tiết §13.7
agent_state_history(cycle_id, product_code, sku_code, state, prev_state, transition_reason, recorded_at)
```

### 3.1.5 Checklist triển khai State Machine

- [ ] `internal/agent/state.go` — enum `AgentState` + map transition.
- [ ] Hàm `nextState(currentState, evaluation, planStatus, maintenanceStatus) AgentState`.
- [ ] Mỗi chu kỳ: load `currentState` per scope → tính `nextState` → ghi `agent_state_history` + bắn SSE `state_changed`.
- [ ] Unit test 11 transition (3.1.3) — bảng fixture trong `internal/agent/state_test.go`.
- [x] UI Dashboard cột TT: màu từ **`liveScopeHealth` + `effectiveRowHealth` §2.3.2** (5 ngưỡng OR + metric live), không map trực tiếp `agent_state_history`.
- [ ] Chat: *"ZING SKU 20k đang ở state nào?"* → đọc `agent_state_history.latest` (Agent state, khác màu TT Dashboard).

---

# Part III — Agent Backend (Go)

> Map code: `cmd/worker-`*, `internal/agent`, `internal/tools`, `internal/rules` — xem §2.2.

## 4. Bước 1 — Scheduler

### 4.1 Vai trò

Đây là thành phần giúp Agent **hoạt động chủ động**.

- Biến Agent từ **phản hồi theo câu hỏi** → **chủ động theo thời gian**.
- Chu kỳ mặc định: **5 phút** — **cấu hình trên UI** (§9.5), scheduler đọc `interval_minutes` khi chạy.

### 4.2 Khác biệt so với chatbot thuần


| Chatbot thuần                   | **OpsOne**                                                          |
| ------------------------------- | ------------------------------------------------------------------- |
| Chỉ phản hồi khi user hỏi       | **Scheduler** tự chạy 10:00, 10:05, 10:10…                          |
| Không có bộ nhớ chu kỳ bắt buộc | Bắt buộc so sánh với lịch sử                                        |
| —                               | **UI chat / voice** dùng khi ops cần hỏi thêm, không thay scheduler |


**Hệ thống này:**

```text
5 phút/lần → Agent tự chạy

Ví dụ: 10:00 → 10:05 → 10:10 → 10:15 → ...
```

**Mỗi lần chạy Agent sẽ:**

1. Thu thập dữ liệu
2. Phân tích
3. So sánh với lịch sử
4. Đưa ra kết luận
5. Tạo recommendation

> Agent có thể nhìn được **xu hướng** chứ không chỉ snapshot.

### 4.3 Việc cần làm khi triển khai

- Scheduler đọc config từ `agent_config` (UI §9.5): `interval_minutes`, `scheduler_enabled`.
- Load **catalog sản phẩm** §1.1 (11 product) — `service_type` + `routing_mode` cho từng dòng.
- Đảm bảo mỗi lần chạy gọi đúng entrypoint **OpsOne** (`cmd/worker-agent`) — không rẽ nhánh chat.
- Timeout & retry cho cả chu kỳ (tránh chồng 2 lần chạy nếu chu kỳ trước chưa xong).

### 4.4 Bảng lịch sử — `agent_analysis_history`

**Mục đích:** Nhìn **xu hướng**, không chỉ số tại một thời điểm.


| Cột            | Kiểu     | Ghi chú                                                      |
| -------------- | -------- | ------------------------------------------------------------ |
| `time`         | datetime | Thời điểm chu kỳ                                             |
| `product`      | string   | VD: ZING                                                     |
| `service_type` | string   | `card`, `topup_data`, hoặc `topup` — quyết định cách routing |
| `sku`          | string   | **Bắt buộc** với `card` / `topup_data`; `null` với `topup`   |
| `provider`     | string   | VD: ESALE                                                    |
| `success`      | number   | % hoặc count tùy chuẩn hóa                                   |
| `pending`      | number   |                                                              |
| `fail`         | number   |                                                              |


**Ví dụ — topup tiền (không có sku):**


| time  | product    | sku | provider | success | pending | fail |
| ----- | ---------- | --- | -------- | ------- | ------- | ---- |
| 10:00 | TOPUP_VINA | —   | ESALE    | 99      | 1       | 0    |
| 10:05 | TOPUP_VINA | —   | ESALE    | 88      | 5       | 7    |


**Ví dụ — thẻ / topup data (bắt buộc có sku):**


| time  | product   | sku   | provider | success | pending | fail |
| ----- | --------- | ----- | -------- | ------- | ------- | ---- |
| 10:00 | ZING      | 20000 | ESALE    | 98      | 1       | 1    |
| 10:05 | ZING      | 20000 | ESALE    | 85      | 8       | 7    |
| 10:10 | DATA_VINA | VNP50 | ESALE    | 72      | 15      | 13   |


> **Thẻ / Topup data:** Cùng provider ESALE có thể ổn ở một SKU/gói nhưng xấu ở SKU khác — history và routing **theo SKU**, không aggregate product để chia tải.

**Checklist lưu trữ:**

- Schema MySQL — xem **§13** (DDL + index + seed).
- Retention (giữ bao nhiêu chu kỳ: VD 24h = 288 bản ghi @ 5 phút; mock 24h @ 1 phút).
- API hoặc tool nội bộ `GetAnalysisHistory(product, provider, sku?, window)` — query bảng `agent_analysis_history`.
- Quy ước: `service_type=topup` → `sku` = `''`; `service_type=card` hoặc `topup_data` → `sku` bắt buộc.

### 4.5 Mock Data Generator — data giả realtime (chạy thử)

Thành phần **tự sinh dữ liệu giả** theo chu kỳ **1 phút** để demo và chạy thử Agent **không cần** nối hệ thống production.

#### 4.5.1 Vai trò


| Mục đích            | Mô tả                                                              |
| ------------------- | ------------------------------------------------------------------ |
| **Demo hackathon**  | Có metric / lỗi / revenue thay đổi liên tục trên UI                |
| **Chạy thử Agent**  | Scheduler + Tools đọc data mới → sinh Incident / Routing Plan thật |
| **Kịch bản inject** | Cố ý làm ESALE (hoặc SKU) xấu dần để test rules                    |


```text
Mỗi 1 phút:
  Mock Generator
    → load cửa sổ bảo trì in-window (scheduled/active, starts_at ≤ now < ends_at)
    → foreach product (catalog §1.1)
    → foreach sku (nếu routing_mode=sku)
    → foreach provider
    → SKIP nếu (product, sku, provider) đang bảo trì — không INSERT mock_metrics / mock_error_stats
    → ngược lại: sinh success / pending / fail / errors / revenue
    → INSERT mock_metrics, mock_error_stats (§13.5)
```

> **Bảo trì & mock:** Scope đang trong cửa sổ bảo trì **không** nhận dòng metric/error mới — tránh nhiễu và giả lập sai (GD vẫn “chạy” trong khi dịch vụ đã BT). Agent đọc metric cũ trong cửa sổ 15p hoặc không có điểm mới; dashboard không bị kéo đỏ oan.

> **Tách biệt 2 chu kỳ:** Generator **1 phút** (làm mới data); Scheduler Agent **3–5 phút** (phân tích). Data tích lũy mỗi phút → Agent thấy xu hướng khi chạy.

#### 4.5.2 Dữ liệu sinh ra mỗi phút


| Loại           | Granularity               | Ví dụ field                                                       |
| -------------- | ------------------------- | ----------------------------------------------------------------- |
| **Metric**     | product × provider × sku? | `success_rate`, `pending_rate`, `fail_rate`, `total_transactions` |
| **Top errors** | product × provider        | `error_code`, `count` (VD `-3004`, `-22`)                         |
| **Revenue**    | product × provider × sku? | `revenue_last_hour` (random walk quanh baseline)                  |
| **Routing**    | theo routing_mode         | Giữ state routing hiện tại (UpdateRouting mock cập nhật store)    |


**Baseline mẫu (ổn định):** success ~95–99%, fail ~1–3%. Generator thêm **nhiễu ±2%** mỗi phút.

#### 4.5.3 Kịch bản inject (scenario)

Chọn trên UI hoặc config — Generator áp dụng xu hướng có chủ đích:


| `mock_scenario` (API) | Nhãn UI (Settings) | Hành vi |
| --------------------- | ------------------ | ------- |
| `normal` | Bình thường — nhiễu nhẹ, không sự cố lớn | Nhiễu ngẫu nhiên nhẹ, không incident lớn |
| `esale_degrading` | ESALE suy giảm — TOPUP_VINA + ZING SKU 20k | ESALE success giảm ~3–5%/phút trên TOPUP_VINA + ZING SKU 20000 |
| `sku_local_fault` | Lỗi cục bộ SKU — DATA_VINA VNP50 / ESALE | Chỉ DATA_VINA / VNP50 trên ESALE xấu; SKU khác ổn |
| `random_spike` | Đột biến lỗi — spike ngẫu nhiên, hồi sau 5–10 phút | Đột ngột fail tăng 1 provider ngẫu nhiên, sau 5–10 phút hồi |
| `imedia_garena_pending` | IMEDIA Garena Pending — Garena 10k qua IMEDIA | Giả lập tỷ lệ pending cao (25-35%) cho Garena 10000 qua IMEDIA |


#### 4.5.4 Config

```json
{
  "data_source": "mock",
  "mock_data": {
    "enabled": true,
    "interval_minutes": 1,
    "scenario": "esale_degrading",
    "retention_hours": 24
  }
}
```


| Field                        | Mặc định                           | Mô tả                                 |
| ---------------------------- | ---------------------------------- | ------------------------------------- |
| `data_source`                | `mock` (dev) / `production` (prod) | Tools đọc store nào                   |
| `mock_data.enabled`          | `true` (dev)                       | Bật/tắt generator                     |
| `mock_data.interval_minutes` | `1`                                | Chu kỳ sinh data (khuyến nghị 1 phút) |
| `mock_data.scenario`         | `normal`                           | Kịch bản inject §4.5.3                |
| `mock_data.retention_hours`  | `24`                               | Xóa data mock cũ                      |


**UI (tab Cấu hình §9.5):** toggle *Bật mock data*, chọn scenario, hiển thị *Lần sinh data cuối* / số bản ghi trong store.

#### 4.5.5 Store & API

**Bảng MySQL:** `mock_metrics`, `mock_error_stats`, `mock_generator_run` — chi tiết **§13.5**.

**API nội bộ:**


| Endpoint              | Mô tả                                     |
| --------------------- | ----------------------------------------- |
| `POST /mock/generate` | Chạy 1 lần sinh data (debug)              |
| `GET /mock/status`    | enabled, last_run, record_count, scenario |
| `PUT /mock/config`    | Đổi scenario / interval                   |


**Tools ở chế độ mock:** GetMetrics / GetTopErrors / GetRevenue đọc **cửa sổ 15m** từ bảng `mock_metrics` / `mock_error_stats` (§13.5).

#### 4.5.6 Checklist triển khai

- Job cron **1 phút** — độc lập scheduler Agent.
- Loop đủ **11 product** × provider × SKU (thẻ + topup data).
- Implement ≥ 2 scenario inject (`normal`, `esale_degrading`).
- `data_source` switch: mock ↔ production không đổi code Agent.
- UI: badge *"Đang dùng MOCK DATA"* khi `data_source=mock`.
- Seed routing ban đầu trong bảng `routing_config` (§13.3).

---

## 5. Bước 2 — OpsOne (Agent core)

### 5.1 Vai trò

Đây là **"bộ não điều phối"** — không tự thay thế toàn bộ logic suy luận, mà:

- Quyết định **gọi tool nào**, **theo thứ tự nào**.
- Gom context cho Reasoning Engine.
- Chọn **loại output** và **mức hành động** (monitor / recommend / execute).

### 5.2 Nhiệm vụ

```text
Đọc service_type / routing_mode của product
    ↓
Thẻ / Topup data (sku): quan sát từng SKU × provider
Topup tiền (provider): quan sát product × provider
    ↓
Phân tích tình trạng → hành động phù hợp
```

**Ví dụ:** Thẻ ZING / VINAPHONE (`card`); DATA_VINA / DATA_MOBI (`topup_data`); TOPUP_VINA / TOPUP_VIETTEL (`topup`) — xem catalog §1.1.

### 5.3 Câu hỏi Agent phải trả lời được (mỗi chu kỳ)

**Chung:**

- Có cần **maintenance** không? (chỉ **1 provider active** — §7.4)

**Nếu `service_type = topup` (nạp tiền):**

- Provider nào **tốt nhất** / **xấu đi**?
- Có cần **chuyển traffic giữa provider** không?

**Nếu `service_type = card` hoặc `topup_data` (routing theo SKU):**

- SKU/gói nào đang **xấu**?
- Trong SKU đó, provider nào kém — cần **shift % giữa provider** cho **SKU/gói đó**?
- SKU khác có bị ảnh hưởng không? (không đổi routing SKU đang ổn)

### 5.4 Mô hình routing theo loại dịch vụ

**Thẻ / Topup data (`routing_mode=sku`) — chỉ routing theo SKU:**

```text
Product ZING (service_type: card)
  ├─ SKU 10.000đ  →  ESALE 60% | IMEDIA 40%
  ├─ SKU 20.000đ  →  ESALE 10% | IMEDIA 90%
  └─ SKU 50.000đ  →  ESALE 50% | SHOPPAY 50%

Product DATA_VINA (service_type: topup_data)
  ├─ SKU VNP20   →  ESALE 70% | IMEDIA 30%
  ├─ SKU VNP50   →  ESALE 20% | IMEDIA 80%   ← chỉ đổi gói này khi lỗi
  ├─ SKU V50K    →  ESALE 50% | IMEDIA 30% | SHOPPAY 20%
  └─ SKU V100K   →  ESALE 40% | SHOPPAY 60%

  ✗ Không có routing aggregate product-level
```

**Topup tiền (`routing_mode=provider`) — chỉ routing theo provider:**

```text
Product TOPUP_VINA (service_type: topup)
  └─ ESALE 70% | IMEDIA 20% | SHOPPAY 10%

  ✗ Không routing theo mệnh giá / SKU
```

**Thứ tự phân tích:**


| Loại                     | Loop thu thập                | UpdateRouting scope         |
| ------------------------ | ---------------------------- | --------------------------- |
| **card**, **topup_data** | product → **sku** → provider | `sku` hoặc `sku_batch` only |
| **topup**                | product → provider           | `provider` only             |


### 5.5 Việc cần làm khi triển khai

- Config `product → service_type` (`card` | `topup_data` | `topup`) và derive `routing_mode` (sku vs provider).
- Map product → provider (GetProviders); **GetSkus gọi khi `routing_mode=sku`** (`card`, `topup_data`).
- Vòng lặp:
  - `routing_mode=sku`: foreach sku → foreach provider
  - `routing_mode=provider`: foreach provider
- **Cấm** UpdateRouting `scope=provider` cho `card` / `topup_data`; **cấm** `scope=sku` cho `topup`.
- Nhánh `active_provider_count == 1` → maintenance (không routing) — kể cả catalog có > 1 dòng nhưng chỉ 1 **active** (§7.4).

---

## 6. Bước 3 — Tools

Implement trong Go: `internal/tools/*.go` — mỗi tool là function nhận `context` + struct input, gọi `internal/store`.

Đây là các **nguồn dữ liệu** mà Agent được phép truy cập. Agent dùng **9 tool** (7 cốt lõi + **GetSkus** + **GetMaintenance** + **SetMaintenance**).

**Nguồn dữ liệu:**


| `data_source` | Nguồn                                           | Khi nào                     |
| ------------- | ----------------------------------------------- | --------------------------- |
| `mock`        | Bảng `mock_metrics`, `mock_error_stats` — §13.5 | Demo / hackathon / chạy thử |
| `production`  | Bảng `metrics_snapshot` hoặc API vận hành thật  | Vận hành thực tế            |


**Tham số chung:**


| Field          | Bắt buộc                                                                    | Mô tả                                                           |
| -------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `product`      | Có                                                                          | VD: `ZING`, `TOPUP_VINA`, `DATA_MOBI` — xem §1.1                |
| `service_type` | Có (config)                                                                 | `card` / `topup_data` → routing SKU; `topup` → routing provider |
| `provider`     | Tuỳ tool                                                                    | VD: `ESALE`                                                     |
| `sku`          | **card / topup_data:** có khi get metric/routing SKU; **topup:** không dùng |                                                                 |


---

### 6.1 GetMetrics

Lấy **metric vận hành**.

**Input — thẻ / topup data (`routing_mode=sku`), bắt buộc có `sku`:**

```json
{ "product": "ZING", "provider": "ESALE", "sku": "20000", "window": "15m" }
```

```json
{ "product": "DATA_VINA", "provider": "ESALE", "sku": "VNP50", "window": "15m" }
```

**Input — topup tiền (`routing_mode=provider`), không có `sku`:**

```json
{ "product": "TOPUP_VINA", "provider": "ESALE", "window": "15m" }
```

**Output:**

```json
{
  "success_rate": 82,
  "pending_rate": 10,
  "fail_rate": 8,
  "total_transactions": 12000
}
```

**Ý nghĩa — Agent dùng để phát hiện:**

- Success giảm
- Pending tăng
- Fail tăng

**Checklist triển khai:**

- `**routing_mode=sku`** (`card`, `topup_data`): metric `GROUP BY product, sku, provider`.
- `**routing_mode=provider**` (`topup`): metric `GROUP BY product, provider`.
- Chuẩn hóa `window` (15m mặc định) và format `sku` (VD luôn VND, không dấu chấm).
- Đăng ký tool trong `internal/agent/tools_registry.go` + mô tả cho LLM prompt (theo `service_type`).

---

### 6.2 GetTopErrors

Lấy **danh sách lỗi phổ biến**.

**Output mẫu:**

```json
[
  { "error": "-3004", "count": 1200 },
  { "error": "-22", "count": 300 }
]
```

**Ý nghĩa:** Agent xác định nguyên nhân.

**Ví dụ:** `-3004` tăng mạnh => **Provider timeout**

**Checklist triển khai:**

- Aggregate log lỗi theo `error_code` + `product` + `provider` + `window`.
- Drill-down lỗi theo `sku` với `**routing_mode=sku`** (thẻ, topup data).
- Trả về top N (VD 5).
- (Tuỳ chọn) Bảng mapping error_code → diễn giải cho LLM.

---

### 6.3 GetProviders

Lấy **danh sách provider** của product — phân biệt **active** vs **inactive**. Agent dùng **`active_count`** (không phải tổng số dòng catalog) để quyết định `suggested_action` (§7.4).

**Nguồn dữ liệu:** `product_providers` JOIN `providers` WHERE `product_providers.enabled = 1` AND `providers.enabled = 1` → **active**; các dòng còn lại → **inactive**.

**Output mẫu — đủ 3 provider active:**

```json
{
  "product": "TOPUP_VINA",
  "active_providers": ["ESALE", "IMEDIA", "SHOPPAY"],
  "inactive_providers": [],
  "active_count": 3,
  "total_count": 3
}
```

**Output mẫu — 2 provider trong catalog, chỉ 1 active (edge case §7.4):**

```json
{
  "product": "ZING",
  "active_providers": ["ESALE"],
  "inactive_providers": ["IMEDIA"],
  "active_count": 1,
  "total_count": 2
}
```

> **Lưu ý:** `total_count = 2` nhưng `active_count = 1` → **maintenance**, không routing — vì không có provider active khác để chuyển traffic.


| Kịch bản | `active_count` | Hàm ý |
| -------- | -------------- | ----- |
| ≥ 2 provider **active** + có backup healthy (Rule 6) | ≥ 2 | Có thể **chuyển traffic** — Routing Plan / UpdateRouting |
| == 1 provider **active** (dù `total_count` có thể > 1) | 1 | Chỉ **maintenance** — không có nơi chuyển traffic |
| ≥ 2 provider **active** nhưng **tất cả đều xấu** (không có backup healthy) | ≥ 2 | **maintenance** — không shift được sang provider tốt hơn |


**Ví dụ:**

- **TOPUP_VINA** — ESALE, IMEDIA, SHOPPAY đều active => routing theo provider khi sự cố
- **ZING (pilot §10.2)** — chỉ **ESALE active** (IMEDIA/SHOPPAY inactive hoặc xóa khỏi catalog) => maintenance
- **ZING (edge §10.2b)** — catalog còn ESALE + IMEDIA nhưng IMEDIA `enabled=0`; ESALE metric xấu => `active_count=1` => maintenance (không routing sang IMEDIA inactive)
- **ZING / GARENA / … (seed đầy đủ)** — 3 provider active => routing theo SKU

**Checklist triển khai:**

- Query `product_providers.enabled` + `providers.enabled`; trả `active_providers[]`, `inactive_providers[]`, `active_count`.
- Agent luôn gọi **trước** `EvaluateThresholds` / quyết định Routing vs Maintenance.
- Admin bật/tắt provider trên UI Settings → cập nhật `product_providers.enabled` (audit log).

---

### 6.4 GetSkus

Lấy **danh sách SKU** — gọi khi `**routing_mode=sku`** (`card`, `topup_data`). Topup tiền **không** dùng tool này.

**Input (thẻ):**

```json
{ "product": "ZING" }
```

**Input (topup data):**

```json
{ "product": "DATA_VINA" }
```

**Output (thẻ):**

```json
{
  "product": "ZING",
  "service_type": "card",
  "skus": [
    { "sku": "10000", "label": "10.000đ" },
    { "sku": "20000", "label": "20.000đ" }
  ]
}
```

**Output (topup data):**

```json
{
  "product": "DATA_VINA",
  "service_type": "topup_data",
  "skus": [
    { "sku": "VNP20", "label": "VNP20" },
    { "sku": "VNP50", "label": "VNP50" },
    { "sku": "V50K", "label": "V50K" },
    { "sku": "V100K", "label": "V100K" }
  ]
}
```

**Ý nghĩa:** Agent biết product có những SKU/gói nào cần quét và routing riêng.

**Checklist triển khai:**

- Catalog SKU từ DB sản phẩm / config ops.
- Product `topup` (tiền) → **bỏ qua** GetSkus trong pipeline.
- Product `card` / `topup_data` → gọi **sau GetProviders**, trước vòng lặp metric theo SKU.

---

### 6.5 GetRouting

Lấy **tỷ lệ routing** hiện tại — scope phụ thuộc `service_type`:

**Output — topup tiền (`scope: provider` only):**

```json
{
  "product": "TOPUP_VINA",
  "service_type": "topup",
  "scope": "provider",
  "routing": { "ESALE": 70, "IMEDIA": 20, "SHOPPAY": 10 }
}
```

**Output — thẻ / topup data (`scope: sku` only):**

```json
{
  "product": "ZING",
  "service_type": "card",
  "scope": "sku",
  "routing_by_sku": {
    "10000": { "ESALE": 60, "IMEDIA": 40 },
    "20000": { "ESALE": 80, "IMEDIA": 20 },
    "50000": { "ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20 }
  }
}
```

**Ý nghĩa:**

- **Topup tiền:** Agent biết traffic đổ vào provider nào (một bảng cho cả product).
- **Thẻ / Topup data:** Agent biết traffic từng SKU/gói đổ vào provider nào — **không** có routing aggregate product-level.

**Ví dụ (`routing_mode=sku`):** SKU `20000` (thẻ) hoặc `VNP50` (topup data): ESALE success = 72% và đang gánh **80%** traffic SKU đó => **Rủi ro cao**; SKU khác trên ESALE vẫn ~98% → không đổi routing SKU đó.

**Checklist triển khai:**

- API routing: `topup` → key `product + provider`; `card` / `topup_data` → key `product + sku + provider`.
- Validate tổng % ≈ 100 **trong từng SKU** (sku-mode) hoặc **toàn product** (topup tiền).
- **Không** fallback routing product-level cho `card` / `topup_data` — mỗi SKU phải có config riêng.

---

### 6.6 GetRevenue

Lấy **doanh thu bị ảnh hưởng**.

**Output — topup tiền:**

```json
{ "product": "TOPUP_VINA", "provider": "ESALE", "revenue_last_hour": 150000000 }
```

**Output — thẻ / topup data (theo SKU):**

```json
{
  "product": "ZING",
  "provider": "ESALE",
  "sku": "20000",
  "revenue_last_hour": 85000000
}
```

**Ý nghĩa:** `**routing_mode=sku`** — ưu tiên SKU có doanh thu lớn. `**routing_mode=provider**` — ưu tiên provider gánh doanh thu cao.

**Checklist triển khai:**

- Nguồn doanh thu theo `product`, `provider`, tuỳ chọn `sku` + cửa sổ (1h / 15m).
- Đưa vào prompt Reasoning để xếp hạng severity.

---

### 6.7 UpdateRouting

**Tool hành động** — đổi tỷ lệ routing. Scope **bắt buộc** theo `service_type`:


| `service_type`           | Scope được phép    | Scope cấm          |
| ------------------------ | ------------------ | ------------------ |
| **topup**                | `provider`         | `sku`, `sku_batch` |
| **card**, **topup_data** | `sku`, `sku_batch` | `provider`         |


**Input — topup tiền (routing theo provider):**

```json
{
  "product": "TOPUP_VINA",
  "service_type": "topup",
  "scope": "provider",
  "routing": { "ESALE": 20, "IMEDIA": 60, "SHOPPAY": 20 }
}
```

**Input — thẻ / topup data (routing theo SKU):**

```json
{
  "product": "DATA_VINA",
  "service_type": "topup_data",
  "scope": "sku",
  "sku": "VNP50",
  "routing": { "ESALE": 10, "IMEDIA": 90 }
}
```

**Input — thẻ (routing theo SKU — mệnh giá 20.000đ):**

```json
{
  "product": "ZING",
  "service_type": "card",
  "scope": "sku",
  "sku": "20000",
  "routing": { "ESALE": 10, "IMEDIA": 90 }
}
```

**Input — thẻ / topup data (cập nhật nhiều SKU một lần):**

```json
{
  "product": "ZING",
  "service_type": "card",
  "scope": "sku_batch",
  "updates": [
    { "sku": "10000", "routing": { "ESALE": 50, "IMEDIA": 50 } },
    { "sku": "20000", "routing": { "ESALE": 10, "IMEDIA": 90 } }
  ]
}
```

**Ý nghĩa:** Topup tiền đổi cả product; thẻ / topup data chỉ đổi SKU/gói bị ảnh hưởng.

**Ví dụ Hiện tại / Đề xuất — topup tiền (provider):**


|              | ESALE | IMEDIA | SHOPPAY |
| ------------ | ----- | ------ | ------- |
| **Hiện tại** | 70%   | 20%    | 10%     |
| **Đề xuất**  | **0%** | 67%    | 33%     |


> Khi fail/pending/số GD lỗi vượt ngưỡng → ESALE **0%** (hard cut), không còn giảm dần còn 10%.

**Ví dụ Hiện tại / Đề xuất — thẻ / topup data, SKU cụ thể (hard cut):**


|              | ESALE | IMEDIA |
| ------------ | ----- | ------ |
| **Hiện tại** | 80%   | 20%    |
| **Đề xuất**  | **0%** | 100%   |


**Chế độ triển khai** — điều khiển **per scope** `(product_code, sku_code)` và **cấp dịch vụ** `(product_code, sku_code="")` trên **Dashboard** (§9.5.2), lưu `routing_scope_state`; Agent + poll overview đọc **`ResolveEffectiveScopeAuto`** (dịch vụ trước, SKU sau):


| `auto_action` **hiệu lực** (§9.5.2) | Hành vi routing + bảo trì (`ShouldAutoApplyScope`) — từ cấu hình **dịch vụ** nếu có row `(product, "")`, else SKU |
| ------------------------ | ---------------------------------------------------------------------------------- |
| `recommend_only` (mặc định) | Sinh Routing Plan / đề xuất bảo trì — **không** gọi `UpdateRouting` / `SetMaintenance` |
| `auto`                   | **Tự động** `UpdateRouting` hoặc `SetMaintenance` khi rules thỏa — **không** hiện hàng đề xuất trên Dashboard |
| `time_window`            | **Tự động** chỉ khi `now` ∈ `[window_start, window_end)`; ngoài giờ → chỉ đề xuất (giống `recommend_only`) |


Ngoài khung giờ (mode `time_window`, VD `now` trước `window_start` hoặc sau `window_end`) hoặc mode `recommend_only`: Routing Plan / đề xuất bảo trì chờ **Duyệt** trên UI — kể cả plan DB tạo lúc ngoài khung vẫn hiện cho đến khi vào khung (poll auto apply) hoặc admin duyệt. Khi `ShouldAutoApplyScope=true`, API `GET /dashboard/overview` **không** trả `pending_plan` / `pending_maintenance`; poll overview + Agent tự apply (force routing/bảo trì §9.5.2).

**Checklist triển khai:**

- API cập nhật routing production (hoặc mock trong giai đoạn hackathon).
- API routing nhận `scope` + `sku` (matrix config).
- Guardrail: validate scope khớp `service_type` trước khi gọi API.
- Guardrail: không đổi quá X% một lần; validate 100% **per scope** (per sku hoặc per topup product).
- Không cascade nhầm: đổi SKU 20k **không** tự động đổi SKU 10k trừ khi `sku_batch` / rule rõ ràng.
- Agent kiểm tra `ShouldAutoApplyScope` (`routing_scope_state.auto_action` + `window_start`/`window_end`) trước mỗi lần gọi `UpdateRouting` hoặc `SetMaintenance`.
- **Trước khi ghi DB:** snapshot `routing_config` hiện tại → `routing_before` (JSON đủ provider × %).
- **Sau khi apply:** ghi `agent_change_log` — `trigger_type`: `auto` | `admin_approve` | `manual_temp` | `manual_baseline` (§8.7); kèm `cycle_id`, `routing_plan_id`, `incident_id` nếu có.
- **Đóng sự cố (§8.3.1):** sau apply thành công → cập nhật `incidents.status`, `handled_by`, `handled_at`, `resolution_action` (duyệt/auto → `resolved`; từ chối plan → `acknowledged`).
- Khôi phục routing: **Mở lại provider** / **Mở lại dịch vụ** (§8.7) — không dùng rollback API.

---

### 6.8 GetMaintenance

Đọc **cửa sổ bảo trì** đang active hoặc sắp diễn ra — gọi **đầu mỗi chu kỳ** trước khi phân tích metric (§8.5).

**Input:**

```json
{ "product": "ZING", "provider": "ESALE", "sku": "" }
```

**Output:** xem §8.5. Query `maintenance_windows` WHERE `status IN ('scheduled','active')` (theo `product`/`provider`/`sku`); **phân loại active/scheduled trong Go** bằng `MaintenanceInWindow` — **không** lọc `ends_at > NOW()` trong SQL (tránh lệch TZ với Dashboard §9.0).

**Checklist:**

- Parameterized query theo `product_code`, `provider_code`, `sku_code`.
- Agent skip cảnh báo metric nếu có cửa sổ `active` bao phủ `now()`.

---

### 6.9 SetMaintenance

**Tool hành động** — tạo cửa sổ bảo trì có **starts_at** và **ends_at**. Dùng khi **`active_provider_count == 1`** hoặc **không có provider active backup healthy** để routing — không thể chuyển traffic sang provider khác.

**Input:** xem §8.5 JSON.

**Validate:**

- `ends_at > starts_at` (không giới hạn thời lượng).
- Không overlap cửa sổ `scheduled`/`active` cùng scope.

**Checklist:**

- Ghi `maintenance_windows`; liên kết `cycle_id`, `incident_id` nếu có.
- `recommend_only` → `status=pending_approve`; auto → `scheduled` hoặc `active`.
- Không gọi UpdateRouting trên cùng product khi maintenance active.

---

### 6.10 Checklist tổng — Tools

- **9 tool** trong `internal/tools/` (7 cốt lõi + GetMaintenance + SetMaintenance).
- Chế độ `data_source=mock`: Tools query bảng §13.5.
- Test từng tool độc lập; sau đó ghép Mock Generator (1 phút) + Scheduler Agent.

---

## 7. Bước 4 — Reasoning Engine

Đây là nơi **LLM suy luận**.

### 7.1 Vai trò

- Hợp nhất: metrics + errors + routing + revenue + **history**.
- Áp dụng **rules** (có thể hard-code trước, LLM bổ sung diễn giải).
- Ra **quyết định**: monitor | incident | shift traffic | maintenance.

### 7.2 Input

```json
{
  "success": 82,
  "pending": 10,
  "fail": 8,
  "routing": {
    "ESALE": 70
  },
  "top_error": "-3004"
}
```

**Input mở rộng khi triển khai** (thêm history, revenue):

```json
{
  "product": "ZING",
  "provider": "ESALE",
  "sku": "20000",
  "success": 72,
  "pending": 15,
  "fail": 13,
  "routing": { "ESALE": 80, "IMEDIA": 20 },
  "top_error": "-3004",
  "history": [
    { "time": "10:00", "success": 98, "fail": 1 },
    { "time": "10:05", "success": 85, "fail": 7 },
    { "time": "10:10", "success": 72, "fail": 13 }
  ],
  "revenue_last_hour": 85000000
}
```

> Rule 7–8 áp dụng `routing_mode=sku` (`card`, `topup_data`). Topup tiền dùng Rule 1–6 ở mức product × provider.

### 7.3 Rules — Agent phân tích

**Tổng quan:**


| #   | Rule                                                | Gợi ý điều kiện triển khai                                                            |
| --- | --------------------------------------------------- | ------------------------------------------------------------------------------------- |
| 1   | Success giảm liên tục                               | ≥ 2 chu kỳ giảm                                                                     |
| 2   | Pending tăng liên tục                               | ≥ 2 chu kỳ tăng                                                                     |
| 3   | Fail tăng đột biến                                  | Δ fail > ngưỡng so với chu kỳ trước                                                   |
| 4   | Top Error thay đổi                                  | Mã lỗi #1 đổi hoặc count tăng mạnh                                                    |
| 5   | Provider đang nhận quá nhiều traffic                | routing% cao + metric xấu (topup tiền: product-level; sku-mode: trong SKU)            |
| 6   | Provider backup có chất lượng tốt hơn               | success_backup > success_primary + margin                                             |
| 7   | **SKU lệch chất lượng** (`routing_mode=sku`)        | Một `sku` success giảm / fail tăng trong khi SKU khác cùng provider vẫn ổn            |
| 8   | **SKU gánh traffic không cân** (`routing_mode=sku`) | routing% cao trên SKU cụ thể + metric xấu tại sku đó                                  |
| 9   | **Doanh thu ưu tiên**                               | sku-mode: sku revenue cao + metric xấu; topup tiền: provider revenue cao + metric xấu |


#### 7.3.1 Định nghĩa rules (machine-readable — AI codegen)

> Mỗi rule là 1 hàm thuần `(input) → RuleResult`. Engine duyệt theo `priority`, gộp tất cả `triggered=true` thành `evidence[]` để LLM tóm tắt. **Rule không tự gọi tool** — input đã chứa metric/history/routing/revenue.

```yaml
rules:
  - id: R1_SUCCESS_DECLINE
    name: "Success giảm liên tục"
    priority: 10
    applies_to: [card, topup_data, topup]
    severity_hint: medium
    inputs: [history.success_rate]
    condition: |
      len(history) >= 2
      AND history[-1].success_rate < history[-2].success_rate
      AND (history[-2].success_rate - history[-1].success_rate) >= 5
    action: contribute_to_incident
    output:
      tag: "success_decline"
      message_vi: "Tỷ lệ thành công giảm {delta}% trong 2 chu kỳ"
      severity_bump: +1

  - id: R2_PENDING_RISE
    name: "Pending tăng liên tục"
    priority: 10
    applies_to: [card, topup_data, topup]
    severity_hint: medium
    inputs: [history.pending_rate, thresholds.pending_rate_max_pct]
    condition: |
      len(history) >= 2
      AND history[-1].pending_rate > history[-2].pending_rate
    action: contribute_to_incident
    output:
      tag: "pending_rise"
      message_vi: "Pending tăng {pending_now}% (2 chu kỳ liên tiếp)"

  - id: R3_FAIL_SPIKE
    name: "Fail tăng đột biến"
    priority: 20
    applies_to: [card, topup_data, topup]
    severity_hint: high
    inputs: [metric.fail_rate, history.fail_rate, thresholds.fail_rate_max_pct]
    condition: |
      metric.fail_rate > thresholds.fail_rate_max_pct
      OR (history[-2] != null AND (metric.fail_rate - history[-2].fail_rate) >= 5)
    action: contribute_to_incident
    output:
      tag: "fail_spike"
      message_vi: "Tỷ lệ lỗi {fail_now}% (Δ +{delta}% so chu kỳ trước)"
      severity_bump: +2

  - id: R4_TOP_ERROR_SHIFT
    name: "Top error thay đổi"
    priority: 15
    applies_to: [card, topup_data, topup]
    severity_hint: medium
    inputs: [top_errors, history.top_errors]
    condition: |
      history.top_errors[0].code != top_errors[0].code
      OR top_errors[0].count >= 2 * history.top_errors[0].count
    action: contribute_to_incident
    output:
      tag: "top_error_shift"
      message_vi: "Lỗi đầu bảng đổi sang {code} ({count} lượt)"

  - id: R5_PROVIDER_OVERLOAD
    name: "Provider gánh quá nhiều traffic + xấu"
    priority: 30
    applies_to: [card, topup_data, topup]
    severity_hint: high
    inputs: [routing.traffic_pct, metric.success_rate, thresholds.success_rate_min_pct]
    condition: |
      routing.traffic_pct[provider] >= 60
      AND metric.success_rate < thresholds.success_rate_min_pct
    action: suggest_routing_shift
    output:
      tag: "provider_overload"
      message_vi: "Provider {provider} gánh {pct}% nhưng success {success}%"

  - id: R6_BACKUP_HEALTHIER
    name: "Provider backup chất lượng tốt hơn"
    priority: 30
    applies_to: [card, topup_data, topup]
    severity_hint: medium
    inputs: [metric.success_rate per provider, agent_settings.routing_good_threshold_pct]
    condition: |
      EXISTS backup IN active_providers WHERE
        backup != current_bad_provider
        AND backup.success_rate >= current_bad_provider.success_rate + 10
        AND backup.success_rate >= agent_settings.routing_good_threshold_pct
    action: suggest_routing_shift
    output:
      tag: "backup_healthier"
      message_vi: "{backup} success {pct}% > {bad} success {bad_pct}%"

  - id: R7_SKU_QUALITY_DIVERGENCE
    name: "SKU lệch chất lượng"
    priority: 25
    applies_to: [card, topup_data]   # routing_mode=sku
    severity_hint: medium
    inputs: [metric.success_rate per sku, thresholds.success_rate_min_pct]
    condition: |
      EXISTS sku_bad IN skus WHERE
        sku_bad.success_rate < thresholds.success_rate_min_pct
      AND EXISTS sku_ok IN skus WHERE
        sku_ok.provider == sku_bad.provider
        AND sku_ok.success_rate >= thresholds.success_rate_min_pct + 10
    action: suggest_routing_shift_per_sku
    output:
      tag: "sku_quality_divergence"
      message_vi: "SKU {sku_bad} xấu nhưng SKU khác cùng provider vẫn ổn"

  - id: R8_SKU_TRAFFIC_IMBALANCE
    name: "SKU gánh traffic không cân"
    priority: 25
    applies_to: [card, topup_data]
    severity_hint: medium
    inputs: [routing.traffic_pct per sku, metric.success_rate per sku]
    condition: |
      EXISTS sku WHERE
        routing.traffic_pct[sku][provider] >= 70
        AND metric.success_rate[sku][provider] < thresholds.success_rate_min_pct
    action: suggest_routing_shift_per_sku
    output:
      tag: "sku_traffic_imbalance"
      message_vi: "SKU {sku} dồn {pct}% vào {provider} đang lỗi"

  - id: R9_REVENUE_PRIORITY
    name: "Doanh thu ưu tiên"
    priority: 5
    applies_to: [card, topup_data, topup]
    severity_hint: high
    inputs: [revenue.last_hour, metric (any rule trên đã trigger)]
    condition: |
      revenue.last_hour >= 50_000_000   # 50M VND/h (configurable)
      AND any(R1..R8 triggered for same scope)
    action: bump_severity
    output:
      tag: "high_revenue_impact"
      message_vi: "Doanh thu ảnh hưởng {revenue} VND/h — ưu tiên xử lý cao"
      severity_bump: +1
```

**Kiểu trả về `RuleResult` (Go):**

```go
type RuleResult struct {
    RuleID        string          // "R3_FAIL_SPIKE"
    Triggered     bool
    Tag           string          // "fail_spike"
    MessageVi     string          // text tiếng Việt đã render
    SeverityBump  int             // -1..+2
    SuggestedAction string        // "" | "routing" | "maintenance"
    Evidence      map[string]any  // {"fail_now": 13, "delta": 6}
}
```

**Pipeline engine (deterministic, không LLM):**

```text
EvaluateThresholds() → if not breached → return state=NORMAL/WARNING, không chạy rules

for rule in rules ORDER BY priority DESC:
    if rule.applies_to includes product.service_type:
        result = rule.evaluate(input)
        if result.triggered:
            evidence.append(result)
            severity += result.severity_bump

decision = build_decision(evidence, suggested_action_from_§7.4)
LLM (optional, §7.6) → human-readable summary using evidence[] only
```

**Sau đó Agent đưa ra quyết định** — combine `EvaluateThresholds.suggested_action` (§7.4) với `rules.evidence[]` (§7.3.1).

### 7.4 Đánh giá ngưỡng theo dịch vụ (`internal/threshold`)

**Input:** metric hiện tại + `product_alert_thresholds` + defaults `agent_settings`.

**Output:**

```json
{
  "product_code": "TOPUP_VINA",
  "breached": true,
  "breach_reasons": [
    "Tỷ lệ lỗi 12% vượt ngưỡng 10%",
    "Số GD lỗi 142 vượt ngưỡng 120"
  ],
  "consecutive_breach_cycles": 2,
  "required_cycles": 2,
  "should_act": true,
  "should_alert_email": true,
  "active_provider_count": 3,
  "healthy_backup_count": 2,
  "suggested_action": "routing",
  "suggested_action_reason": "3 provider active; IMEDIA/SHOPPAY healthy — có thể chuyển traffic khỏi ESALE"
}
```

**Quy tắc `suggested_action` (sau khi `breached` + đủ `consecutive_cycles`):**

> **Tiêu chí cốt lõi:** đếm **provider active** (`GetProviders.active_count`), **không** đếm tổng dòng catalog.
> Provider **inactive** (`product_providers.enabled = 0` hoặc `providers.enabled = 0`) **không** được coi là backup để routing.

**Cây quyết định:**

```text
GetProviders(product)
  active_count = |active_providers|

IF active_count == 1
  → suggested_action = "maintenance"
  → Lý do: chỉ 1 luồng provider đang chạy — kể cả catalog có 2+ dòng nhưng 1 inactive
  → VD: ZING có ESALE (active) + IMEDIA (inactive), ESALE lỗi/pending → bảo trì

ELSE IF active_count > 1
  healthy_backup = active providers khác (trong scope SKU/provider) có success_rate ≥ good_threshold (§8.6.3)
  IF healthy_backup_count >= 1
    → suggested_action = "routing"
    → Chuyển traffic sang provider active còn lại (Rule 5, 6, §8.6.3)
  ELSE
    → suggested_action = "maintenance"
    → Lý do: nhiều provider active nhưng tất cả đều xấu — không có nơi shift an toàn

IF scope.auto_action == "recommend_only" OR (scope.auto_action == "time_window" AND now ∉ window)
  → Chỉ output (Incident + Plan/Maintenance), không auto execute UpdateRouting
IF scope.auto_action IN ("auto", "time_window" trong giờ)
  → Có thể gọi UpdateRouting trực tiếp (trigger_type=auto) khi rules thỏa
```


| # | Điều kiện | `suggested_action` | Ghi chú |
|---|-----------|-------------------|---------|
| 1 | `active_count == 1` | `maintenance` | Catalog có thể > 1 provider nhưng chỉ 1 **active** |
| 2 | `active_count > 1` AND `healthy_backup_count >= 1` | `routing` | Có provider active khác đủ tốt để nhận traffic |
| 3 | `active_count > 1` AND `healthy_backup_count == 0` | `maintenance` | Tất cả provider **đang routing** (`routing_pct>0`) đều vi phạm — đề xuất bảo trì SKU (§9.0 `SKURoutingDecision`) |
| 4 | `auto_action = recommend_only` trên scope | (giữ action trên) | Chỉ đề xuất; admin approve thủ công |


**Ví dụ edge case — 2 provider catalog, 1 inactive, provider active còn lại xấu:**

```text
Product ZING:
  product_providers: ESALE (enabled=1), IMEDIA (enabled=0)
  GetProviders → active_count=1, inactive_providers=["IMEDIA"]
  ESALE: fail 35%, pending 22% — vượt ngưỡng
  → suggested_action = "maintenance"   ← KHÔNG routing sang IMEDIA (inactive)
  → Incident + Đề xuất bảo trì ESALE
```

**Ví dụ routing — 3 provider active, 1 xấu:**

```text
Product TOPUP_VINA:
  active_providers: ESALE, IMEDIA, SHOPPAY (cả 3 enabled)
  ESALE fail 12%; IMEDIA success 98%; SHOPPAY success 97%
  → active_count=3, healthy_backup_count=2
  → suggested_action = "routing"
  → Routing Plan: giảm ESALE, tăng IMEDIA/SHOPPAY
```


**Liên kết health_status (§8.2):**

- `breached` + `consecutive_breach_cycles >= required` → tối thiểu 🟡; nếu fail_rate hoặc fail_txn vượt mạnh → 🔴.
- Ghi `breach_reasons` vào Incident summary và email §8.9.

**Checklist:**

- `EvaluateThresholds` gọi **sau** `GetProviders` — dùng `active_count`, không `total_count`.
- `EvaluateThresholds` gọi **trước** rules 1–9; rules bổ sung xu hướng (chu kỳ liên tiếp).
- Đếm `fail_txn_count` = `total_transactions * fail_rate / 100` hoặc sum từ production feed.
- `error_event_count` = `SUM(error_count)` từ `mock_error_stats` tại **`recorded_at` trùng snapshot GetMetrics** (`SumErrorEventsAtRecordedAt`) — **không** SUM toàn cửa sổ 15 phút khi metric là latest row (§2.3.2).
- Lưu `threshold_breach_log` (tuỳ chọn) hoặc embed trong `agent_analysis_cycles.decision` JSON.

### 7.5 Việc cần làm khi triển khai

- Tách **rules engine** (code) vs **LLM** (tóm tắt + lý do human-readable).
- Ngưỡng (threshold) đưa vào `**product_alert_thresholds`** + fallback `agent_settings` — không hard-code trong prompt.
- Output reasoning: `decision`, `confidence`, `evidence[]`, `**health_status**` cho audit — **text tiếng Việt**.
- Trường hợp **không đủ dữ liệu** → Recommendation: monitor thêm 15 phút.

### 7.6 Prompt Engineering Spec (LLM)

> **Vai trò LLM trong OpsOne — RẤT HẸP:**
>
> 1. **Tóm tắt** rules-engine output thành câu tiếng Việt cho ops (Incident summary, Routing Plan reason, email body).
> 2. **Trả lời chat / voice** on-demand từ ops (giải thích incident, đọc trạng thái).
>
> **LLM TUYỆT ĐỐI KHÔNG:**
> - Tự quyết định routing %, thuật toán shift đã ở §8.6.3 (code).
> - Tự đánh giá ngưỡng — đã ở §7.4 (code).
> - Tự gọi tool mà không có rules trigger trước.
> - Hallucinate metric/provider/SKU không có trong context.
>
> Mọi quyết định **đỏ/vàng/xanh + suggested_action** do code (rules + threshold engine) quyết định. LLM **chỉ viết lại** evidence cho người đọc.

#### 7.6.1 System Prompt (chung cho mọi LLM call)

File: `internal/reasoning/prompts/system_vi.txt`

```text
Bạn là OpsOne — trợ lý vận hành traffic cho hệ thống thanh toán/topup tại Việt Nam.

NGUYÊN TẮC BẮT BUỘC:
1. Luôn trả lời bằng tiếng Việt, văn phong ngắn gọn cho team ops (≤ 3 câu cho summary).
2. CHỈ dùng dữ liệu trong context (CONTEXT block) hoặc kết quả tool đã trả về.
   KHÔNG tự bịa số liệu, mã lỗi, tên provider, SKU không có trong context.
3. KHÔNG tự quyết định routing % — chỉ tóm tắt kế hoạch đã được rules engine sinh ra.
4. KHÔNG đề xuất bảo trì nếu rules engine không yêu cầu.
5. Nếu thông tin không đủ → trả lời rõ "Chưa đủ dữ liệu, theo dõi thêm 15 phút".
6. Mọi tên kỹ thuật giữ nguyên: ESALE, IMEDIA, SHOPPAY, ZING, TOPUP_VINA, SKU 20000…
7. Khi đề cập trạng thái dùng nhãn tiếng Việt: 🟢 Hệ thống OK, 🟡 Đang theo dõi/xử lý, 🔴 Đang có vấn đề.

KHÔNG dùng các cụm tiếng Anh: "Maintenance Recommended", "Shift traffic", "Severity: High",
"Routing Plan pending", "degrading" — thay bằng: Đề xuất bảo trì, Chuyển traffic,
Mức Cao, Kế hoạch routing chờ duyệt, suy giảm.

KHI ĐƯỢC YÊU CẦU OUTPUT JSON: chỉ trả về JSON hợp lệ, không markdown wrapper, không giải thích.
```

#### 7.6.2 Task Prompt — Sinh Incident summary

File: `internal/reasoning/prompts/task_incident_summary.txt`

```text
{{system}}

NHIỆM VỤ: Viết `summary` (1–2 câu tiếng Việt) cho Incident dựa trên evidence từ rules engine.

CONTEXT:
{
  "product": "{{product_code}} ({{product_label}})",
  "provider": "{{provider_code}}",
  "sku": "{{sku_code}}" ,                 // có thể rỗng
  "severity": "{{severity}}",             // low | medium | high
  "metric": {
    "success_rate": {{success_rate}},
    "pending_rate": {{pending_rate}},
    "fail_rate": {{fail_rate}}
  },
  "breach_reasons": {{breach_reasons_json}},   // mảng từ §7.4
  "evidence": {{rule_evidence_json}},           // mảng từ §7.3.1
  "top_error": "{{top_error_code}}"             // VD "-3004"
}

YÊU CẦU OUTPUT (JSON đúng schema):
{
  "summary": "string — 1–2 câu tiếng Việt",
  "headline": "string — 1 dòng ≤ 60 ký tự, đặt cho card UI"
}

VÍ DỤ:
Input metric: success=72, fail=13, top_error=-3004, evidence có "fail_spike" + "provider_overload"
Output:
{
  "summary": "TOPUP_VINA qua ESALE tỷ lệ lỗi 13% (timeout -3004), provider đang gánh 70% traffic — cần chuyển bớt.",
  "headline": "TOPUP_VINA · ESALE lỗi 13% (timeout)"
}
```

#### 7.6.3 Task Prompt — Sinh Routing Plan reason

File: `internal/reasoning/prompts/task_routing_plan_reason.txt`

```text
{{system}}

NHIỆM VỤ: Viết đoạn `reason` (2–4 câu) giải thích vì sao đề xuất routing này.

CONTEXT:
{
  "product": "{{product_code}}",
  "scope": "{{scope}}",                     // provider | sku | sku_batch
  "sku": "{{sku_code}}",
  "current_routing":  {{current_routing_json}},   // {ESALE: 70, IMEDIA: 20, SHOPPAY: 10}
  "suggested_routing": {{suggested_routing_json}},// {ESALE: 0, IMEDIA: 67, SHOPPAY: 33} — hard cut khi fail/pending/txn breach
  "provider_metric":  {{provider_metric_json}},   // success_rate per provider
  "expected_success_improvement_pct": {{delta}},
  "evidence_tags": {{evidence_tags_json}}         // ["provider_overload","backup_healthier"]
}

QUY TẮC NGÔN NGỮ:
- Không bịa số. Nếu suggested_routing có ESALE=0 thì viết "cắt ESALE về 0%"; provider healthy nhận phần còn lại (có thể 100% nếu chỉ còn một nhà).
- Nhắc lý do tiếng Việt: "ESALE đang suy giảm", "IMEDIA chất lượng tốt hơn".
- Kết thúc bằng dự đoán: "Dự kiến tỷ lệ thành công {{current_success}}% → {{expected_success}}%".

YÊU CẦU OUTPUT:
{
  "reason": "string — tiếng Việt, 2–4 câu"
}
```

#### 7.6.4 Task Prompt — Email body §8.9

File: `internal/reasoning/prompts/task_email_body.txt`

```text
{{system}}

NHIỆM VỤ: Viết phần "Hành động OpsOne vừa thực hiện" và "Tin nhắn mẫu gửi provider"
cho email §8.9. KHÔNG tự bịa group chat name — dùng nguyên `chat_escalation`.

CONTEXT:
{
  "trigger_event": "{{routing_applied|maintenance_active}}",
  "product": "{{product_code}} ({{product_label}})",
  "provider": "{{provider_code}}",
  "routing_before": {{before_json}},
  "routing_after":  {{after_json}},
  "maintenance":    {{maintenance_json_or_null}},
  "chat_escalation": {                          // copy nguyên text từ DB
    "chat_app_name":   "{{chat_app}}",
    "chat_group_name": "{{chat_group}}",
    "mention_tags":    "{{tags}}"
  }
}

OUTPUT (plain text — KHÔNG markdown):
{
  "action_summary": "string — 1–2 dòng tóm tắt routing/bảo trì đã làm",
  "sample_chat_message": "string — tiếng Việt 2–3 câu, có chứa {{tags}} ở đầu"
}
```

#### 7.6.5 Chat Agent — On-demand chat/voice (§9) ✅ triển khai

**Code:** `internal/api/chat_agent.go`, `internal/api/chat_metrics.go`, `internal/api/chat_maintenance.go`, `internal/api/chat_actions.go`, `internal/api/chat_commands.go`, `internal/api/chat_turn.go`, `internal/chatresolve/` (`aliases.go`, `intent.go`, `commands.go`, `sku.go`), `internal/tools/metrics_format.go`, `internal/tools/maintenance_format.go`, `internal/tools/maintenance_build.go`, `internal/store/chat_intent_stats.go`, `internal/store/chat_log.go`, `internal/llm/client.go` — `POST /api/v1/chat`:

```json
{
  "message": "string (bắt buộc)",
  "session_id": "UUID client (khuyến nghị — map session in-memory + DB)",
  "user_display_name": "string? — tên hiển thị bubble + system prompt",
  "input_source": "text | voice (mặc định text)",
  "stt_raw": "string? — transcript STT gốc khi voice"
}
```

→ `{ "reply", "session_id" }`. **502** `llm_error` nếu MaaS lỗi (chỉ path LLM).

**Thứ tự routing (`chatAgentReply`):** `DetectChatIntent` → **`tryChatMetricsReply`** (§7.6.5.2) → **`tryChatCommandReply`** (§7.6.5.4) → **`tryChatMaintenanceReply`** (§7.6.5.1) → LLM tool loop (hoặc stub). Sau mỗi lượt: ghi **`chat_intent_stats`** (§7.6.5.3) + persist **`chat_sessions` / `chat_messages` / `chat_interaction_log`** (§7.6.5.5 P1–P2).

| Trạng thái | Hành vi |
|------------|---------|
| **Câu hỏi metric / GD pending** (§7.6.5.2) | **Luôn** tra DB trực tiếp (`tryChatMetricsReply`) — **không** chuyển sang LLM; metric = `provider_metrics` Dashboard (cửa sổ 15m) |
| **Câu hỏi bảo trì** (§7.6.5.1) | **Luôn** tra DB trực tiếp (`tryChatMaintenanceReply`) — **không** chuyển sang LLM; dữ liệu = `GET /dashboard/overview` |
| `LLM_API_KEY` **có** | Agent loop OpenAI-compatible (GreenNode AIP), tool calling tối đa **6 vòng**; session in-memory tối đa **40** message/`session_id` |
| `LLM_API_KEY` **trống** | Fallback keyword stub (`handlers.go` — health/incident/**bảo trì** cơ bản) |

**System prompt** (`chatSystemPrompt`): tiếng Việt; admin vs non-admin; **OpsOne** = tên chính thức, **Zalopay** = alias giọng nói (`chatresolve.AssistantIdentityHint`); **user đổi tên/avatar/xưng hô** → đồng ý (phần lớn xử lý client §9.2.1); danh mục product load từ DB + gợi ý viết tắt; quy tắc mobi/vina/viettel là **dịch vụ** không phải provider.

**Alias dịch vụ (`internal/chatresolve`):** trước khi gọi tool, args được `NormalizeToolArgs` — map tên ops sang `product_code` / provider routing.

| Viết tắt user (ví dụ) | `product_code` |
|------------------------|----------------|
| topup mobi, nap mobifone, mobi topup | `TOPUP_MOBI` |
| topup vina, nap vinaphone | `TOPUP_VINA` |
| topup viettel | `TOPUP_VIETTEL` |
| thẻ zing, zing | `ZING` |
| thẻ garena, garena | `GARENA` |
| thẻ mobi, mobifone | `MOBIFONE` |
| data mobi / data vina / data viettel | `DATA_MOBI` / `DATA_VINA` / `DATA_VIETTEL` |

| Provider routing (không phải nhà mạng) | `ESALE`, `IMEDIA`, `SHOPPAY` |

**Gộp lỗi LLM:** `product=topup` + `provider=mobi` → `TOPUP_MOBI` (bỏ provider sai). `get_metrics` **không** truyền provider → backend trả metric **cả 3** provider (ESALE/IMEDIA/SHOPPAY).

**Tool tra cứu** (mọi user `Ops`):

| Tool | Mô tả |
|------|--------|
| `get_metrics` | success/pending/fail + GD; `product` viết tắt OK; `provider` tuỳ chọn (bỏ trống = cả 3) |
| `get_top_errors` | mã lỗi top |
| `get_routing` | tỉ lệ routing product (chỉ cần product — phù hợp câu hỏi tổng quan topup) |
| `get_maintenance` | cửa sổ BT active/scheduled — **chỉ cần `product`**; `provider`/`sku` tuỳ chọn; trả `summary_vi`, `by_sku`, `active_count` |
| `get_incidents` | sự cố gần đây |
| `list_pending_actions` | routing plan + đề xuất bảo trì chờ duyệt |

##### 7.6.5.2 Tra cứu metric / GD pending — direct reply (khớp Dashboard)

**Vấn đề đã xử lý:** Câu *"thẻ Mobifone 50.000 … quay đơn … banding"* bị nhận nhầm là bảo trì (`IsMaintenanceScopeQuery` chỉ vì có *thẻ + mệnh giá*) → trả *"không có bảo trì"* thay vì **29 GD pending ESALE** trên Dashboard.

**Luồng (`chat_metrics.go` → `tryChatMetricsReply`):**

1. `chatresolve.ShouldLookupMetrics` — true khi:
   - Có token metric/pending (`pending`, `GD pending`, `quay đơn`, `banding`, `treo`, `kẹt đơn`, `success`, `fail`, …), **hoặc**
   - **Follow-up** sau câu hỏi metric trong cùng `session_id`.
2. `ExtractProductFromText` / `ExtractSKUFromText` + `NormalizeSKU`; fallback lịch sử session. Thẻ/topup data (`routing_mode=sku`) **bắt buộc** SKU.
3. **`metricsForChat(product, sku)`** — **cùng nguồn Dashboard:**
   - `GetAgentSettings` → `data_source` (`mock` / `production`);
   - `GetMetricsInWindow` per provider `ESALE` / `IMEDIA` / `SHOPPAY` (cửa sổ **15m**);
   - Tính `pending_txn` / `fail_txn` = `round(total × rate / 100)`;
   - So ngưỡng `GetProductThreshold` → `ScopeBreachedFromSnapshot` / `SnapshotBreachReasons`.
4. **`FormatMetricsReply`** — trả lời tiếng Việt theo provider; nhấn GD pending khi câu hỏi pending-focused.
5. **Không fall-through LLM** — kể cả lỗi DB hoặc thiếu product/SKU → message trực tiếp.

**Ưu tiên intent:** `DetectChatIntent` chọn **metrics trước maintenance**; `IsMaintenanceScopeQuery` **bỏ qua** khi `IsMetricsQuery`.

**Nhận biết deploy đúng:** câu trả lời dạng `**MOBIFONE 50000** — GD pending (cửa sổ 15m):` + `**ESALE**: **29** GD pending` — **không** có *"không có bảo trì"*.

**Ví dụ metric:**

```text
USER: "hiện tại thẻ Mobifone 50.000 đi qua rồi quay đơn có đang bị banding không"
→ MOBIFONE / sku=50000 → ESALE 29 GD pending (3.6%), cảnh báo vượt ngưỡng pending_txn.

USER: "TOPUP_MOBI ESALE pending thế nào?"
→ get_metrics path LLM nếu không khớp direct; hoặc direct khi có token pending + product.
```

##### 7.6.5.1 Tra cứu bảo trì — direct reply (khớp Dashboard)

**Vấn đề đã xử lý:** LLM tự kết luận sai (vd *"provider ESALE không có bảo trì"* trong khi Dashboard hiện BT); gọi tool với `provider`/`sku` sai (`10.000` ≠ `10000`); follow-up không có từ *bảo trì*; lệch múi giờ khi SQL dùng `ends_at > NOW()` thay vì lọc `time.Now()` như overview; **cả câu hỏi bị upper-case thành tên product giả** khi không nhận diện alias (`ExtractProductFromText` trả `""` thay vì upper-case toàn bộ message).

**Luồng (`chat_maintenance.go` → `tryChatMaintenanceReply`):**

1. `chatresolve.ShouldLookupMaintenance(userMsg, sessionHistory)` — true khi:
   - **Không** phải câu metric (§7.6.5.2), **và**
   - Có token bảo trì (`bảo trì`, `có bảo trì`, `đang bảo trì`, `đang bt`, `maintenance`, …), **hoặc**
   - Có **thẻ + mệnh giá** (`IsMaintenanceScopeQuery`: *thẻ Mobifone 10.000* — **chỉ** khi không phải câu pending/metric), **hoặc**
   - **Follow-up** sau câu hỏi bảo trì trong cùng `session_id`, **hoặc**
   - Câu **toàn hệ thống** (`IsGlobalMaintenanceQuery`: *ngoài ra còn dịch vụ nào đang bảo trì*).
2. `ExtractProductFromText` / `ExtractSKUFromText` + `NormalizeSKU` (`10.000` → `10000`); fallback lịch sử session.
3. **`maintenanceForChat(product, sku?)`** — **cùng nguồn Dashboard:**
   - `ListMaintenanceWindows(product, status='active', limit=500)`;
   - Lọc in-window trong Go: `!ends.Before(now) && !starts.After(now)` (`MaintenanceInWindow` — §9.0);
   - Tuỳ chọn lọc `sku_code`; thêm `scheduled` tương lai.
4. **`FormatMaintenanceReply`** / **`FormatAllMaintenanceReply`** (câu toàn hệ thống, `product=""`) — trả lời tiếng Việt theo SKU/provider.
5. **Không fall-through LLM** — kể cả lỗi DB hoặc không nhận product → trả message trực tiếp (stub tiếng Việt).

**Tool `get_maintenance` (LLM):** cũng gọi `maintenanceForChat` + `EnrichMaintenanceOutput`; `provider` routing tuỳ chọn qua `FilterMaintenanceByProvider`.

**`NormalizeToolArgs`:** `provider` nhầm tên dịch vụ → `product`; `sku` qua `NormalizeSKU`. Provider routing chỉ `ESALE`/`IMEDIA`/`SHOPPAY`.

**Nhận biết deploy đúng:** câu trả lời dạng `**GARENA** — có **N** mệnh giá đang bảo trì:` — **không** có câu LLM kiểu *"(provider ESALE) không có bảo trì"*; **không** lặp câu hỏi user thành tên product (`**NGOÀI_RA_CÒN_...**`).

**Ví dụ bảo trì:**

```text
USER: "thẻ Garena có đang bảo trì không"
→ GARENA → liệt kê 10000, 20000, 50000, 100000 (khớp nhãn BT dưới SKU trên Dashboard).

USER: "Thẻ Mobifone 10.000 có bảo trì không?"
→ MOBIFONE / sku=10000.

USER (follow-up): "Ý tôi nói là thẻ Mobifone 10.000"
→ session follow-up → MOBIFONE / 10000.

USER: "Ngoài ra còn loại dịch vụ nào đang bảo trì không"
→ FormatAllMaintenanceReply — liệt kê mọi dịch vụ đang BT, không lặp câu hỏi.
```

##### 7.6.5.3 Thống kê câu hỏi thường gặp (`chat_intent_stats`)

Mỗi lượt chat có intent đã biết (`maintenance`, `metrics`, …) → `BumpChatIntentStat` (async) ghi DB:

| Cột | Mô tả |
|-----|--------|
| `intent_key` | `maintenance` \| `metrics` (mở rộng sau: `routing`, `incident`, …) |
| `pattern_hash` | SHA-256 rút gọn của `intent_key` + message đã normalize |
| `sample_message` | Câu mẫu gần nhất (≤512 ký tự) |
| `hit_count` | Số lần gặp (UPSERT +1) |
| `route_key` | Route thực tế lần cuối (`direct_metrics`, `direct_set_maintenance`, `llm`, …) — ✅ P2 |
| `success_count` / `fail_count` | Tăng theo `action_result` của lượt chat — ✅ P2 |
| `last_seen_at` | Lần cuối user hỏi |

**Mục đích:** theo dõi pattern ops hay hỏi; ưu tiên direct reply (§7.6.5.1–2) thay vì LLM — nhanh và khớp Dashboard. UI admin xem top FAQ — *tuỳ chọn triển khai sau*.

**DDL:** bảng `chat_intent_stats` — §13.9. *Planned:* cột `status` (`candidate` \| `approved` \| `deprecated`) gắn với vòng học lệnh §7.6.5.5 P3+.

##### 7.6.5.4 Lệnh trực tiếp — chat/voice Tầng B ✅ triển khai

**Hai tầng xử lý (§9.2):**

| Tầng | Xử lý | Ví dụ |
|------|--------|--------|
| **A — Voice UI** (React) | Mic + khung chat; **không** gọi API vận hành | `alo`, `bật mic`, `đóng chat`, `tắt mic` |
| **B — OpsOne Agent** (`POST /chat`) | Tra cứu, duyệt, bảo trì, đổi chế độ | *"bảo trì giúp tôi thẻ Garena"* |

**Direct handler (`tryChatCommandReply`)** — chạy **sau** metric, **trước** tra cứu BT status; tránh nhầm *"bảo trì giúp tôi"* → *"không có BT active"*:

| Nhóm lệnh | Trigger (ví dụ) | Hành động |
|-----------|-----------------|-----------|
| Duyệt / từ chối | `duyệt`, `ok`, `từ chối`, `không` | Dùng **pending focus** (`session_id`) hoặc scope trong câu |
| Xem pending | `xem pending`, `việc chờ duyệt` | `list_pending_actions` + set focus mục đầu |
| **Bật bảo trì** | `bật bảo trì`, **`bảo trì giúp tôi`**, `bảo trì toàn bộ mệnh giá` | `set_maintenance` — product / SKU / provider |
| Mở lại dịch vụ | `mở lại dịch vụ`, `kết thúc bảo trì` | `reopen_service` |
| Chế độ auto | `đặt … chế độ tự động`, `chỉ đề xuất`, `tự động theo khung giờ` | `set_scope_auto` |

**Phân biệt tra cứu vs lệnh BT** (`internal/chatresolve/commands.go`):

- **Tra cứu:** `IsMaintenanceStatusQuery` — *"có đang bảo trì không"*, *"co dang"*, *"?"* → `tryChatMaintenanceReply`
- **Lệnh bật BT:** `IsSetMaintenanceCommand` — *"giúp tôi"*, *"toàn bộ"*, *"bật bảo trì"* → `tryChatCommandReply` → **không** fall-through tra cứu

**Pending focus (in-memory, theo `session_id`):** sau `list_pending_actions` hoặc liệt kê pending — lưu mục đầu (`routing_plan` \| `maintenance_suggestion`); lệnh ngắn `ok`/`không` áp dụng focus đó. *Planned §7.6.5.5:* persist vào DB.

**Template phản hồi sau hành động** (`formatChatActionReply`) — tối đa ~3 dòng:

1. **Kết quả** — Đã / Không thực hiện được (+ tên dịch vụ · SKU)
2. **Chi tiết** — thời gian, provider, trạng thái mới
3. **Gợi ý tiếp** *(tuỳ chọn)* — *"Nói xem pending để liệt kê"*

**Tool Admin (LLM + direct handler):**

| Tool | Mô tả |
|------|--------|
| `approve_routing_plan` / `reject_routing_plan` | theo `plan_id` |
| `approve_scope_routing` / `reject_scope_routing` | theo `product` (+ `sku?`) |
| `approve_scope_maintenance` / `reject_scope_maintenance` | đề xuất BT scope |
| `set_maintenance` | Bật BT thủ công — `product` bắt buộc; `sku`/`provider` tuỳ chọn; bỏ `sku` = toàn dịch vụ |
| `reopen_service` | Hủy BT active + baseline routing |
| `set_scope_auto` | `auto` \| `time_window` \| `recommend_only` |

**Quy tắc:**

1. Gọi `list_pending_actions` trước khi duyệt nếu chưa biết `plan_id`/scope.
2. User không phải Admin → **không** gọi tool duyệt; hướng dẫn Dashboard.
3. Không expose trực tiếp `UpdateRouting` / `SetMaintenance` registry — duyệt/BT qua tool chat hoặc API scope giống UI Admin.
4. Lệnh phá hoại (BT, duyệt, đổi auto) **cần Admin**; non-admin → từ chối + hướng dẫn Dashboard.
5. Câu hỏi ngoài vận hành → từ chối lịch sự.

##### 7.6.5.5 Học kinh nghiệm giao tiếp & lệnh — P1–P2 ✅ · P3–P5 📋

**Mục tiêu:** Agent **tích lũy kinh nghiệm theo thời gian** — nghe/voice/chat chuẩn hơn, ít nhầm tra cứu vs lệnh — **không** tự sửa rule production không kiểm duyệt.

**Nguyên tắc:**

| Nguyên tắc | Ý nghĩa |
|------------|---------|
| Log trước, học sau | Mọi lượt `/chat` ghi `chat_interaction_log` ✅ |
| Học có kiểm duyệt | Pattern mined → `candidate` → Admin **promote** → `approved` 📋 |
| Ba tầng nhớ | Rule code → pattern DB đã duyệt → LLM + few-shot |
| Voice riêng | STT sai → `chat_voice_corrections`, không trộn rule BT 📋 |

**Triển khai P1–P2 (code):**

| Thành phần | File | Mô tả |
|------------|------|--------|
| Persist session | `internal/store/chat_log.go` — `UpsertChatSession` | `session_uuid` + `user_id` (`X-OpsOne-Actor` / dev bypass) |
| Persist messages | `InsertChatMessage` | User + assistant mỗi lượt; user có `input_source`, `stt_raw` |
| Interaction log | `InsertChatInteractionLog` | Route, intent, slots, tools, result, latency |
| Wire handler | `internal/api/chat_turn.go` — `persistChatTurn` | Gọi sau mỗi `POST /chat` (sync, không chặn reply) |
| Route metadata | `internal/api/chat_agent.go` — `ChatTurnOutcome` | Điền trong `chatAgentReply` |
| Frontend voice | `web/src/components/ChatWidget.tsx` | Gửi `input_source: "voice"`, `stt_raw` khi mic gửi câu |

**Giá trị `route` (`chat_interaction_log.route`):**

| Route | Nguồn |
|-------|--------|
| `direct_metrics` | `tryChatMetricsReply` |
| `direct_maintenance` | `tryChatMaintenanceReply` |
| `direct_list_pending` | `tryChatCommandReply` — xem pending |
| `direct_approve` / `direct_reject` | Lệnh ngắn ok/không |
| `direct_approve_reject_scoped` | Duyệt/từ chối có scope trong câu |
| `direct_set_maintenance` | Bật bảo trì chủ động |
| `direct_reopen_service` | Mở lại dịch vụ |
| `direct_set_scope_auto` | Đổi chế độ auto/khung giờ/chỉ đề xuất |
| `llm` | Tool loop OpenAI-compatible |
| `stub` | Không có `LLM_API_KEY` — keyword fallback |

**`slots_json` (direct routes):** product, sku, auto_action, duration_min — extract từ `chatresolve` (`ExtractProductFromText`, …).

**`action_result`:** `success` (reply OK) · `error` (LLM lỗi) · `no_op` (vượt max tool rounds) · `wrong_route` *(reserved — P5 retry signal)*.

**Lưu ý:** Session LLM vẫn **in-memory** (tối đa 40 message/`session_id`); DB `chat_messages` phục vụ audit + học — **chưa** hydrate lại context LLM từ DB.

**Luồng runtime (mục tiêu đầy đủ):**

```text
User (text/voice)
  → Router: approved pattern → rule code → LLM+few-shot → direct reply
  → Thực thi tool/API
  → Ghi chat_interaction_log (+ chat_messages)     ← ✅ P1–P2
  → Tín hiệu feedback (explicit + ngầm: retry, "sai rồi", tool success)   ← 📋 P5
  → Job đào pattern (daily) → chat_command_patterns (candidate)           ← 📋 P3
  → Admin promote → pattern/few-shot dùng lại lần sau                     ← 📋 P3–P4
```

**Router ưu tiên (planned — P3+):**

1. `chat_command_patterns` **approved** (confidence cao)
2. Rule code (`chatresolve/commands.go`, `intent.go`) ← **hiện tại**
3. Pattern **candidate** + slot — hỏi xác nhận nếu confidence < ngưỡng (lệnh Admin)
4. LLM + top-K `chat_few_shot_examples`
5. Direct metric/BT status

**Tín hiệu học:**

| Tín hiệu | Suy ra |
|----------|--------|
| User gửi lại cùng ý trong ~60s | Lần trước **wrong_route** |
| *"sai rồi"*, *"không phải"* sau reply | `fail` interaction trước |
| *"ok"*, *"đúng rồi"* sau lệnh thực thi | `success` |
| Tool + DB thay đổi (BT created, routing applied) | `success` cứng |
| Voice: user sửa transcript trước Gửi | Cặp `(stt_raw → corrected)` |

**Lộ trình triển khai:**

| Phase | Việc | Trạng thái |
|-------|------|------------|
| **P0** | Rule code + `tryChatCommandReply` + `chat_intent_stats` | ✅ |
| **P1** | Wire `chat_interaction_log` + persist `chat_sessions`/`chat_messages` | ✅ |
| **P2** | Log `route`, `slots_json`, `action_result`, `latency_ms`, `tools_called` mỗi lượt | ✅ |
| **P3** | Job đào pattern + Admin UI promote | 📋 |
| **P4** | `chat_few_shot_examples` inject prompt | 📋 |
| **P5** | `chat_voice_corrections` + retry signal | 📋 |

**DDL:** §13.9 — `chat_interaction_log`, `chat_command_patterns`, `chat_feedback`, `chat_few_shot_examples`, `chat_voice_corrections`, `chat_user_prefs`.

**Kiểm tra sau deploy:**

```sql
SELECT route, intent_key, action_result, latency_ms, created_at
FROM chat_interaction_log
ORDER BY id DESC LIMIT 10;
```

**Ví dụ:**

```text
USER: "Tình hình topup mobi hiện tại ok không?"
→ Map TOPUP_MOBI → get_routing + get_metrics (cả 3 provider) → tóm tắt tiếng Việt.

USER: "Tại sao ESALE TOPUP_VINA fail tăng?"
→ get_metrics(TOPUP_VINA, ESALE) + get_top_errors → tóm tắt 2–3 câu tiếng Việt.

USER (Admin): "Duyệt routing plan đang chờ cho TOPUP_VINA"
→ list_pending_actions → approve_scope_routing hoặc approve_routing_plan → báo kết quả.

USER (Admin): "bảo trì giúp tôi thẻ Garena toàn bộ mệnh giá"
→ tryChatCommandReply → set_maintenance(GARENA, sku=*) → template 3 dòng (§7.6.5.4).

USER: "Thẻ Garena có đang bảo trì không?"
→ tryChatMaintenanceReply — **không** bật BT.

USER: "Khôi phục routing TOPUP_VINA" (chưa yêu cầu duyệt)
→ Hướng dẫn **Mở lại provider** / **Mở lại dịch vụ** trên Dashboard.
```

#### 7.6.6 Guardrails LLM

| Loại | Quy tắc | Implement |
|------|---------|-----------|
| **Token budget** | `max_input_tokens=8000`, `max_output_tokens=400` cho summary; `1500` cho chat | Cấu hình `internal/reasoning/llm_client.go` |
| **Temperature** | `0.2` cho summary/reason (đều đặn); `0.5` cho chat (tự nhiên hơn) | Per-prompt |
| **JSON mode** | Bật `response_format: json_object` cho task §7.6.2–7.6.4 | OpenAI-compatible API |
| **Timeout** | 10s mỗi LLM call; quá → fallback dùng `breach_reasons[]` ghép text | Context với deadline |
| **Fallback offline** | LLM down → dùng template tiếng Việt tĩnh trong `internal/i18n/template_vi.go` | Phase 4 mandatory |
| **No leak system prompt** | Không bao giờ trả system prompt cho user dù được hỏi | Filter response |
| **Audit** | Log mỗi LLM call: prompt_name, latency_ms, tokens, output snippet (truncate 200 chars) | Bảng `llm_call_log` (tuỳ chọn) |

#### 7.6.7 Cấu hình LLM

**Chat agent (triển khai):** file `ai_gen_src/.env`. `config.Load()` **tự đọc** `.env` trong thư mục làm việc (không ghi đè biến env đã set sẵn). Khuyến nghị dev: `scripts/run-api.ps1` (nạp `.env` + kill port `:8080` + log `LLM chat: enabled model=…`).

```bash
# ai_gen_src/.env — GreenNode AI Platform (MaaS, OpenAI-compatible)
LLM_API_URL=https://maas-llm-aiplatform-hcm.api.vngcloud.vn/v1   # optional; default nếu trống
LLM_API_KEY=<AIP-key>          # alias AIP_API_KEY; trống → chat stub keyword
LLM_MODEL=minimax/minimax-m2.5 # id từ GET .../v1/models — chuẩn hóa lowercase khi load
LLM_TIMEOUT_SECONDS=30
```

**Lấy model id:** `curl -H "Authorization: Bearer $LLM_API_KEY" https://maas-llm-aiplatform-hcm.api.vngcloud.vn/v1/models` → dùng đúng `id` (phân biệt hoa/thường). Bật model trên [AI Platform Console](https://aiplatform.console.vngcloud.vn/models).

**Reasoning scheduler (Phase 4):** vẫn fallback template tiếng Việt khi chưa gọi LLM summary; biến `LLM_API_URL`/`LLM_API_KEY` dùng chung config.

**Tuỳ chọn provider khác (OpenAI, Ollama, Azure):**

```bash
LLM_API_URL=https://api.openai.com/v1
LLM_API_KEY=sk-...
LLM_MODEL=openai/gpt-4o-mini
LLM_TIMEOUT_SECONDS=30
```

#### 7.6.8 Checklist Prompt Engineering

- [ ] 5 prompt file riêng (system + 4 task) trong `internal/reasoning/prompts/`.
- [ ] Mỗi prompt **versioned** (`task_incident_summary_v1.txt`); test snapshot.
- [ ] Unit test: feed fixture context → assert LLM-free fallback ra đúng template Việt (không gọi API trong CI).
- [ ] Integration test (cờ env): real LLM call → assert JSON schema khớp, ngôn ngữ Việt, không leak system.
- [ ] `LLM_ENABLED=false` → toàn hệ thống vẫn chạy được (chỉ mất summary tự nhiên — fallback template).

---

## 8. Bước 5 — Output

Đây là **kết quả cuối cùng**.

### 8.1 Bốn loại output


| Loại               | Khi nào sinh                              | Mức độ ưu tiên demo                     |
| ------------------ | ----------------------------------------- | --------------------------------------- |
| **Incident**       | Metric vượt ngưỡng / xu hướng xấu         | Cao                                     |
| **Recommendation** | Cần hành động nhưng chưa auto             | Cao                                     |
| **Maintenance**    | Chỉ 1 provider **active** hoặc không có backup healthy | Trung bình (**ZING** — §10.2, §10.2b) |
| **Routing Plan**   | Multi provider, có backup tốt hơn         | **WOW** — trọng tâm                     |
| **Health Status**  | Mỗi chu kỳ phân tích — tổng quan hệ thống | **Cao** — nhìn một icon biết tình trạng |


### 8.2 Trạng thái tổng quan — 3 icon màu (`health_status`)

Sau mỗi lần Agent phân tích, output **bắt buộc** có trạng thái đại diện bằng **một icon màu** — hiển thị trên **header UI**, **feed**, và **từng product** (tuỳ chọn).


| Icon | `health_status` | Nhãn UI                   | Ý nghĩa                                                               |
| ---- | --------------- | ------------------------- | --------------------------------------------------------------------- |
| 🟢   | `green`         | **Hệ thống OK**           | Metric ổn định; không incident; không cần hành động                   |
| 🟡   | `yellow`        | **Đang theo dõi / xử lý** | Nghi vấn; đang **chuyển routing**; đang **monitor thêm**; chờ approve |
| 🔴   | `red`           | **Đang có vấn đề**        | Incident nghiêm trọng; metric vượt ngưỡng; maintenance                |


**Quy tắc gán trạng thái (ưu tiên từ cao xuống thấp):**

```text
🔴 red    — Vượt ngưỡng product §1.2 (tỷ lệ lỗi HOẶC số GD lỗi HOẶC pending %)
         — HOẶC Sự cố mức Cao
         — HOẶC tỷ lệ thành công < success_rate_min_pct (product)
         — HOẶC Đề xuất / đang thực hiện bảo trì

🟡 yellow — Sự cố mức Trung bình / Thấp
         — HOẶC Đề xuất: theo dõi thêm (VD 15 phút)
         — HOẶC Kế hoạch routing chờ duyệt
         — HOẶC Vừa điều phối routing / đang chuyển traffic
         — HOẶC 1–2 rule cảnh báo chưa vượt ngưỡng đỏ

🟢 green  — Không thỏa điều kiện vàng/đỏ
         — Quyết định: theo dõi; metric trong baseline
```

**Trạng thái tổng hệ thống vs từng scope:**


| Cấp             | Nguồn | Mô tả                                                               |
| --------------- | ----- | ------------------------------------------------------------------- |
| **Global (UI header + logo badge)** | `useOverallHealth` | `max` mọi dòng `GET /dashboard/overview` (`effectiveRowHealth`: live health + plan chờ 🔴) **và** `GET /health-status` chu kỳ — **1 SKU/tab đỏ → header 🔴** |
| **Per product (API `/health-status`)** | Agent chu kỳ | `products[]` — aggregate mọi scope trong product; SSE `health_status` |
| **Per scope — Dashboard cột TT** | **`liveScopeHealth` + `effectiveRowHealth` §2.3.2** | Metric **gom scope** (5 ngưỡng OR); vi phạm bất kỳ → 🔴; scope OK → 🟢 dù Agent state còn `WARNING` |
| **Per scope — Agent / audit** | `agent_state_history` | `NORMAL`/`WARNING`/`INCIDENT`/`RECOVERING` — consecutive breach, recovery, chat *"state nào?"* |

> **Dashboard ≠ Agent state:** Cột TT và tab dịch vụ dùng **live metric** (§2.3.2). `agent_state_history` vẫn ghi mỗi chu kỳ cho pipeline hành động — không dùng trực tiếp làm màu TT sau fix 2026-06.


**Payload output mẫu:**

```json
{
  "analysis_cycle": "2026-06-04T10:10:00+07:00",
  "health_status": "yellow",
  "health_label": "Đang theo dõi / xử lý",
  "health_summary": "ZING SKU 20k suy giảm — Kế hoạch routing chờ duyệt",
  "products": [
    { "product": "ZING", "health_status": "yellow", "health_summary": "SKU 20000 ESALE — tỷ lệ lỗi tăng" },
    { "product": "TOPUP_VINA", "health_status": "green", "health_summary": "Ổn định" },
    { "product": "DATA_VINA", "health_status": "green", "health_summary": "Ổn định" }
  ]
}
```

**Checklist triển khai:**

- Rules engine tính `health_status` sau Reasoning — không để LLM tự đoán màu.
- Mỗi output JSON có field `health_status` + `health_summary` (1 câu). Chu kỳ đỏ: `health_summary` dạng `Sự cố: {product_label, …}` — nhãn từ `products.label` / `catalog.ProductDisplayLabel`, không liệt kê mã `DATA_MOBI` thô.
- UI: icon 🟢🟡🔴 + text nhãn; hỗ trợ color-blind (không chỉ dựa màu — kèm chữ).
- Mobile: icon trên header + từng incident card.
- Lưu `health_status` vào history để vẽ xu hướng (VD: 🟡 chu kỳ 1 → 🔴 chu kỳ 2 khi đủ `consecutive_cycles_required`).

### 8.3 Incident

Agent **tự sinh** Incident.

**Checklist nội dung:**

- `incident_id` (VD: `20260604-001`)
- `severity`: Thấp | Trung bình | Cao (enum DB: `low` | `medium` | `high`)
- `product`, `provider`; tuỳ scope thêm `sku` / `sku_label` (VD: 20.000đ)
- Biến động metric (before → after)

**Ví dụ:**

```text
Sự cố #20260604-001

Mức độ: Cao

Sản phẩm:  ZING (Thẻ Zing)
Nhà cung cấp: ESALE
SKU:      20.000đ (20000)

Tỷ lệ thành công: 85% → 72%  (SKU 20k trên ESALE; SKU 10k vẫn ~98%)
Tỷ lệ lỗi:        7% → 13%
```

**Trạng thái & xử lý (§8.3.1):**

Sự cố sinh ra với `status=open`. Khi ops/agent xử lý qua routing plan, hệ thống **cập nhật trạng thái** và ghi audit:

| Hành động | `status` | `resolution_action` | `handled_by` | `handled_at` |
| --------- | -------- | ------------------- | ------------ | ------------ |
| Admin **Duyệt** routing plan (`POST .../approve`) | `resolved` | `admin_approve` | Actor từ JWT / `X-OpsOne-Actor` (dev) | `NOW()` |
| Admin **Từ chối** routing plan (`POST .../reject`) | `acknowledged` | `admin_reject` | Actor admin | `NOW()` |
| Agent **Auto** apply routing (`trigger_type=auto`) | `resolved` | `auto` | `opsone-agent` hoặc `ExecutedBy` | `NOW()` |

Liên kết sự cố: ưu tiên cùng `cycle_id` + `product_code` + `sku_code` với routing plan; fallback sự cố `open` mới nhất cùng scope. Cập nhật khi `status=open` **hoặc** `handled_at IS NULL` (cho phép bổ sung audit sự cố đã đóng trước khi có cột audit).

**Ràng buộc 1 sự cố mở / SKU (triển khai `reasoning.go` + `FindOpenIncidentForScope`):**

- Key scope: `(product_code, sku_code)` — topup `sku_code=''` vẫn là một scope.
- Trước `InsertIncident`: nếu đã có incident **`status=open`** cùng scope → **không tạo mới**; tái dùng `incident_id` cũ cho routing plan / recommendation / đóng sự cố khi duyệt-từ chối.
- Chỉ tạo incident mới sau khi sự cố cũ đã **Duyệt** (`resolved`) hoặc **Từ chối** (`acknowledged`) qua approve/reject routing plan §8.3.1.

**Sau duyệt/từ chối:** UI invalidate cache `incidents` + `dashboard-overview` (Dashboard banner).

**Đổi DDL:** sửa `db/schema.sql` rồi chạy `Invoke-OpsOneReset` — script DROP toàn bộ bảng và CREATE lại + `seed.sql` (không dùng file patch riêng).

API `GET /incidents`, `GET /incidents/{id}` trả thêm `handled_by`, `handled_at`, `resolution_action`.

### 8.4 Recommendation

**Đề xuất** hành động.

**Checklist:**

- Hành động cụ thể (shift %, monitor thêm, escalate)
- Không mơ hồ (“cần theo dõi” phải kèm thời gian)

**Ví dụ:**

```text
Đề xuất

Chuyển 40% traffic (SKU 20.000đ)
từ ESALE sang IMEDIA
```

**Hoặc — chỉ đổi một mệnh giá, giữ nguyên SKU khác:**

```text
Đề xuất

SKU 20.000đ: hard cut ESALE 80% → **0%**, IMEDIA 20% → **100%**
SKU 10.000đ: giữ nguyên (theo dõi thêm)
```

**Hoặc:**

```text
Đề xuất

Theo dõi thêm 15 phút.
```

### 8.5 Maintenance — bảo trì có thời hạn (starts_at / ends_at)

**Khi nào dùng:** Product có **`active_provider_count == 1`** — VD **ZING chỉ ESALE active** (§10.2) hoặc **catalog 2 provider nhưng 1 inactive** (§10.2b) — **không** thể chuyển traffic sang provider khác → OpsOne đề xuất hoặc thực thi **bảo trì có thời gian bắt đầu và kết thúc**.

Cũng dùng khi **`active_count > 1` nhưng `healthy_backup_count == 0`** — tất cả provider active đều lỗi/pending, không có nơi routing an toàn (§7.4).

**Khái niệm — cửa sổ bảo trì (`maintenance_windows`):**


| Trường      | Mô tả                                                                                    |
| ----------- | ---------------------------------------------------------------------------------------- |
| `starts_at` | Thời điểm **bắt đầu** bảo trì (product/provider tạm ngưng / hiển thị trạng thái bảo trì) |
| `ends_at`   | Thời điểm **kết thúc** bảo trì — hệ thống tự chuyển `completed`                          |
| `status`    | `pending_approve` → `scheduled` → `active` → `completed` (hoặc `cancelled`)              |


**Trạng thái vòng đời:**

```text
pending_approve  ──approve──► scheduled (starts_at > now)
                    │              │
                    │              └── starts_at đến ──► active
                    │                                    │
                    └── approve (starts_at ≤ now) ──────►│
                                                         ends_at đến ──► completed
cancel ──► cancelled (bất kỳ lúc nào trước completed)
```

**Điều kiện kích hoạt (OpsOne):**

```text
GetProviders(`ZING`) → active_count === 1   (chỉ ESALE active; IMEDIA inactive hoặc không có)
Tỷ lệ thành công < 50% VÀ tỷ lệ lỗi > 30% — 2 chu kỳ liên tiếp (≈ 10 phút)
→ suggested_action = "maintenance"
→ Đề xuất bảo trì (kèm starts_at / ends_at)
```

**Ví dụ output — ZING (chỉ ESALE):**

```text
Đề xuất bảo trì

Sản phẩm:   ZING (Thẻ Zing)
Nhà cung cấp:  ESALE

Thời gian bảo trì:
  Bắt đầu:  2026-06-04 10:15:00 +07:00
  Kết thúc: 2026-06-04 11:15:00 +07:00   (mặc định 60 phút — cấu hình UI)

Lý do: Tỷ lệ thành công 42%, tỷ lệ lỗi 35% — 2 chu kỳ liên tiếp;
       ZING chỉ có ESALE, không có nhà cung cấp dự phòng.
```

**JSON output (lưu DB / API):**

```json
{
  "maintenance_id": "20260604-M001",
  "product_code": "ZING",
  "provider_code": "ESALE",
  "sku_code": "",
  "starts_at": "2026-06-04T10:15:00+07:00",
  "ends_at": "2026-06-04T11:15:00+07:00",
  "status": "pending_approve",
  "trigger_type": "opsone_recommend",
  "reason": "ZING chỉ ESALE — metric xấu 2 chu kỳ, không provider dự phòng",
  "cycle_id": 12
}
```

**Tool §6.8 GetMaintenance — đọc cửa sổ hiện tại:**

```json
// Input
{ "product": "ZING", "provider": "ESALE" }

// Output
{
  "active": [
    {
      "maintenance_id": "20260604-M001",
      "starts_at": "2026-06-04T10:15:00+07:00",
      "ends_at": "2026-06-04T11:15:00+07:00",
      "status": "active",
      "remaining_minutes": 42
    }
  ],
  "scheduled": []
}
```

**Tool §6.9 SetMaintenance — lên lịch / kích hoạt:**

```json
{
  "product": "ZING",
  "provider": "ESALE",
  "starts_at": "2026-06-04T10:15:00+07:00",
  "ends_at": "2026-06-04T11:15:00+07:00",
  "trigger_type": "opsone_auto",
  "reason": "Bảo trì tự động sau 2 chu kỳ metric xấu liên tiếp"
}
```

**Quy tắc thời gian:**

- `ends_at` **bắt buộc** > `starts_at`.
- Chỉ yêu cầu `ends_at > starts_at`; admin chọn `starts_at` / `ends_at` tùy ý (không giới hạn thời lượng).
- Mặc định khi OpsOne auto: `starts_at = now()`, `ends_at = now() + maintenance_default_duration_min` (§9.5 / `agent_settings`).
- **Không chồng lấn:** cùng `(product_code, provider_code, sku_code)` không có 2 cửa sổ `active`/`scheduled` giao nhau — trả lỗi validate.

**Tác động runtime:**

- Trong `[starts_at, ends_at)`: product card UI hiển thị 🔧 *Đang bảo trì* + countdown; `health_status` tối thiểu 🟡.
- **`worker-mock` (§4.5):** không sinh `mock_metrics` / `mock_error_stats` cho `(product_code, sku_code, provider_code)` đang trong cửa sổ — tránh noise và breach giả sau khi admin đã BT.
- Agent chu kỳ sau: **bỏ qua** cảnh báo metric giả tạo do bảo trì (đã biết đang maintenance).
- Hết `ends_at`: worker/API chuyển `completed`; metric được đánh giá lại bình thường.

**Chế độ triển khai:** Bảo trì qua **đề xuất + Duyệt** trên Dashboard:
- Recommendation có `id` (Agent) → `POST /recommendations/{id}/approve` — `SetMaintenance`, `trigger_type=admin_manual`.
- Đề xuất **synthetic** (overview, không `id`) → `POST /scopes/{product}/{sku}/maintenance/approve`.
- `maintenance_auto_enabled` trên `agent_settings` (tùy chọn) cho auto `SetMaintenance` khi `active_provider_count==1` (`trigger_type=opsone_auto`).


**Checklist triển khai:**

- Bảng `maintenance_windows` §13.8.
- `internal/maintenance` — Create, Approve, Cancel, TickStatus (scheduled→active→completed).
- GetMaintenance / SetMaintenance trong `internal/tools/`.
- Nhánh logic khi `GetProviders.active_count === 1` hoặc `healthy_backup_count === 0` (§7.4).
- Đếm 2 chu kỳ liên tiếp thỏa điều kiện trước khi đề xuất/auto (`consecutive_cycles_required`, mặc định 2).
- UI: card bảo trì + countdown; form admin chỉnh starts_at / ends_at khi approve.
- Chat: *"ZING bảo trì đến mấy giờ?"* → đọc GetMaintenance.

### 8.6 Routing Plan — phần "WOW"

Agent **không chỉ nói lỗi** — đưa ra **phương án tối ưu**.

> **Nguyên tắc cốt lõi (đọc trước khi triển khai):**
> Tỉ lệ routing **ban đầu do Business** ấn định và seed vào `routing_config` (§13.3, §13.10).
> **OpsOne KHÔNG tự sinh tỉ lệ baseline** — Agent **chỉ điều chỉnh tạm thời** khi phát hiện sự cố / vượt ngưỡng,
> và **có thể tự khôi phục** (Mở lại provider / Mở lại dịch vụ) về cấu hình Business khi sự cố hồi.

#### 8.6.1 Hai pha routing — Khởi tạo (Business) vs Điều chỉnh (Agent)

| Pha | Khi nào | Ai quyết định tỉ lệ | Lưu ở | Trigger |
|-----|---------|--------------------|-------|---------|
| **A. Baseline (khởi tạo)** | Seed hệ thống / Admin PUT với `set_as_baseline=true` | **Business / Admin** | `baseline_pct` + `traffic_pct` trong `routing_config` | Thủ công — quyết định kinh doanh |
| **A′. Admin chỉnh tạm** | Admin PUT với `set_as_baseline=false` | **Admin** (can thiệp tạm) | Chỉ `traffic_pct`; `baseline_pct` giữ nguyên; `pending_restore=1` | Thủ công — override tạm khi sự cố |
| **B. Điều chỉnh (Agent)** | Agent chu kỳ phát hiện `breached` (§7.4) + rule trigger (§7.3) | **OpsOne** sinh Routing Plan; admin approve hoặc auto (§9.5.2) | `traffic_pct` + `agent_change_log` | Có sự cố / vượt ngưỡng |
| **C. Khôi phục baseline** | Metric ổn định ≥ N chu kỳ (§8.6.3 Bước 4) | **OpsOne** `restore_baseline` | `traffic_pct ← baseline_pct`; `pending_restore=0` | Hệ thống ổn sau Pha B hoặc A′ |

**Pha A — KHÔNG thuộc trách nhiệm Agent:**

- Baseline phản ánh **chiến lược kinh doanh**: hợp đồng SLA với provider, ưu tiên giá, chia tải an toàn, cam kết doanh số.
- VD baseline TOPUP_VINA: `ESALE 70 | IMEDIA 20 | SHOPPAY 10` — là **quyết định Business**, không tính từ `success_rate`.
- Admin nhập trên UI Settings hoặc seed SQL (§13.10). Mọi thay đổi baseline (`set_as_baseline=true`) ghi `config_audit_log` (§13.4).

**Pha A′ — Admin chỉnh tạm (`set_as_baseline=false`):**

- Admin can thiệp **traffic đang chạy** khi cần xử lý nhanh — **không** đổi baseline Business.
- Hệ thống đánh dấu scope `(product, sku)` với `pending_restore=1` → khi metric **ổn định** (§8.6.3 Bước 4), Agent tự `traffic_pct = baseline_pct`.
- VD: baseline TOPUP_VINA `70/20/10`; admin tạm set `50/30/20` trong lúc ESALE lỗi → khi hệ thống ổn → tự về `70/20/10`.

**Pha B — Agent CHỈ kích hoạt khi:**

```text
EvaluateThresholds(product) == breached
  AND consecutive_breach_cycles >= required (§7.4)
  AND scope.auto_action allows auto (§9.5.2) — nếu chỉ đề xuất thì vẫn sinh Plan, không apply
```

Không vượt ngưỡng → Agent **giữ nguyên** tỉ lệ baseline; output `health=green` + Recommendation `monitor` (nếu cần). **Không** sinh Routing Plan trong chu kỳ ổn định.

#### 8.6.2 Cấu trúc Routing Plan output (khi vào Pha B)

Phải có **4 khối:** Hiện tại | Hiệu năng | Đề xuất | Kết quả kỳ vọng.

- **Topup tiền:** một bảng routing theo **provider** (cả product).
- **Thẻ / Topup data:** một bảng **per SKU** cần hành động (không có bảng aggregate product).

**Ví dụ — topup tiền (routing theo provider):**

```text
Sản phẩm: TOPUP_VINA (topup)

Hiện tại
  ESALE     70%
  IMEDIA    20%
  SHOPPAY   10%

Hiệu năng
  ESALE     82%
  IMEDIA    98%
  SHOPPAY   97%

Đề xuất
  ESALE     10%
  IMEDIA    60%
  SHOPPAY   30%

Kết quả kỳ vọng
  Tỷ lệ thành công dự kiến
  82% → 97%
```

**Bảng tóm tắt (topup tiền):**


| Provider | Hiện tại | Hiệu năng | Đề xuất |
| -------- | -------- | --------- | ------- |
| ESALE    | 70%      | 82%       | 10%     |
| IMEDIA   | 20%      | 98%       | 60%     |
| SHOPPAY  | 10%      | 97%       | 30%     |


**Ví dụ — thẻ, SKU 20.000đ (WOW khi lỗi cục bộ mệnh giá):**

```text
Phạm vi: SKU 20000 (20.000đ) — Sản phẩm ZING

Hiện tại
  ESALE     80%
  IMEDIA    20%

Hiệu năng
  ESALE     72%
  IMEDIA    98%

Đề xuất
  ESALE     10%
  IMEDIA    90%

Kết quả kỳ vọng (SKU 20000)
  Tỷ lệ thành công: 72% → 96%
```


| Provider | Hiện tại | Hiệu năng | Đề xuất |
| -------- | -------- | --------- | ------- |
| ESALE    | 80%      | 72%       | 10%     |
| IMEDIA   | 20%      | 98%       | 90%     |


#### 8.6.3 Thuật toán điều chỉnh khi có sự cố (Pha B)

**Input:** snapshot routing hiện tại + metric provider/SKU (window 15m, §1.2) + history N chu kỳ + baseline.

**Bước 1 — Xác định provider vi phạm trong scope (per product hoặc per SKU):**

Provider **breached** khi vi phạm **bất kỳ** ngưỡng metric §1.2 / §7.4 (`success_rate`, `fail_rate`, `pending_rate`, `fail_txn_count`, `pending_txn_count`…) **và** đủ `consecutive_cycles_required` (§9.0).

**Bước 2 — Hard cut (duy nhất):** Chỉ shift khi có ≥ 1 provider healthy (không breach) còn `routing_pct > 0`. Mọi provider breached → **0%**:

```text
for p in breached where current_pct[p] > 0:
    drop[p] = current_pct[p]
    new_pct[p] = 0

donor_pool = Σ drop[p]
weight_sum = Σ success_rate[g] for g in healthy where new_pct[g] > 0

for g in healthy:
    new_pct[g] = current_pct[g] + donor_pool * (success_rate[g] / weight_sum)

# Round số nguyên, điều chỉnh ±1% để Σ new_pct = 100
# Nếu chỉ còn 1 provider healthy → new_pct[healthy] = 100
```

**Guardrails (BẮT BUỘC — fail-safe trước khi gọi UpdateRouting):**

| Rule | Giá trị | Lý do |
|------|---------|-------|
| Provider breached | **0%** | Cắt hẳn traffic — không giảm dần, không giữ mẫu monitor |
| Mỗi provider | **0–100%** | Admin/agent tự do chia; không cap 10%/90% |
| `Σ new_pct == 100` | bắt buộc | Validate trước apply |
| Tất cả provider active đều breached **hoặc** chỉ 1 provider active | **bỏ shift** → `SetMaintenance` (§8.5) | Không có nơi chuyển traffic an toàn |
| `expected_success ≤ current_weighted_success + min_improvement_pct` (mặc định `+5%`) | **không apply** | Tránh đổi vô ích — giữ baseline |

> **Đã bỏ:** `max_shift_per_cycle`, `min_pct_per_provider` (10%), `max_pct_per_provider` (90%), **gradual shift** — mọi breach đủ chu kỳ đều hard cut về 0%.

**Admin mở lại provider / Mở lại dịch vụ (Dashboard §9.0):**

Sau hard cut hoặc hết cửa sổ bảo trì, admin bấm **Mở lại** / **Mở lại dịch vụ**:

```text
1. POST /scopes/{product}/{sku}/maintenance/reopen-service (Dashboard **Mở lại dịch vụ**)  
   hoặc POST /scopes/{product}/{sku}/routing/restore-baseline (**Mở lại** provider)
2. BuildReopenRoutingPlan — đọc metric mỗi provider từ agent_analysis_history (chu kỳ Agent gần nhất),
   fallback GetMetricsInWindow (mock/production) — KHÔNG dùng snapshot overview (routing_pct=0 → metric 0)
3. proposed_pct = baseline_pct (biz), round Σ=100
4. UpdateRouting(proposed) → trigger_type=manual_baseline; recovery_apply_cycle_id = latest cycle
5. Poll overview + Agent: grace `recovery_apply_cycle_id` — **chặn auto shift routing** trong chu kỳ hiện tại; **không** chặn `ShouldForceAutoMaintenance` / `ShouldForceAutoMaintenanceAllProviders` khi metric vẫn vi phạm (VD 1 provider routing 100% vẫn đỏ sau Mở lại)
6. Chu kỳ Agent tiếp theo: metric phản ánh phân bổ mới → auto/routing plan hoạt động bình thường
```

> **Vì sao không dùng metric live khi bấm Mở lại:** provider inactive (`routing_pct=0`) trên dashboard luôn hiển thị metric **0** — nếu auto apply ngay sẽ coi như không đổi được routing (ESALE vẫn 100%) và hoàn tác baseline.

**Không hiện / không cho Mở lại khi:** có `pending_plan` hoặc `pending_maintenance` chờ duyệt; đang có cửa sổ bảo trì active (`maintenance` — cột provider **disabled**, metric **0**).

**Bước 3 — Tính `expected_success`:**

```text
current_weighted = Σ (current_pct[i] * latest_success[i]) / 100
expected_success = Σ (new_pct[i]     * latest_success[i]) / 100

apply_plan = (expected_success - current_weighted) >= min_improvement_pct
```

Nếu `apply_plan == false` → output `Recommendation: monitor`, **không** gọi UpdateRouting.

**Timeline hồi phục sau duyệt / auto routing** (mặc định `scheduler_interval_min = 5`):

| Thời điểm | UI health | Ghi chú |
| --------- | --------- | ------- |
| Chu kỳ apply (duyệt xong) | 🔴 (thường vẫn đỏ nếu metric chưa hồi) | Ghi `recovery_apply_cycle_id` trên `routing_scope_state` |
| **+1 chu kỳ agent** | 🟡 **RECOVERING** | Luôn vàng — đang theo dõi sau routing, **không** nhảy thẳng xanh |
| **+2 chu kỳ agent** | 🟢 nếu metric **OK** (không breach); 🔴 nếu vẫn breach | Xóa cờ recovery khi xanh |

`routing_recovery_cycles` trong `agent_settings` = **1** (số chu kỳ ổn định sau vàng trước khi xanh — tổng **2** chu kỳ kể từ apply).

**Bước 4 — Khôi phục về baseline (recovery → Pha C):**

Áp dụng khi **một trong hai** điều kiện scope đang lệch baseline:

| Nguồn lệch | Cờ theo dõi | Khi nào restore |
|------------|-------------|-----------------|
| Agent điều chỉnh (Pha B) | `traffic_pct ≠ baseline_pct` (do Agent/approve plan) | Metric ổn định ≥ N chu kỳ |
| Admin chỉnh tạm (Pha A′) | `routing_scope_state.pending_restore = 1` | Metric ổn định ≥ N chu kỳ |

**Điều kiện "ổn định"** (chung cho cả hai): provider/scopes liên quan **không còn `breached`** sau **2 chu kỳ agent** kể từ apply routing (chu kỳ +1 vàng, chu kỳ +2 xanh nếu OK). Tuỳ chọn restore baseline: metric hồi thêm `routing_recovery_cycles` chu kỳ (mặc định **1**) với `success_rate ≥ success_rate_min_pct + recovery_buffer` (buffer 5%).

**Luồng restore:**

1. Agent sinh **Recommendation** `restore_baseline` (kèm diff: *Hiện tại* vs *Baseline Business*).
2. Tuỳ `routing_scope_state.auto_action` của scope:
   - `recommend_only` / `time_window` (ngoài giờ) → chờ admin approve.
   - `auto` / `time_window` (trong giờ) → tự apply: `traffic_pct = baseline_pct`.
3. Ghi `agent_change_log`: `trigger_type=auto`, `reason="recovery_to_baseline"` hoặc `"restore_after_manual_override"`.
4. `UPDATE routing_scope_state SET pending_restore=0` (nếu có).
5. SSE `agent_change_applied` + email (cooldown §8.9).

**Ví dụ áp dụng — TOPUP_VINA (baseline 70/20/10):**

| Provider | Baseline | Current (đang sự cố) | Success_rate | Phân loại | Drop | New_pct |
|----------|---------:|---------------------:|-------------:|-----------|-----:|--------:|
| ESALE    | 70%      | 70%                  | 78%          | **bad**   | -70  | **0%**  |
| IMEDIA   | 20%      | 20%                  | 98%          | **good**  | —    | 47%     |
| SHOPPAY  | 10%      | 10%                  | 96%          | **good**  | —    | 53%     |

Tính (hard cut — success hoặc fail_txn vượt ngưỡng): `drop[ESALE] = 70`; `donor_pool = 70`; phân cho IMEDIA/SHOPPAY theo `success_rate` (98 và 96) → IMEDIA `+70*98/194 ≈ +35.4`, SHOPPAY `+70*96/194 ≈ +34.6`. Sau round → ESALE **0** | IMEDIA 55 | SHOPPAY 45 (Σ=100). `expected_success` ≈ 0.55·98 + 0.45·96 ≈ **97.1%** vs current `≈ 83%` → **apply**.

> Success thấp cũng hard cut về **0%** — không còn gradual shift / floor 10%.

**Ví dụ — GARENA SKU 100000 (nhiều provider vi phạm pending GD):**

| Provider | Hiện tại | Pending (GD) | Vi phạm | Đề xuất |
|----------|----------|--------------|---------|---------|
| ESALE    | 40%      | 0            | Không   | **100%** |
| IMEDIA   | 35%      | 5 (≥ ngưỡng) | Có      | **0%**   |
| SHOPPAY  | 25%      | 5 (≥ ngưỡng) | Có      | **0%**   |

Hard cut: `drop[IMEDIA]+drop[SHOPPAY]=60` → toàn bộ sang ESALE (provider duy nhất healthy).

> Sau khi ESALE hồi metric OK ở **chu kỳ +2** sau duyệt → 🟢; tuỳ chọn recovery baseline về 70/20/10 (Pha C).

#### 8.6.4 Checklist triển khai

- [ ] Bảng `routing_baseline` (hoặc cột `baseline_pct` trong `routing_config` §13.3) — **chỉ Admin** ghi; Agent đọc-only.
- [ ] Tách rõ trong code: `internal/store/baseline.go` (Pha A) vs `internal/rules/router.go` (Pha B–C).
- [ ] Guardrails `min_pct`, `max_pct`, `min_improvement_pct` đưa vào `agent_settings` (§13.4) — admin configurable, không hard-code.
- [ ] `expected_success` < threshold cải thiện → output **Recommendation `monitor`**, không sinh Routing Plan để approve.
- [x] Recovery: **+1 chu kỳ → 🟡**, **+2 chu kỳ (OK) → 🟢** qua `recovery_apply_cycle_id` + `overlayRecoveryState` (`internal/agent/recovery.go`).
- [ ] Recovery baseline restore (Pha C) — kịch bản G đầy đủ.
- [ ] UI Settings: form routing + checkbox `set_as_baseline` (§9.5.4.1); validate Σ = 100.
- [ ] Routing Plan UI hiển thị **3 cột mốc:** *Baseline (Business)* | *Hiện tại* | *Đề xuất* — để ops thấy rõ Agent đang lệch khỏi baseline bao nhiêu.
- [ ] Thẻ / Topup data: baseline + plan **per SKU** (mỗi SKU một section riêng); Topup tiền: một section provider duy nhất.
- [ ] Test guardrail: `active_count == 1` (kể cả `total_count > 1`) → bỏ shift, chuyển nhánh `SetMaintenance` (§8.5, §10.2b).
- [ ] Test recovery: mock `random_spike` → spike → hồi → routing tự khôi phục baseline.
- [ ] Admin PUT `set_as_baseline=true/false` — kịch bản §10.12.

#### 8.6.5 Admin PUT tỉ lệ routing — checkbox `set_as_baseline`

Khi Admin sửa tỉ lệ routing trên UI Settings (§9.5.4.1), mỗi lần **Lưu** gọi `PUT /products/{code}/routing` kèm checkbox:

| Checkbox UI | `set_as_baseline` | Hệ thống hiểu | Ghi DB |
|-------------|-------------------|---------------|--------|
| ☑ **Đặt làm baseline mới (Business)** | `true` | Tỉ lệ mới = **baseline chính thức** — dùng làm mốc recovery sau này | `baseline_pct` = `traffic_pct` = giá trị mới; `pending_restore=0` |
| ☐ **Chỉnh tạm thời** (mặc định khi đang sự cố) | `false` | Override **traffic đang chạy** — **giữ baseline cũ**; tự trả baseline khi ổn định | Chỉ `traffic_pct` = giá trị mới; `baseline_pct` **không đổi**; `pending_restore=1` |

**Request `PUT /products/{code}/routing`:**

```json
{
  "scope": "provider",
  "sku_code": "",
  "set_as_baseline": false,
  "routing": { "ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20 },
  "reason": "Giảm tạm ESALE trong giờ cao điểm"
}
```

**SKU mode (`scope=sku`):**

```json
{
  "scope": "sku",
  "sku_code": "20000",
  "set_as_baseline": true,
  "routing": { "ESALE": 60, "IMEDIA": 40 },
  "reason": "Business cập nhật baseline SKU 20k"
}
```

**Luồng backend (`internal/store/routing.go`):**

```text
PUT /products/{code}/routing
  → Validate: Admin role; Σ routing = 100; scope khớp service_type
  → BEGIN TX

  IF set_as_baseline == true:
    UPDATE routing_config SET baseline_pct = ?, traffic_pct = ?  (mỗi provider)
    UPDATE routing_scope_state SET pending_restore = 0
    INSERT config_audit_log (change_type=routing_baseline, before/after JSON)
    INSERT agent_change_log (trigger_type=manual_baseline, routing_before, routing_after)

  ELSE set_as_baseline == false:
    -- snapshot baseline hiện tại (không ghi đè)
    UPDATE routing_config SET traffic_pct = ? ONLY
    UPSERT routing_scope_state SET pending_restore = 1, manual_override_by = UPN, manual_override_at = NOW()
    INSERT agent_change_log (trigger_type=manual_temp, routing_before, routing_after,
                             metadata: { "baseline_unchanged": {...}, "pending_restore": true })

  COMMIT → SSE routing_updated
```

**Response 200:**

```json
{
  "product_code": "TOPUP_VINA",
  "scope": "provider",
  "sku_code": "",
  "set_as_baseline": false,
  "baseline_pct": { "ESALE": 70, "IMEDIA": 20, "SHOPPAY": 10 },
  "traffic_pct": { "ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20 },
  "pending_restore": true,
  "message": "Đã lưu tỉ lệ tạm — hệ thống sẽ trả về baseline khi ổn định"
}
```

**UI hiển thị sau khi lưu (Pha A′):**

```text
TOPUP_VINA — Routing
  Baseline (Business):  ESALE 70% | IMEDIA 20% | SHOPPAY 10%
  Đang chạy:            ESALE 50% | IMEDIA 30% | SHOPPAY 20%   ⚠ Chỉnh tạm — chờ khôi phục
  [Badge: pending_restore]
```

**Quy tắc tương tác với Agent:**

| Tình huống | Hành vi |
|------------|---------|
| `pending_restore=1` + Agent muốn routing (Pha B) | Agent **vẫn** có thể điều chỉnh `traffic_pct`; recovery cuối cùng về **`baseline_pct`** (Business), không về tỉ lệ admin tạm |
| Admin `set_as_baseline=true` giữa lúc `pending_restore=1` | Xóa cờ `pending_restore`; baseline mới = tỉ lệ vừa lưu |
| Recovery (Pha C) khi `pending_restore=1` | `traffic_pct ← baseline_pct` (baseline **không** đổi từ lúc admin chỉnh tạm) |

**Checklist triển khai §8.6.5:**

- [ ] API `GET /products/{code}/routing` — trả `baseline_pct`, `traffic_pct`, `pending_restore` per scope/SKU.
- [ ] API `PUT /products/{code}/routing` — body bắt buộc `set_as_baseline` (boolean).
- [ ] Bảng `routing_scope_state` (§13.3.1) — cờ `pending_restore`.
- [ ] UI checkbox + helper text tiếng Việt (§9.5.4.1).
- [ ] Agent pipeline: trước recovery check `pending_restore` OR `traffic_pct ≠ baseline_pct`.
- [ ] Test §10.12: admin temp → ổn định → auto restore baseline.

### 8.7 Audit log thay đổi routing (`agent_change_log`)

Mọi lần **OpsOne thực thi** thay đổi routing (`UpdateRouting`) ghi **snapshot trước/sau** vào bảng `agent_change_log` — phục vụ audit nội bộ / truy vết sự cố. **Không** có API hay UI xem lịch sử / rollback; khôi phục routing dùng **Mở lại provider** (`POST /scopes/.../routing/restore-baseline`) hoặc **Mở lại dịch vụ** (`POST /scopes/.../maintenance/reopen-service`).

**Phạm vi ghi log (bắt buộc):**

| `change_type` | Nguồn thay đổi | Ghi vào `agent_change_log` |
| ------------- | -------------- | -------------------------- |
| `routing` | OpsOne `trigger_type=auto` | Có |
| `routing` | Admin PUT + `set_as_baseline=true` | Có (`manual_baseline`) |
| `routing` | Admin PUT + `set_as_baseline=false` | Có (`manual_temp`) |
| `routing` | Admin **Mở lại provider** | Có (`manual_baseline`) |
| `routing` | Admin routing tùy chỉnh (`POST /scopes/.../routing/apply`) | Có (`manual_temp`) |
| `routing` | Admin approve plan → `admin_approve` | Có |
| `agent_settings` | Admin `PUT /config` | `config_audit_log` riêng |

> **Tách bạch ENUM `trigger_type`:**
> - `agent_change_log`: `auto`, `admin_approve`, `manual_baseline`, `manual_temp` — mọi thay đổi routing.
> - `maintenance_windows`: `opsone_recommend`, `opsone_auto`, `admin_manual` — **không** dùng cho routing log.
>
> `agent_analysis_history` = metric theo chu kỳ phân tích; **`agent_change_log`** = thay đổi **cấu hình routing thực tế** trên hệ thống.

**Luồng ghi log khi UpdateRouting:**

```text
BEGIN TRANSACTION
  1. SELECT routing_config (product, sku?) → routing_before (JSON)
  2. Validate scope + guardrail (§6.7)
  3. UPDATE routing_config → routing_after
  4. INSERT agent_change_log (
       change_type=routing,
       change_status=applied,
       trigger_type=auto|admin_approve|manual_temp|manual_baseline,
       cycle_id, routing_plan_id, incident_id,
       reason, executed_by='opsone-agent'|admin
     )
  5. UPDATE routing_plans.status = executed (nếu từ approve)
COMMIT
  6. SSE /events: agent_change_applied
```

**Định dạng snapshot JSON (`routing_before` / `routing_after`):**

```json
{
  "product_code": "TOPUP_VINA",
  "scope": "provider",
  "sku_code": "",
  "providers": { "ESALE": 70, "IMEDIA": 20, "SHOPPAY": 10 }
}
```

**Khôi phục routing (thay cho rollback):**

- **Mở lại provider** — `POST /scopes/{product}/{sku}/routing/restore-baseline` → áp `baseline_pct`, ghi `manual_baseline` + `recovery_apply_cycle_id`.
- **Mở lại dịch vụ** — `POST /scopes/{product}/{sku}/maintenance/reopen-service` → hủy BT active + baseline biz (atomic).

**Checklist triển khai:**

- `internal/store/routing_update.go` — `InsertAgentChangeLog` trong transaction `UpdateRouting`.
- **Không** triển khai `GET /agent-changes`, rollback API, trang `/changes`.

### 8.8 Việc cần làm khi triển khai — Output

- Template JSON hoặc markdown cho từng loại output **+ Health Status**.
- Kênh gửi: **UI responsive** (ưu tiên) — icon trạng thái cập nhật realtime; **email ops** §8.9; (tuỳ chọn) Slack.
- Không trùng Incident trùng id trong cùng ngày (idempotent).
- Mỗi UpdateRouting có dòng tương ứng trong `agent_change_log` (audit DB).
- Maintenance output luôn có `starts_at` + `ends_at`; lifecycle scheduled→active→completed.

### 8.9 Email thông báo vận hành & leo thang provider (chat)

Sau khi OpsOne **thực thi routing** (`UpdateRouting`) hoặc **kích hoạt bảo trì** (`SetMaintenance` → `scheduled`/`active`), nếu dịch vụ vẫn **đỏ** hoặc metric xấu → **gửi email** team vận hành với đầy đủ ngữ cảnh và **gợi ý liên hệ provider** qua nhóm chat nội bộ.

**Điều kiện gửi mail (OR — ít nhất một):**


| Điều kiện               | Nguồn                           | Ghi chú                                                               |
| ----------------------- | ------------------------------- | --------------------------------------------------------------------- |
| **Vượt ngưỡng product** | `product_alert_thresholds` §1.2 | fail %, pending %, success min, **fail_txn_count**, error_event_count |
| `health_status`         | `red`                           | Sau EvaluateThresholds                                                |
| Incident                | `open` + severity ≥ medium      |                                                                       |
| Vừa thực thi            | routing / maintenance           | Kèm ngưỡng bị vượt trong mail                                         |


**Fallback ngưỡng email:** nếu product không override → dùng `agent_settings.notification_`* (§13.4).

**Cooldown:** cùng `(product, provider, sku, trigger_event)` không gửi lại trong `notification_cooldown_min` (mặc định 60 phút) — dedupe qua `notification_log.dedupe_key`.

**Luồng (Go `internal/notify`):**

```text
EvaluateThresholds() — vượt ngưỡng product §1.2
  → IF should_act → UpdateRouting / SetMaintenance commit OK
  → IF should_alert_email (vượt ngưỡng HOẶC vừa hành động)
  → Load metrics + breach_reasons + action summary
  → IF alert_conditions AND notification_enabled
  → Load provider_chat_escalation (provider bị ảnh hưởng)
  → Render template (HTML + text/plain)
  → SMTP send (hoặc NOTIFICATION_MOCK=true → chỉ ghi log)
  → INSERT notification_log (status=sent|failed)
```

**Cấu hình leo thang chat provider** (`provider_chat_escalation`):


| Trường            | Ví dụ                        | Mô tả                              |
| ----------------- | ---------------------------- | ---------------------------------- |
| `chat_app_name`   | `Microsoft Teams`            | Tên app chat team dùng             |
| `chat_group_name` | `[OpsOne] ESALE NOC`         | Tên nhóm / channel                 |
| `mention_tags`    | `@esale-oncall @provider-n2` | Tag người cần nhắc (copy vào chat) |


Seed mặc định:


| Provider | App             | Nhóm                      | Tag                        |
| -------- | --------------- | ------------------------- | -------------------------- |
| ESALE    | Microsoft Teams | `[OpsOne] ESALE Support`  | `@esale-oncall`            |
| IMEDIA   | Microsoft Teams | `[OpsOne] IMEDIA Support` | `@imedia-ops`              |
| SHOPPAY  | Zalo            | `OpsOne x SHOPPAY`        | `@shopay-support @n2-lead` |


**Mẫu nội dung email (text):**

```text
Subject: [OpsOne 🔴] TOPUP_VINA — đã điều phối routing sau sự cố ESALE

── Tình trạng hiện tại ──
Sản phẩm:    TOPUP_VINA (Topup Vinaphone)
Nhà cung cấp: ESALE
Trạng thái:  🔴 Đang có vấn đề
Thành công:  78%  |  Pending: 18%  |  Lỗi: 12%
Ngưỡng vượt: Tỷ lệ lỗi 12% > ngưỡng 10%; Số GD lỗi 142 > ngưỡng 120
Sự cố:       #20260604-001 (Mức Cao, Đang mở)
Lỗi hàng đầu: -3004 (142 lượt), -22 (38 lượt)

── Hành động OpsOne vừa thực hiện ──
Điều phối routing (phạm vi: toàn provider):
  Trước:  ESALE 70% | IMEDIA 20% | SHOPPAY 10%
  Sau:    ESALE **0%** | IMEDIA 55% | SHOPPAY 45%
  Thời điểm: 2026-06-04 10:10 +07  |  Kích hoạt: tự động

── Leo thang — nhờ nhà cung cấp kiểm tra ──
Vui lòng vào nhóm chat và tag đội provider:

  Ứng dụng chat:  Microsoft Teams
  Nhóm:           [OpsOne] ESALE Support
  Tag:            @esale-oncall

Tin nhắn mẫu (copy & paste):
  "Team ESALE ơi, TOPUP_VINA qua ESALE đang pending 18%, lỗi 12%.
   OpsOne đã cắt traffic ESALE 70%→0%. Nhờ team kiểm tra hệ thống
   phía ESALE giúp ạ. Sự cố #20260604-001."

Bảng điều khiển: http://localhost:5173/incidents/20260604-001
```

**Mẫu email — sau bảo trì ZING (chỉ ESALE):**

```text
Subject: [OpsOne 🔴] ZING — bảo trì ESALE 10:15–11:15

── Bảo trì ──
Sản phẩm:    ZING (Thẻ Zing)
Nhà cung cấp: ESALE (chỉ một provider)
Bắt đầu:     2026-06-04 10:15
Kết thúc:    2026-06-04 11:15
Trạng thái:  Đang bảo trì

── Chỉ số (trước bảo trì) ──
Thành công 42% | Pending 22% | Lỗi 35%

── Leo thang ──
  Ứng dụng: Microsoft Teams | Nhóm: [OpsOne] ESALE Support | Tag: @esale-oncall

Tin nhắn mẫu:
  "Team ESALE, ZING chỉ qua ESALE đang lỗi nặng. OpsOne đã lên lịch
   bảo trì đến 11:15. Nhờ kiểm tra dịch vụ thẻ Zing bên ESALE giúp."
```

**JSON `notification_log.metrics_snapshot`:**

```json
{
  "success_rate": 78.0,
  "pending_rate": 18.0,
  "fail_rate": 12.0,
  "health_status": "red",
  "top_errors": [{"code": "-3004", "count": 142, "label": "Lỗi timeout"}],
  "breach_reasons": ["Tỷ lệ lỗi 12% vượt ngưỡng 10%", "Số GD lỗi 142 vượt ngưỡng 120"]
}
```

**API `GET /notifications` (mẫu item):**

```json
{
  "id": 7,
  "trigger_event": "routing_applied",
  "product_code": "TOPUP_VINA",
  "provider_code": "ESALE",
  "subject": "[OpsOne 🔴] TOPUP_VINA — đã routing sau sự cố ESALE",
  "status": "sent",
  "sent_at": "2026-06-04T10:10:05+07:00",
  "chat_escalation": {
    "chat_app_name": "Microsoft Teams",
    "chat_group_name": "[OpsOne] ESALE Support",
    "mention_tags": "@esale-oncall"
  }
}
```

**Checklist triển khai:**

- `internal/notify/smtp.go` — gửi mail; `NOTIFICATION_MOCK` cho dev.
- `internal/notify/template_vi.go` — render HTML/text **tiếng Việt**; embed tin nhắn chat mẫu.
- Hook sau `UpdateRouting` + `SetMaintenance` / approve maintenance trong agent pipeline bước 9.
- UI Settings §9.5.5 — recipients, bật/tắt, ngưỡng pending/fail, bảng provider chat.
- Admin `POST /notifications/test` — verify SMTP/MailHog.
- Không spam: dedupe + cooldown; log mọi lần thử gửi.

---

# Part IV — Frontend (React)

> Map code: `web/src/` — xem §2.3 API contract.

## 9. Bước 6 — UI giao tiếp (Chat & Voice, Responsive)

React 18 + Vite + TypeScript. Breakpoint: 768px / 1024px (§9.3).

**Routes (`react-router`):**


| Path             | Component         | Mô tả                                   |
| ---------------- | ----------------- | --------------------------------------- |
| `/`              | `Dashboard`       | §9.0 — bảng routing + health header |
| `/incidents`     | `IncidentsPage`   | Sự cố gần đây — bảng phân trang full-width §9.0 |
| `/settings`      | `Settings`        | §9.5 scheduler + thời lượng BT mặc định + mock (compact; không chỉnh ngưỡng/auto) |
| `/maintenance`   | `MaintenancePage` | Danh sách cửa sổ bảo trì (trang riêng)          |


**State:** TanStack Query — `health-status`, `dashboard/overview`, `incidents`; **refresh 60s** + SSE `/events` invalidate cache (`useSSE`).

### 9.0 Dashboard — triển khai thực tế (`web/src/pages/Dashboard.tsx`)

Màn chính gom **một bảng tổng quan** thay vì nhiều card rời (cập nhật 2026-06).

**1. Header + logo:** `useOverallHealth()` — badge 🟢🟡🔴 = trạng thái **xấu nhất** trong toàn bộ SKU (`dashboard/overview`) **hoặc** chu kỳ API; kèm `health_summary` tương ứng. **Hero Dashboard** (khi đỏ): `health_summary` hiển thị **tên dịch vụ có dấu** — VD `Sự cố: Data Mobifone, Thẻ Garena, Topup Viettel` — không còn mã thô `[DATA_MOBI …]`; backend `catalog.JoinProductDisplayLabels` + `FormatCycleHealthSummary` trên `GET /health-status`; frontend `dashboardHealth.formatLegacyIncidentSummary` fallback từ overview rows.

**2. Bảng routing tổng quan** (`ServiceOverviewTable` ← `GET /dashboard/overview`):

**Tab loại dịch vụ** (`Dashboard.tsx` — `service-tabs`): ba tab **Thẻ** (`card`) · **Topup** (`topup`) · **Data** (`topup_data`); lọc `rows` theo `service_type` (API) hoặc fallback prefix `DATA_` / `TOPUP_`. Mỗi tab hiển thị badge số dòng.

**Icon trạng thái trên tab:**

| Điều kiện | Icon tab |
| --- | --- |
| Có ≥ 1 SKU `pending_plan` chờ duyệt **và** scope `ShouldAutoApplyScope=false` trong tab | 🔴 **đỏ** (ưu tiên cao nhất) |
| Không có plan chờ — `max(effectiveRowHealth)` các **dòng SKU** trong tab (5 ngưỡng OR + live_metrics) | 🔴 > 🟡 > 🟢 |

Tooltip tab: *Có kế hoạch routing chờ duyệt* hoặc *Ổn định / Đang theo dõi / Có vấn đề*.

**Điều hướng SKU đỏ** (`RedSkuScrollNav` — góc bảng): khi có ≥ 1 SKU `effectiveRowHealth=red`, hiện **▲** · **🔴** · **`1/N`** · **▼**; bấm ▲/▼ cuộn tới `#overview-scope-{product}:{sku}` (smooth + flash highlight). Icon 🔴 cạnh bộ đếm (compact `HealthBadge`).

**Gom nhóm theo dịch vụ:** Các SKU cùng `product_code` (VD `DATA_VINA`: VNP20, VNP50, V50K, V100K) được **group** — cột **Dịch vụ** dùng `rowspan` cho hành động batch (§9.0.7, §9.5.2); **`product_label`** hiển thị trên **hàng Ngưỡng** (§9.5.3), không lặp trong ô `rowspan`. **SKU sort tăng dần** (số: 10000→20000→50000→100000). Mỗi nhóm nằm trong `<tbody class="overview-table__product-group">` riêng — viền trên + nền nhẹ. Cột **Dịch vụ** rộng (`min-width: 18rem`) — layout ngang: **Bảo trì** căn trái · **Chế độ BT / Routing** căn phải, căn **top** (`overview-table__product-cell-layout`). Topup `routing_mode=provider` (một scope) vẫn một dòng — `ScopeAutoEditor` ở cột điều khiển SKU (không lặp ở cột Dịch vụ).

**Thứ tự hàng trong mỗi nhóm product:**

1. **Ngưỡng cảnh báo** (`ProductThresholdEditor`) — **hàng đầu** nhóm; 5 input inline (§9.5.3).
2. Các dòng **SKU** — **6 chỉ số / provider** (`ProviderMetricCell` ← `provider_metrics`).
3. **Kế hoạch routing** hoặc **Đề xuất bảo trì** (nếu có) — **hàng con ngay dưới SKU** tương ứng.

| Cột | Nội dung |
| --- | -------- |
| Dịch vụ | **Một ô / nhóm SKU** (`rowspan`) — **không** lặp tên dịch vụ (tên ở hàng Ngưỡng). **Sku-mode** (≥2 SKU): trái — **Bảo trì dịch vụ** (`ServiceMaintenanceButton` batch SKU chưa BT) hoặc **Mở lại dịch vụ** + **Gia hạn bảo trì** (`ProductMaintenanceActions`, chỉ khi **mọi SKU** đang BT — §9.0.7); phải — **Chế độ BT / Routing** (`ScopeAutoEditor`, `skuCode=""`, §9.5.2). **Topup provider-mode:** ô trống (auto ở cột điều khiển SKU) |
| SKU | Mã SKU (`fmtSku`); topup provider-mode hiển thị `—`. **Khi BT active:** gộp với TT (`colspan=2`) — tên SKU + nhãn thời gian **4 dòng** (`SkuMaintenanceTimeLabel`, §9.0.1) |
| TT | Icon 🟢🟡🔴 — **`health_status` API** (consecutive §2.3.2): 🟡 = vi phạm lần 1; 🔴 = đủ chu kỳ liên tiếp hoặc có plan chờ duyệt; **gộp vào ô SKU** khi đang bảo trì (§2.3.2) |
| **ESALE / IMEDIA / SHOPPAY** | Mỗi cột 6 dòng (`provider_metrics`): **% Routing** · **% Success** · **% Pending** · **% Fail** · **Pending** · **Fail** — đỏ/xanh theo ngưỡng khi `routing_pct>0`; provider `routing_pct=0` (hard cut) → muted + metric **0** (chỉ hiển thị) + **Mở lại** → hàng con *Mở lại provider* (§9.0); **Lưu** baseline → `restore-baseline` (metric chu kỳ Agent + grace); BT active → disabled, không **Mở lại** |
| **Bảo trì + Chế độ BT / Routing** | **Một ô** (`colspan=2`, `overview-table__scope-controls-cell`) — layout giống cột Dịch vụ: trái **Bảo trì** · phải **Chế độ BT / Routing**, căn top. **Chưa BT:** `ServiceMaintenanceButton` per SKU. **Đang BT:** `ActiveMaintenanceCell` (Mở lại / Gia hạn, grid 2 cột ngang). **`ScopeAutoEditor`** per SKU — §9.5.2; `PUT /scopes/{product}/{sku}/auto` |

#### 9.0.1 Nhãn thời gian bảo trì active (cột SKU)

Khi `row.maintenance` active, nhãn nằm **dưới tên SKU** (không lặp ở cột Bảo trì). Format **4 dòng** (`formatDatetimeVi` — `dd/mm/yyyy hh:mm AM/PM`):

```text
Bảo trì từ
11/06/2026 12:54 AM
-
11/06/2026 01:54 AM
```

- Component: `SkuMaintenanceTimeLabel` ← `maintenanceActiveTimes()` trong `maintenanceWindow.ts`.
- Hover `title` = `maintenance.reason` hoặc `label_vi`.
- Fallback khi thiếu `starts_at`/`ends_at`: *Bảo trì đang hoạt động*.

#### 9.0.2 Khối SKU và phân cách hàng

Mỗi scope `(product, sku)` gồm **dòng SKU** + **hàng con** (nếu có: *Kế hoạch routing*, *Đề xuất bảo trì*, *Mở lại provider*):

- **Không** viền ngăn giữa dòng SKU và hàng con đầu tiên — gom một khối.
- **Viền dưới đậm** (`overview-table__scope-block-end`) sau hàng cuối của khối — phân tách SKU kế tiếp.
- Hàng con *Đề xuất bảo trì* / *Kế hoạch routing* luôn nằm **ngay dưới** SKU tương ứng.

#### 9.0.3 Hàng con「Đề xuất bảo trì」

Khi `pending_maintenance` và chưa có cửa sổ bảo trì active (`ServiceOverviewTable`):

| Cột (số thứ tự, bỏ cột Dịch vụ rowspan) | Nội dung |
| --- | --- |
| 1–2 (SKU + TT) | Nhãn *Đề xuất bảo trì* — căn trái, giữa dọc |
| 3–4 (ESALE + IMEDIA) | Lý do vàng (`maintenanceSuggestLabel`) — **1 dòng**, ellipsis + `title` |
| 5 → cuối (SHOPPAY + điều khiển SKU `colspan=2`) | **Từ** · **Đến** (`DateTimeLocalField` compact) · **Duyệt** · **Từ chối** — grid/flex một hàng, căn giữa dọc, chiều cao control **28px** |

- Mặc định **Từ/Đến** = now → now+60p; body approve `{ starts_at, ends_at }` (ISO).
- Nhãn thời gian: **Từ** / **Đến** (không dùng *Bắt đầu* / *Kết thúc*).

#### 9.0.4 Toast phản hồi scope

Toast sau duyệt/từ chối routing, bảo trì, mở lại, gia hạn… dùng **`scopeDisplayLabel`** (`dashboardRowOrder.ts`) — **không** lộ `product_code`:

```text
Data Mobifone · VNP20
```

Ví dụ: *Đã duyệt bảo trì Data Mobifone · VNP20 — cửa sổ bảo trì đã kích hoạt.*

#### 9.0.5 Gia hạn bảo trì (`ActiveMaintenanceCell` + `POST .../maintenance/extend`)

- Bấm **Gia hạn bảo trì** → inline **Từ/Đến** (prefill từ cửa sổ hiện tại) + **Lưu** / **Hủy**.
- **Lưu** không đổi thời gian → UI: *Thời gian bảo trì không thay đổi* (inline, nút **Lưu** vẫn bấm được; hint xóa khi sửa Từ/Đến); **không** gọi API.
- API (`ScopeMaintenanceExtend`): `UpdateActiveMaintenanceTimesForSKU` trả `RowsAffected=0` nhưng `CountActiveMaintenanceForSKU>0` → **400** `no_change`; không còn cửa sổ active → **404** *Không có cửa sổ bảo trì đang hoạt động*.
- So sánh unchanged: `maintenanceWindowUnchanged()` (`maintenanceWindow.ts`).

#### 9.0.6 Bảo trì dịch vụ thủ công (`ServiceMaintenanceButton`)

Khi scope **chưa** có cửa sổ bảo trì active — cột **Bảo trì** hiện nút **Bảo trì dịch vụ** (ẩn khi `row.maintenance` active → thay bằng `ActiveMaintenanceCell`):

1. Bấm **Bảo trì dịch vụ** → inline **Từ** · **Đến** (`DateTimeLocalField` compact) + **Lưu** / **Hủy** (class `service-maint-btn`).
2. Mặc định **Từ/Đến** = `defaultMaintenanceWindow(duration)` — now → now + N phút (`maintenance_default_duration_min` từ §9.5.5; fallback UI **60**).
3. **Lưu** hợp lệ → `POST /scopes/{product}/{sku}/maintenance/approve` body `{ reason, starts_at, ends_at }`; `reason` = `manualServiceMaintenanceReason()` (audit).
4. Validate cửa sổ: `maintenanceWindowError()` — lỗi hiện inline trước **Lưu**.

#### 9.0.7 Bảo trì / routing cấp dịch vụ (cột Dịch vụ)

Khi nhóm SKU cùng `product_code` (`routing_mode=sku`, ≥2 dòng SKU) — ô **Dịch vụ** (`rowspan`) bổ sung hành động batch. Layout `overview-table__product-cell-layout`: **trái** (`product-cell-main`) hành động BT · **phải** (`product-cell-auto`) `ScopeAutoEditor` — căn **top**, một hàng ngang.

**Chưa BT (còn ≥1 SKU chưa bảo trì):**

- `ServiceMaintenanceButton` (`service-maint-btn--product`) — **Bảo trì dịch vụ** cho **danh sách SKU chưa BT** (`skuCodes[]` = mọi row `!isSkuUnderActiveMaintenance`).
- **Lưu** → Dashboard gọi lần lượt `POST /scopes/{product}/{sku}/maintenance/approve` cho từng SKU (cùng cửa sổ Từ/Đến).
- **Ẩn** `ProductMaintenanceActions` khi chưa phải mọi SKU đang BT.

**Mọi SKU đang BT (`allSkusUnderMaintenance`):**

- Điều kiện hiện `ProductMaintenanceActions` (`ServiceOverviewTable`):
  - `group.rows.every(isSkuUnderActiveMaintenance)`;
  - `productActiveScopes.length === group.rows.length` (mỗi row có `row.maintenance` active).
- **Ẩn** `ServiceMaintenanceButton` batch (không còn SKU chưa BT).
- **Mở lại dịch vụ** → batch `POST .../maintenance/reopen-service` cho từng SKU.
- **Gia hạn bảo trì** → inline Từ/Đến + **Lưu** → batch `POST .../maintenance/extend` (cùng cửa sổ).
- Nút **Mở lại** / **Gia hạn** per SKU vẫn có ở cột điều khiển SKU khi SKU đó đang BT.

**Chế độ BT / Routing cấp dịch vụ:**

- `ScopeAutoEditor` (`level="product"`, `skuCode=""`) — compact: nhãn *Chế độ BT / Routing* trên · badge + ⋯ dưới; `PUT /scopes/{product}/auto`.
- **Lưu chế độ:** ⋯ → chọn chế độ → **Lưu** trong `ScopeAutoEditor` (cột Dịch vụ). **Không** dùng nút **Lưu** hàng *Ngưỡng cảnh báo* — nút đó chỉ `PUT /products/{code}/thresholds` (§9.5.3).
- Lưu `routing_scope_state` với `sku_code=""`.
- **Ưu tiên hiệu lực** (§9.5.2): `ResolveEffectiveScopeAuto` — row dịch vụ trước, fallback SKU.
- **Topup** `routing_mode=provider` (một dòng, `sku_code=""`): **không** lặp editor ở cột Dịch vụ.

**Hàng「Ngưỡng cảnh báo」** — chi tiết §9.5.3; tóm tắt cột:

| Ô | `colspan` | Nội dung |
| --- | --- | --- |
| Dịch vụ | 1 | **`product_label`** căn giữa (`overview-table__threshold-product-name`) |
| SKU + TT | 2 | Nhãn **Ngưỡng cảnh báo** |
| Provider + Bảo trì | `providers.length + 1` | 5 cụm input — **bắt đầu thẳng cột Provider**; trải thêm sang cột **Bảo trì** để giãn cách (**không** đổi `colgroup` width cột Provider) |
| Chế độ BT / Routing | 1 | Checkbox **Email** + **Lưu** ngưỡng (`PUT /products/{code}/thresholds`) — **không** lưu chế độ auto/routing |

**Hàng con「Kế hoạch routing」** (`RoutingPctEditor` — compact/readOnly): provider không support → ô `—`; validate tổng 100% và mỗi provider **0–100%** trước Duyệt. **`GET /dashboard/overview`** (poll ~60s): nếu scope còn plan `pending_approve`/`draft` và metric live vẫn vi phạm → **tái tính `proposed_pct`** từ snapshot, `UPDATE routing_plans.plan_json`; nếu không còn vi phạm → hủy plan pending. UI sync draft khi `proposed_pct` API đổi.

| Cột | Nội dung |
| --- | -------- |
| SKU + TT (`colspan=2`) | Nhãn *Kế hoạch routing* hoặc *Đề xuất routing* (`pending_plan.suggested`) |
| ESALE / IMEDIA / SHOPPAY | Ô % đề xuất (hoặc read-only) **thẳng cột provider**; provider **không support** product/SKU → `—` |
| Bảo trì + Chế độ (`colspan=2`) | **Duyệt** / **Từ chối** căn **phải**; lỗi validate (tổng ≠ 100%, ESALE 9%…) trên cột này. Plan DB: `POST /routing-plans/{id}/approve` body `{ "proposed_pct": {...} }`. Synthetic: `POST /scopes/{product}/{sku}/routing/approve` |

**Nút「Mở lại」provider** (`routing_pct=0`, không BT): mở hàng *Mở lại provider* — **Trả lại** điền `baseline_pct`; **Lưu** → `restore-baseline` (nếu = baseline, metric chu kỳ Agent + grace) hoặc `routing/apply` (`manual_temp` nếu admin sửa %).

**Điều kiện ẩn nút Mở lại:** `pending_plan` / `pending_maintenance` chờ duyệt; cửa sổ bảo trì active; đang mở hàng *Mở lại provider* scope khác.

**Điều kiện hiện hàng đề xuất:** `ShouldAutoApplyScope=false` (§9.5.2 — `recommend_only`, hoặc `time_window` **ngoài** `[window_start, window_end)`) **và** (provider active vi phạm **1/5 chỉ số** **và** `consecutive_breach_cycles >= consecutive_cycles_required` — **hoặc** đã có plan DB `pending_approve`/`draft` refresh mỗi poll 60s). SKU **Tự động** hoặc **Tự động theo khung giờ (trong khung)** → **không** hiện hàng đề xuất; poll `GET /dashboard/overview` tự `UpdateRouting` / `SetMaintenance` (kể cả 🟡 + force §9.5.2). **Ẩn** khi hết vi phạm hoặc có cửa sổ bảo trì active.

- Không còn cột metric tổng (S/P/F %, GD tổng) — chỉ metric **per provider**.
- Bảng scroll ngang nhẹ khi cần; text dài truncate + `title`.
- **Duyệt:** Admin sửa % trước approve; bước **1%**; **0%** khi cắt provider vi phạm (hard cut).

**3. Trang「Sự cố」** (`IncidentsPage` + `IncidentsTable` ← `GET /incidents?page=&page_size=`):

| Cột | Nội dung |
| --- | -------- |
| Thời gian | `created_at` |
| Mã | `incident_id` — text thường (không link detail) |
| Mức độ | Icon + severity |
| Sản phẩm / Provider / SKU | |
| Tóm tắt | Wrap nhiều dòng (`white-space: normal`) |
| TT + Xử lý | Gộp: trạng thái + `handled_by` / `handled_at` / `resolution_action` |
| Người xử lý | `handled_by` |
| Thời gian xử lý | `handled_at` định dạng `vi-VN` |

**Layout:** Full width; **không** scroll ngang; **không** route `/incidents/:id` (đã bỏ `IncidentDetail`).

**Phân trang:** Mặc định **10 dòng/trang**; nút **Trước / Sau**; hiển thị *Trang X/Y · N sự cố*; query key `['incidents', page, pageSize]`; refresh **60s** + SSE invalidate.

**Navigation (`Layout.tsx`):** Menu **tab ngang trên header** — Dashboard · Sự cố · Bảo trì · Cấu hình (không còn sidebar trái). **Logo ZaloPay** (32×32) cạnh chữ **OpsOne**; badge 🟢🟡🔴 global cạnh brand. **Favicon** tab trình duyệt: `web/public/favicon.png` (logo ZaloPay).

**Chat (`ChatWidget.tsx` + `useChatDock.ts` + `useVoiceInput.ts` + `useOpsOneWake.ts`):** Widget nổi panel **~800×780px** (mobile ~full width / 88vh); nhãn bubble assistant **OpsOne**; nhãn user **Anh/Chị + tên** (`userBubbleLabel`, §9.2.1); phản hồi AI full-width, `pre-wrap`; **một vùng scroll** trên feed. **Kéo** nút Chat hoặc header để **dock** 4 góc; resize 4 góc (`useChatResize`); vị trí `localStorage` (`opsone-chat-corner`). **Enter** gửi, **Shift+Enter** xuống dòng. **Nhận diện danh tính (identity detection)** client-only khi user tự nói tên/xưng hô — §9.2.1. **Voice** `vi-VN`: wake **`alo`** (§9.2); Mic toggle; im lặng **2s** → gửi kèm `input_source: voice`, `stt_raw` §7.6.5.5; *đóng chat* (mic vẫn bật); *tắt mic* / *bye bye* (tắt mic + đóng chat). Cần **click/tap một lần** trên trang trước wake nền. Không route `/chat` riêng.

#### 9.2.1 Tự động nhận diện danh tính & avatar (Identity Detection)

**Code:** `chatIntro.ts`, `chatOnboarding.ts`, `chatUserProfile.ts`, `ChatAvatar.tsx`; `web/public/chat-avatars.png` (sprite **7×4**), `favicon-64.png` (avatar OpsOne).

**Mở chat:** OpsOne gửi lời chào đơn giản — giới thiệu khả năng hỗ trợ metrics/sự cố/routing. **Không chủ động hỏi tên.**

**Nhận diện danh tính:**
- Agent chỉ cập nhật xưng hô/tên khi user **tự giới thiệu** hoặc nói lệnh kèm tên (vd: "Anh Khiêm đây", "Ghi nhận cho anh Tuấn nhé", "Tôi là Lan Anh").
- Khi phát hiện thông tin định danh mới (`inferProfileFromSingleMessage`), Agent phản hồi xác nhận và cập nhật avatar ngay lập tức.
- **Tuổi không bắt buộc:** suy từ lời nói (nếu có) để chọn avatar, mặc định là **trung niên**.

**Nhãn bubble:** `Anh Khiêm` / `Chị Lan` — giữ **dấu tiếng Việt**; sửa ASR (`ăn`→`anh`). Không hiển thị *Ten Khiem* / chữ *tên* trong nhãn.

**Avatar:** silhouette xám → sprite `(gender, ageGroup)` khi user tự định danh; OpsOne = favicon.

#### 9.2.2 Tự động mở chat khi có đề xuất mới (Auto-chat-open)

**Tính năng:** Khi hệ thống phát hiện đề xuất routing plan hoặc bảo trì mới, backend phát sự kiện SSE `pending_suggestions` → **chat tự động mở** và hiển thị danh sách đề xuất với **lời nhắc hành động**.

> [!NOTE]
> Để tránh gây phiền hà khi vừa tải trang, OpsOne được cấu hình **chặn tự động mở trong 3 giây đầu tiên**. Mọi đề xuất nhận được trong thời gian này sẽ chỉ hiển thị khi user chủ động mở chat hoặc sau khi hết thời gian chờ, đảm bảo dashboard được hiển thị ổn định trước.

**Backend (internal/api/sse.go):**
- Mỗi giây, SSE listener kiểm tra `getPendingSuggestionsForSSE()` — tổng hợp routing plan + bảo trì pending từ DB
- Nếu `has_suggestions=true` → gửi event `pending_suggestions` kèm dữ liệu:
  ```json
  {
    "has_suggestions": true,
    "plan_count": 2,
    "maintenance_count": 1,
    "routing_plans": [
      {
        "product_code": "ZING",
        "sku_code": "20k",
        "status": "pending",
        "plan_id": "P001",
        "proposed_pct": { "ESALE": 60, "IMEDIA": 40 },
        "reason_vi": "Success rate thấp"
      }
    ],
    "maintenance_suggestions": [
      {
        "product_code": "GARENA",
        "sku_code": "50k",
        "provider_code": "ESALE",
        "detail": "Error rate cao",
        "id": "M001"
      }
    ]
  }
  ```

**Frontend (web/src/components/ChatWidget.tsx):**
- Listener `useEffect` lắng nghe event `pending_suggestions` từ SSE
- Gọi `formatSuggestionSystemMessage()` để định dạng thông báo tiếng Việt với emoji:
  ```
  📢 **Có việc mới cần xử lý!**

  🔄 **Đề xuất Routing:** (Thay đổi phân phối traffic)
  • ZING/20k → ESALE: 60% / IMEDIA: 40% (Success rate thấp)
  • (và nhiều hơn nữa...)

  🔧 **Đề xuất Bảo trì:**
  • GARENA/50k — Error rate cao (và nhiều hơn...)

  💡 **Hành động:**
  • Gõ "xem pending" để xem chi tiết
  • Admin: duyệt/từ chối ngay hoặc vào Dashboard
  ```
- Thêm system message vào chat
- Gọi `setOpen(true)` → mở widget nếu chưa mở
- Gọi `scrollFeedToBottom()` → cuộn đến message mới
- Kiểm tra duplicate (message cuối có chứa emoji `📢`?) → tránh mở lại nhiều lần

**Quy trình User:**
```
Dashboard đang xem
    ↓
Backend phát hiện routing plan mới
    ↓
SSE gửi pending_suggestions event
    ↓
Chat tự mở + hiển thị "📢 Có việc mới cần xử lý!"
    ↓
User xem danh sách đề xuất rõ ràng
    ↓
Gõ "xem pending" hoặc "duyệt" + product/SKU
```

**Code:**
- Backend: `internal/api/chat_actions.go` → `getPendingSuggestionsForSSE()`, `formatSuggestionSystemMessage()`
- Backend: `internal/api/sse.go` → thêm event `pending_suggestions` trong ticker loop
- Frontend: `web/src/components/ChatWidget.tsx` → thêm `useEffect` lắng nghe SSE event
- Frontend: `web/src/api/client.ts` → `eventsUrl()` trả về `/events` endpoint

#### 9.2.3 Tự động cập nhật dữ liệu sau hành động bảo trì (Auto-refetch)

**Tính năng:** Sau khi user thực hiện hành động bảo trì (duyệt/từ chối/gia hạn/mở lại/trả routing/chế độ tự động), chat hiển thị thông báo confirm rồi tự động refetch dữ liệu Dashboard **sau 1.2 giây** để cập nhật ngay. **Không reload trang** — giữ lại chat history.

**Cơ chế:**
1. **Keyword detection** trong response từ backend:
   - `Đã bật bảo trì` → Set maintenance
   - `Đã gia hạn` → Extend maintenance window
   - `Đã từ chối` → Reject routing plan or maintenance
   - `Đã duyệt` → Approve routing plan or maintenance
   - `Đã mở lại` → Reopen service
   - `Đã trả` → Restore baseline routing
   - `Đã cập nhật` → Change scope auto mode

2. **Frontend (web/src/components/ChatWidget.tsx):**
   - Mutation `send.onSuccess()` callback:
     1. Invalidate all query keys: `dashboard-overview`, `maintenance`, `incidents`, `routing-plans`, `health-status`
     2. Kiểm tra reply message có chứa action keyword nào không
     3. Nếu match → Sau 1.2 giây, gọi `queryClient.refetchQueries()` cho 5 queries trên (silent refetch ngầm trong background)
   - Không gọi `window.location.reload()` → chat vẫn hiển thị

3. **Auto-polling mỗi 1 phút:**
   - `useEffect` hook: `setInterval(() => refetchQueries(), 60 * 1000)`
   - Chạy liên tục khi component mount (chat widget vẫn open)
   - Cập nhật data tự động ngay cả khi user chỉ xem (không hành động)
   - Silent error handling — không báo lỗi nếu refetch thất bại

**Quy trình User:**
```
Chat: "Duyệt thẻ Garena 50K bảo trì"
    ↓
Backend xử lý → Update DB → return "Đã duyệt bảo trì thẻ Garena 50K..."
    ↓
Chat hiển thị reply message (user thấy confirm) ✅
    ↓
1.2 giây → Refetch data ngầm (không reload page)
    ↓
Dashboard tự cập nhật:
  - Bảo trì active đổi status
  - Health status đổi
  - Incident list update
  - Chat history vẫn giữ nguyên ✅
    ↓
[Mỗi 60 giây] → Auto-polling refetch data để luôn fresh
```

**Lợi ích:**
- ✅ **Giữ chat history** — không reload page
- ✅ **User không cần F5** thủ công
- ✅ **Data luôn fresh** từ server (refetch sau action + auto-polling 1 phút)
- ✅ **Trải nghiệm smooth** — không gián đoạn hội thoại
- ✅ **Silent refetch** — background update, không phiền người dùng

**Code:**
- Frontend: `web/src/components/ChatWidget.tsx`
  - `send.onSuccess()`: keyword detection + refetch queries (không reload)
  - New `useEffect`: auto-polling `setInterval(refetchQueries, 60000)`

UI là **mặt tiếp xúc với người vận hành** — dùng được trên **điện thoại** và **PC**, nhận lệnh qua **chat** hoặc **micro**.

### 9.1 Vai trò


| Chức năng              | Mô tả                                                                                |
| ---------------------- | ------------------------------------------------------------------------------------ |
| **Cấu hình**           | Admin chỉnh scheduler, mock (§9.5); **ngưỡng per product** + **auto per dịch vụ / SKU** trên Dashboard |
| **Trạng thái**         | Icon 🟢 / 🟡 / 🔴 — tổng quan sau mỗi chu kỳ phân tích (§8.2)                        |
| **Bảng tổng quan**     | Routing hiện tại mọi product/SKU + bảo trì + 1 plan chờ/scope — refresh 60s + SSE   |
| **Feed sự cố**         | Trang `/incidents` — bảng phân trang full-width (không drill-down)                  |
| **Chat**               | User gõ câu hỏi hoặc lệnh → Agent trả lời (on-demand)                                |
| **Voice**              | Wake **alo**; Mic toggle; im lặng 2s → gửi; §9.2 + §9.2.1 identity detection tự động |
| **Hành động**          | Approve / từ chối Routing Plan + đề xuất bảo trì (khi scope `recommend_only` hoặc `time_window` ngoài giờ); **Mở lại provider** / **Mở lại dịch vụ** |
| **Thông báo email**    | Xem mail đã gửi; cấu hình nhóm chat provider (§8.9, §9.5.5)                          |


> Scheduler **vẫn phân tích** theo chu kỳ đã cấu hình. **Auto per dịch vụ / SKU** (§9.5.2 — `ResolveEffectiveScopeAuto`) quyết định **có tự gọi UpdateRouting hay không**; trang Cấu hình chỉ đặt chu kỳ.

### 9.2 Hai kênh nhập lệnh

**Hai tầng (§7.6.5.4):** Tầng **A** = voice UI (mic/chat); Tầng **B** = Agent backend (`POST /chat`) thực thi vận hành. Agent **học dần** từ log + feedback — §7.6.5.5.

**Chat (text):**

```text
[Ô nhập tin]  "Giải thích incident #20260604-001"
              "ZING SKU 20k đang routing thế nào?"
              "Approve routing plan vừa đề xuất"
              "bảo trì giúp tôi thẻ Garena toàn bộ"
              "duyệt" / "ok" (sau khi bot liệt kê pending)
              "Mở lại routing TOPUP_VINA về baseline"
```

**Voice (micro) — hội thoại liên tục + wake:**

```text
[Nền — sau 1 lần click trang]  nói "alo" hoặc "bật mic"  →  Mở chat + bật Mic ●
[Bấm Mic]  →  Bật session (Mic ●)  →  STT vi-VN continuous
              "Anh Khiêm 32 tuổi"   (tự định danh — §9.2.1)
              (im lặng 2 giây → xác nhận danh tính & cập nhật avatar)
              "Cho tôi biết ESALE ZING hôm nay thế nào"
              (im lặng 2 giây → gửi câu tiếp…)
Nói "đóng chat"     →  Thu gọn panel (mic VẪN bật)
Nói "tắt mic" / "bye bye"  →  Tắt mic + đóng chat
[Bấm Mic lần nữa]  →  Tắt session
```

**Yêu cầu voice (`useVoiceInput.ts` + `useOpsOneWake.ts`):**

- **Wake word / bật mic:** `alo` (và biến thể STT: `a lo`, `allo`, `alo ơi`, …) hoặc **`bật mic` / `mở mic`** (`VOICE_MIC_ON_PHRASES`) — listener nền khi chat đóng + mic tắt + đã `speechPrimed`.
- Nút **Mic** = **toggle**: `micOn` — idle (`Mic`) / đang nghe (`Mic ●`); tắt session: bấm Mic lại **hoặc** lệnh thoại (`VOICE_END_SESSION_PHRASES`).
- **Đóng chat (không tắt mic):** `đóng chat`, `thu gọn chat`, `ẩn chat`, … (`VOICE_CLOSE_CHAT_PHRASES`).
- **Kết thúc session:** `tắt mic`, `tắt micro`, `bye bye`, `bye` (`VOICE_END_SESSION_PHRASES`) → tắt mic **và** đóng chat.
- Hiển thị **bản ghi nhận dạng giọng** trong ô nhập khi đang nghe; user sửa được nếu STT sai trước khi im lặng đủ 2s.
- **Tự gửi từng câu:** `VOICE_SILENCE_MS = 2000` — reset timer mỗi lần STT cập nhật; hết 2s → gửi; **xóa ô nhập**; **khởi động lại session STT** (~80ms) sau gửi để Chrome không ghép transcript câu trước.
- **Session liên tục:** `micSessionRef` giữ true; `onend`/`onerror` → `spawnRecognition()` với **epoch** (bỏ event cũ), `abort()` session trước; **watchdog** 5s — nếu >12s không `onresult` → restart.
- Chặn `onresult` cũ sau gửi (`ignoreResultsUntil` ~800ms).
- Gợi ý header chat: *Nói "alo" hoặc "bật mic" để mở chat + bật mic · "đóng chat" · "tắt mic" / "bye bye" để thoát.*
- Fallback: trình duyệt không hỗ trợ STT → ẩn micro, chỉ dùng chat.
- Ngôn ngữ: **tiếng Việt bắt buộc** (§2.5); STT Web Speech API `vi-VN`.

### 9.3 Layout responsive

**Desktop (≥ 1024px):**

```text
┌──────────────────────────────────────────────────────────────┐
│ [ZP] OpsOne  🟡 Hệ thống   [Dashboard][Sự cố][Bảo trì][Cấu hình] │
├──────────────────────────────────────────────────────────────┤
│  Dashboard — bảng routing (group SKU) + incidents            │
│                                                              │
│                                    ┌─────────────────────┐   │
│                                    │ Chat OpsOne      [−]│   │
│                                    │ [feed]              │   │
│                                    │ [input] [mic][Gửi]  │   │
│                                    └─────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

**Tablet (768–1023px):** Tab menu xuống dòng thứ hai; chat widget thu nhỏ `min(360px, 100vw-2rem)`.

**Mobile (< 768px):**

```text
┌─────────────────────┐
│ 🟡 Hệ thống         │  ← icon + nhãn global
│ Header + menu ≡     │
├─────────────────────┤
│  Tab: Feed | Chat | ⚙ Cấu hình │
├─────────────────────┤
│ 🟡 ZING — SKU 20k   │  ← icon per product card
│ 🟢 TOPUP_VINA       │
├─────────────────────┤
│  [Chat messages]    │
│  [input] [🎤] [➤]   │
└─────────────────────┘
```

**Checklist responsive:**

- Mobile-first CSS; breakpoint: 768px, 1024px.
- Touch-friendly: nút micro / send ≥ 44×44px.
- Bảng routing trên mobile: **không scroll ngang** — cùng `table-layout: fixed` + truncate; provider % có thể xuống dòng trong ô nếu cần.
- Input chat không bị che bởi bàn phím ảo (iOS/Android).
- Test Chrome mobile + Safari iOS + desktop.

### 9.4 Luồng tích hợp Agent

```text
                    ┌─────────────────┐
  Scheduler ───────►│     OpsOne       │──► Output ──► UI Feed
  (đọc config UI)   │  (Agent core)    │
                    └────────┬────────┘
                             │ đọc agent_config mỗi chu kỳ
  UI Cấu hình ───────────────┤ (scheduler interval, mock)
  Dashboard (Auto dịch vụ/SKU) ──┤ (`ResolveEffectiveScopeAuto`, `auto_action`, `window_*`)
  User chat/voice ───────────┘
```


| Endpoint                            | Mục đích                                               |
| ----------------------------------- | ------------------------------------------------------ |
| `GET /config`                       | Đọc cấu hình scheduler + mock                          |
| `PUT /config`                       | Admin lưu cấu hình (audit log bắt buộc)                |
| `PUT /scopes/{product}/auto`       | Lưu chế độ BT / Routing **cấp dịch vụ** (`sku_code=""`) — áp dụng mọi SKU; nếu `ShouldAutoApplyScope` → `CancelPendingRoutingPlansForProduct` (§9.5.2) |
| `PUT /scopes/{product}/{sku}/auto`  | Lưu chế độ BT / Routing **cấp SKU** (fallback khi chưa cấu hình dịch vụ) (§9.5.2)               |
| `POST /scopes/{product}/{sku}/routing/approve`   | Duyệt đề xuất routing synthetic (§2.3)        |
| `POST /scopes/{product}/{sku}/routing/apply`     | Cập nhật routing thủ công tùy chỉnh (`manual_temp`) |
| `POST /scopes/{product}/{sku}/routing/restore-baseline` | Trả routing về `baseline_pct` — **Mở lại** provider (`manual_baseline`) |
| `POST /scopes/{product}/{sku}/maintenance/reopen-service` | **Mở lại dịch vụ** — hủy BT + baseline + grace (atomic) |
| `POST /scopes/{product}/{sku}/routing/reject`    | Từ chối đề xuất routing synthetic             |
| `POST /scopes/{product}/{sku}/maintenance/approve` | Duyệt đề xuất bảo trì synthetic           |
| `POST /scopes/{product}/{sku}/maintenance/reject`  | Từ chối đề xuất bảo trì synthetic         |
| `POST /scopes/{product}/{sku}/maintenance/extend`  | Gia hạn BT active — **400** `no_change` nếu thời gian không đổi |
| `GET /health-status`                | Trạng thái global + per product (`product_label`, icon, summary) |
| `GET /dashboard/overview`           | Bảng routing + bảo trì + pending plan (refresh `proposed_pct` trên poll) + `provider_metrics` + `under_maintenance` §9.0 |
| `GET /incidents?page=&page_size=`   | Feed incident phân trang (bảng §9.0)                     |
| `GET /incidents/{incident_id}`      | Chi tiết sự cố                                         |
| `GET /routing-plans/latest`         | (Legacy list) — UI Dashboard dùng `overview` + dedupe  |
| `POST /chat`                        | Text hoặc voice (STT client) — body `input_source`, `stt_raw` §7.6.5.5 |
| `POST /routing-plans/{id}/approve`  | Admin approve → UpdateRouting                          |
| `POST /routing-plans/{id}/reject`   | Từ chối đề xuất                                        |
| `POST /recommendations/{id}/approve`| Duyệt đề xuất bảo trì (Agent)                          |
| `POST /recommendations/{id}/reject` | Từ chối đề xuất bảo trì (Agent)                        |
| `GET /maintenance`                  | Cửa sổ bảo trì (starts_at / ends_at)                   |
| `POST /maintenance/{id}/approve`    | Duyệt bảo trì — admin có thể sửa thời gian             |
| `POST /maintenance/{id}/cancel`     | Hủy bảo trì                                            |
| `GET /notifications`                | Lịch sử email đã gửi §8.9                              |
| `POST /notifications/test`          | Gửi mail thử (admin)                                   |
| `GET /escalation-chat`              | Cấu hình app/nhóm/tag provider                         |
| `POST /voice`                       | (Tuỳ chọn) Audio → server STT → `/chat`                |


**Câu hỏi / lệnh UI hỗ trợ:**

- Giải thích incident `#id`
- Trạng thái product / provider / SKU
- So sánh metric với chu kỳ trước
- Tóm tắt Routing Plan đề xuất
- (Tuỳ chọn) Approve / reject routing (khi chưa auto)

### 9.5 Bảng cấu hình Agent (UI Settings)

Màn **Cấu hình** trên UI (admin; ops có thể **xem**). Lưu `PUT /config` → `agent_settings` — scheduler và Agent đọc **mỗi chu kỳ**.

**Layout compact (`Settings.tsx` + `index.css`):**

- Một card `.settings-card` — 3 nhóm `.settings-group`: **Scheduler** · **Bảo trì** · **Mock data** (phân cách `border-bottom`).
- Trường thường: hàng ngang `.settings-row` — label trái · control phải; checkbox cùng hàng label.
- **Kịch bản mock:** `.settings-row--select` — label trên · `<select>` **full-width** card (tránh cắt nhãn dài §4.5.3).
- Input số `.settings-input--num` — rộng cố định ~4.25rem; trang `max-width: 40rem`.
- Dòng dẫn ngắn: *Ngưỡng & chế độ Auto → Dashboard*; nút **Lưu cấu hình** dưới card.

#### 9.5.1 Scheduler — chu kỳ phân tích


| Trường UI            | Kiểu      | Mặc định  | Mô tả                                                             |
| -------------------- | --------- | --------- | ----------------------------------------------------------------- |
| **Bật scheduler**    | Toggle    | Bật       | Tắt = không chạy phân tích định kỳ (vẫn chat/voice + chạy tay)    |
| **Chu kỳ phân tích** | Số (phút) | `5`       | Khoảng cách giữa các lần Agent phân tích (VD: 3, 5, 10, 15 phút)  |
| **Mock data**        | Toggle    | Bật (dev) | Bật Mock Data Generator §4.5                                      |
| **Chu kỳ mock**      | Số (phút) | `1`       | Tần suất sinh data giả (khuyến nghị 1 phút)                       |
| **Kịch bản mock**    | Select    | `normal`  | Giá trị API giữ nguyên; **nhãn UI tiếng Việt** (§4.5.3): *Bình thường* · *ESALE suy giảm* · *Lỗi cục bộ SKU* · *Đột biến lỗi* |
| **Nguồn dữ liệu**    | Radio     | `mock`    | `mock` (chạy thử) | `production`                                  |


```text
Chu kỳ phân tích:  [  5  ] phút     (min 1 — max 60)
Scheduler:         [●] Bật  [ ] Tắt

Bảo trì
  Thời lượng mặc định (phút)     [ 60 ]

Mock data:         [●] Bật  [ ] Tắt
Kịch bản:
  [ Bình thường — nhiễu nhẹ, không sự cố lớn                    ▼ ]
  (select full-width — nhãn dài không bị cắt)
Nguồn: mock
```

- Đổi chu kỳ → scheduler **áp dụng từ chu kỳ tiếp theo** (không chồng 2 job).
- Hiển thị **lần chạy tiếp theo** / lần chạy cuối trên UI.

#### 9.5.2 Chế độ BT / Routing — cấp dịch vụ + SKU (Dashboard)

Cấu hình lưu `routing_scope_state.auto_action`, `window_start`, `window_end` — hai tầng:

| Tầng | Key DB | API lưu | UI |
| --- | --- | --- | --- |
| **Dịch vụ** | `(product_code, sku_code="")` | `PUT /scopes/{product}/auto` | `ScopeAutoEditor` cột **Dịch vụ** (sku-mode, §9.0.7) |
| **SKU** | `(product_code, sku_code)` | `PUT /scopes/{product}/{sku}/auto` | `ScopeAutoEditor` cột **Chế độ BT / Routing** |
| **Topup provider** | `(product_code, sku_code="")` | `PUT /scopes/{product}/auto` | Một dòng — editor ở cột routing (không lặp cột Dịch vụ) |

**Ưu tiên hiệu lực** (`ResolveEffectiveScopeAuto` — `internal/store/scope_auto.go`; poll `dashboard.go`; Agent `reasoning.go`):

```text
IF tồn tại row routing_scope_state (product, sku_code="")
  → effective = cấu hình dịch vụ (áp dụng mọi SKU thuộc product)
ELSE
  → effective = cấu hình (product, sku_code) hoặc mặc định recommend_only
```

**Response `GET /dashboard/overview` (mỗi row SKU):**

| Field | Ý nghĩa |
| --- | --- |
| `auto_action`, `window_*` | Cấu hình **hiệu lực** (`ResolveEffectiveScopeAuto` → `ShouldAutoApplyScope`, ẩn/hiện đề xuất) |
| `product_auto_action`, `product_window_*` | Cấu hình **gốc** dịch vụ — `ScopeAutoEditor` cột Dịch vụ (`initial` từ `product_auto_action`) |
| `scope_auto_action`, `scope_window_*` | Cấu hình **gốc** SKU — `ScopeAutoEditor` cột điều khiển SKU (`initial` từ `scope_auto_action`, fallback `auto_action` nếu thiếu) |

**Backend assemble (`internal/api/dashboard.go`):** load `ListScopeAutoConfig` → map key `ScopeAutoMapKey(product, sku)` (`internal/store/scope_auto.go`). Mỗi row overview:

1. `product_auto_action` ← lookup `(product, sku_code="")`; không có row → `"recommend_only"`.
2. `scope_auto_action` ← lookup `(product, sku_code)`.
3. `auto_action` ← `ResolveEffectiveScopeAuto` (dịch vụ **đã lưu** — kể cả `recommend_only` — ghi đè mọi SKU).

> **Triển khai:** không dùng `MaintenanceScopeKey` / `maintKey()` cho auto — chỉ `ScopeAutoMapKey`. `PUT` và `GET` overview phải cùng phiên bản API (restart `cmd/api` sau đổi `dashboard.go`).


| Giá trị `auto_action` | Nhãn UI (`ScopeAutoEditor`) | Hành vi Agent + Dashboard |
| ----------------------- | --------------------------- | ------------------------- |
| `recommend_only` (mặc định) | Chỉ đề xuất | Sinh plan / đề xuất bảo trì; **không** `UpdateRouting` / `SetMaintenance`; UI hiện hàng Duyệt/Từ chối |
| `auto` | Tự động | Tự `UpdateRouting` hoặc `SetMaintenance` (`trigger_type=auto`); **ẩn** `pending_plan` / `pending_maintenance` trên Dashboard |
| `time_window` | **Tự động theo khung giờ** | Trong `[window_start, window_end)` → giống `auto`; **ngoài giờ** → giống `recommend_only` |


**UI `ScopeAutoEditor` (compact):**

- Mặc định: hai dòng canh phải — `Chế độ BT / Routing` + badge giá trị (vd. **Tự động**); nút **⋯** mở form.
- Form chỉnh: dropdown + khung giờ (nếu `time_window`) + **Lưu** / **Hủy**; label canh phải.
- **Lưu:** `PUT /scopes/.../auto` → `patchOverviewCache` (React Query `dashboard-overview`) cập nhật `product_auto_action` / `scope_auto_action` + `auto_action` khi lưu cấp dịch vụ → `refetchQueries` đồng bộ server.
- **Hai nút Lưu khác nhau:** hàng *Ngưỡng cảnh báo* = ngưỡng + email; `ScopeAutoEditor` = chế độ BT/routing (§9.5.3).

**UI khi chọn Tự động theo khung giờ:** hai input datetime (Từ – Đến), ví dụ `08:00` – `22:00`. Hỗ trợ khung qua nửa đêm (VD `22:00`–`06:00`).

**Logic `ShouldAutoApplyScope` (`internal/store/scope_auto.go`; frontend `utils/scopeAuto.ts`):**

```text
if auto_action == "auto"                    → UpdateRouting + SetMaintenance (nếu rules OK)
if auto_action == "recommend_only"          → chỉ đề xuất; không apply
if auto_action == "time_window"             → apply chỉ khi now ∈ [window_start, window_end)
```

**Dashboard API:** `GET /dashboard/overview` chỉ trả `pending_plan` / `pending_maintenance` khi `ShouldAutoApplyScope=false` cho scope đó.

**`PUT /scopes/{product}/auto`:** sau lưu, nếu `ShouldAutoApplyScope(saved, now)=true` → `CancelPendingRoutingPlansForProduct` (hủy plan chờ mọi SKU).

**`PUT /scopes/{product}/{sku}/auto`:** sau lưu, nếu `ShouldAutoApplyScope(saved, now)=true` → `CancelPendingRoutingPlansForScope` (xóa plan chờ cũ); poll overview tiếp theo auto apply.

**Agent (`reasoning.go`):** đọc `ResolveEffectiveScopeAuto(product, sku)` — routing / maintenance theo cấu hình hiệu lực; nếu `ShouldAutoApplyScope` và còn plan pending → hủy pending + auto apply; sau routing nếu `ShouldForceAutoMaintenance` → chain `SetMaintenance` cùng chu kỳ; maintenance — `SetMaintenance` (mặc định 60 phút) thay vì chỉ `InsertRecommendation`. Cùng bộ **force** như poll Dashboard.

**Force bypass gate chu kỳ** (`internal/agent/breach.go`; poll `scopeAutoApplyAllowed` + `autoApplyScopeFromSnapshot`):

| Hàm | Điều kiện | Hành động |
| --- | --- | --- |
| `ShouldForceAutoRouting` | `ActiveRoutingCount ≥ 2`, `SKURoutingDecision=routing` (còn provider khỏe) | Tự **shift routing** ngay |
| `ShouldForceAutoMaintenanceAllProviders` | `ActiveRoutingCount ≥ 2`, `SKURoutingDecision=maintenance` (tất cả active vi phạm) | Tự **bảo trì SKU** ngay |
| `ShouldForceAutoMaintenance` | `ActiveRoutingCount == 1`, provider đó vi phạm | Tự **bảo trì** ngay (hard cut đã hội tụ, VD ESALE 0% IMEDIA 100% vẫn đỏ) |

**Dashboard poll auto (`autoApplyScopeFromSnapshot`):** khi `ShouldAutoApplyScope=true` và `scopeAutoApplyAllowed` (mục force trên hoặc `ShouldAct`) → tự apply mỗi `GET /dashboard/overview` (tối đa 2 pass: routing rồi bảo trì); routing no-op (`proposed == current`) → fallback bảo trì.

**Ví dụ `time_window`:** khung `11/06 08:00` → `12/06 18:00`, `now=11/06 00:30` → **ngoài khung** → hiện hàng *Kế hoạch routing* + Duyệt/Từ chối; `now=11/06 10:00` → **trong khung** → không trả `pending_plan`, poll tự shift (VD 100% provider khỏe).

> **Đã bỏ:** `agent_settings.auto_routing_mode`, bảng `auto_routing_time_windows`, chế độ global `always` trên trang Cấu hình.

#### 9.5.3 Ngưỡng cảnh báo theo dịch vụ

**Ngưỡng** cấu hình trên **Dashboard → bảng routing** (`ProductThresholdEditor` **đầu** mỗi nhóm `product_code`). Trang **Cấu hình** chỉ còn scheduler + mock — có dòng dẫn link sang Dashboard.

**Layout `ProductThresholdEditor`:** **một hàng `<tr class="overview-table__threshold-row">`** đầu mỗi nhóm — căn giữa dọc (`vertical-align: middle`) toàn hàng. Cấu trúc ô — §9.0.7 bảng hàng Ngưỡng.

**5 cụm ngưỡng** (`overview-threshold-field`) — mỗi cụm: label · toán tử (`≤` / `≥`) · input; `flex: 0 0 auto` (không nén chồng chữ); giãn cách đều; đường phân cách nhẹ giữa nhóm **%** và nhóm **GD** (cụm thứ 4). Nhãn UI:

| Nhãn UI | API field | Mặc định | Toán tử |
| ------- | --------- | -------- | ------- |
| **% Success** | `success_rate_min_pct` | 80 | `≤` |
| **% Pending** | `pending_rate_max_pct` | 15 | `≥` |
| **% Fail** | `fail_rate_max_pct` | 10 | `≥` |
| **Pending** (GD) | `pending_txn_count_max` | 5 | `≥` |
| **Fail** (GD) | `fail_txn_count_max` | 50 | `≥` |

**Email** + **Lưu** — cột Chế độ BT / Routing (hàng ngưỡng). `PUT /products/{code}/thresholds`. **Chỉ** ngưỡng + bật email — **không** lưu `auto_action` (chế độ → `ScopeAutoEditor` §9.5.2).

| Trường UI (legacy / API)  | API field                 | Mặc định hệ thống | Mô tả |
| ------------------------- | ------------------------- | ----------------- | ----- |
| Success tối thiểu (%)     | `success_rate_min_pct`    | 80                | Dưới → cảnh báo |
| Pending tối đa (%)        | `pending_rate_max_pct`    | 15                | Trên → cảnh báo |
| Fail tối đa (%)           | `fail_rate_max_pct`       | 10                | Trên → cảnh báo |
| **Số GD fail tối đa**     | `fail_txn_count_max`      | **50**            | Trong cửa sổ 15 phút |
| **Số GD pending tối đa**  | `pending_txn_count_max`   | **5**             | Trong cửa sổ 15 phút |
| Bật cảnh báo (email)      | `alert_email_enabled`     | Bật               | Khi vượt ngưỡng + bật → gửi email §8.9 |

**Logic cảnh báo (§7.4):** OR tất cả điều kiện % và số GD → `breached=true` → `should_alert_mode=true` (UI 🔴/🟡). Email chỉ khi `alert_email_enabled`.

```text
Mặc định agent_settings (§13.4):
  fail_txn_count_max = 50   pending_txn_count_max = 5
  success 80% / pending 15% / fail 10%
```

- UI hiển thị gợi ý: vượt bất kỳ ngưỡng → bật chế độ cảnh báo.
- Hard cut routing (§8.6.3): vượt ngưỡng fail/pending **% hoặc số GD** → đề xuất cắt provider về 0% (**sau khi** đủ `consecutive_cycles_required`).
- **Chu kỳ vi phạm liên tiếp** (`consecutive_cycles_required`, mặc định 2): chu kỳ 1 → 🟡 monitor; chu kỳ 2+ → đề xuất / auto theo §9.5.2; hết vi phạm → 🟢.

#### 9.5.4.1 Cấu hình tỉ lệ routing (Admin PUT — §8.6.5)

**Tab Cấu hình → Tỉ lệ routing** — Admin chọn product (+ SKU nếu `routing_mode=sku`), sửa bảng provider × %.

**Layout form:**

```text
Sản phẩm: [ TOPUP_VINA ▼ ]     Phạm vi: provider (topup)

┌──────────┬───────────┬───────────┐
│ Provider │ Baseline  │ Đang chạy │  ← cột Baseline read-only nếu pending_restore
├──────────┼───────────┼───────────┤
│ ESALE    │   70 %    │ [ 50 ] %  │
│ IMEDIA   │   20 %    │ [ 30 ] %  │
│ SHOPPAY  │   10 %    │ [ 20 ] %  │
└──────────┴───────────┴───────────┘
Tổng đang chạy: 100 %  ✓

☐ Đặt làm baseline mới (Business)
   → set_as_baseline = true
   → Tỉ lệ mới là mốc chính thức; recovery sau này về đây

☑ Chỉnh tạm thời — tự trả baseline khi hệ thống ổn định   (mặc định khi unchecked baseline)
   → set_as_baseline = false
   → Giữ baseline cũ; badge "Chỉnh tạm" trên Dashboard

[Lý do (tuỳ chọn): ________________________________]

[ Lưu ]   [ Huỷ ]
```

**Quy tắc UI:**

| Trạng thái | Hiển thị |
|------------|----------|
| `pending_restore=1` | Badge vàng *"Chỉnh tạm — chờ khôi phục"*; cột Baseline = giá trị Business |
| Checkbox **baseline mới** bật | Ẩn badge; sau Lưu baseline = traffic |
| Checkbox **baseline mới** tắt | Helper: *"Hệ thống sẽ trả về baseline (ESALE 70% \| …) khi metric ổn định"* |
| Hai checkbox **loại trừ** nhau | Chỉ một mode: `set_as_baseline` true **hoặc** false |

**Validate client:**

- `Σ traffic_pct = 100` per scope/SKU trước khi gọi PUT.
- Chỉ role **Admin** thấy nút Lưu; Ops read-only + tooltip *"Liên hệ admin"*.

**API:** `GET/PUT /products/{code}/routing` — xem §8.6.5.

#### 9.5.5 Bảo trì — thời lượng mặc định

**UI (`Settings.tsx` — nhóm *Bảo trì* trong card §9.5):** `PUT /config` field `maintenance_default_duration_min` → `agent_settings.maintenance_default_duration_min`.


| Trường UI (label ngắn)          | API field                         | Kiểu          | Mặc định | Mô tả                                                                      |
| ------------------------------- | --------------------------------- | ------------- | -------- | -------------------------------------------------------------------------- |
| **Thời lượng mặc định (phút)**  | `maintenance_default_duration_min` | Number (phút) | `60`     | Dashboard: `defaultMaintenanceWindow(duration)` khi mở form BT; Agent + poll overview auto: `ends_at = starts_at + duration` |
| **Tự động bảo trì**             | `maintenance_auto_enabled`        | Toggle        | Tắt      | *(Chưa expose UI)* — bật → khi `active_provider_count == 1` có thể `SetMaintenance` auto |


```text
Bảo trì
  Thời lượng mặc định (phút)     [ 60 ]    ← input số hẹp, căn phải
```

**Backend:** `NormalizeMaintenanceDefaultDurationMin` (`internal/store/settings.go`); `maintenanceDefaultDurationMin` trong API (`maintenance_helpers.go`, `reasoning.go`, `dashboard_suggest.go`).

**Frontend:** `normalizeConfig()` — nếu API thiếu/0 → hiển thị **60**; `useMaintenanceDefaultDurationMin` (React Query `['config']`) — `ServiceMaintenanceButton`, `ServiceOverviewTable`.

- Admin approve maintenance có thể **sửa** `starts_at` / `ends_at` trước khi lưu.
- UI hiển thị countdown: *Còn 42 phút* đến `ends_at`.

#### 9.5.6 Thông báo email & leo thang chat provider

**Tab Cấu hình → Thông báo:**


| Trường UI          | Kiểu             | Mặc định               | Mô tả                                                                          |
| ------------------ | ---------------- | ---------------------- | ------------------------------------------------------------------------------ |
| **Bật email**      | Toggle           | Bật                    | `notification_enabled`                                                         |
| **Người nhận**     | Tags input email | `ops-team@company.com` | JSON array — team vận hành                                                     |
| **Gửi khi 🔴**     | Toggle           | Bật                    | `notification_on_red_only` — nếu tắt, gửi cả vàng khi vượt ngưỡng pending/fail |
| **Ngưỡng pending** | %                | `15`                   | `notification_pending_threshold`                                               |
| **Ngưỡng fail**    | %                | `10`                   | `notification_fail_threshold`                                                  |
| **Cooldown**       | phút             | `60`                   | Tránh gửi trùng                                                                |


> **Lưu ý:** Ngưỡng **kích hoạt hành động + mail** lấy từ **§9.5.3 per product**. Các trường pending/fail ở đây là **fallback** khi product chưa cấu hình.

**Bảng leo thang chat theo provider** (editable):


| Provider | App chat        | Tên nhóm                  | Tag mention       |
| -------- | --------------- | ------------------------- | ----------------- |
| ESALE    | Microsoft Teams | `[OpsOne] ESALE Support`  | `@esale-oncall`   |
| IMEDIA   | Microsoft Teams | `[OpsOne] IMEDIA Support` | `@imedia-ops`     |
| SHOPPAY  | Zalo            | `OpsOne x SHOPPAY`        | `@shopay-support` |


- `PUT /escalation-chat` — admin sửa từng provider.
- Nút **Gửi mail thử** → `POST /notifications/test`.
- Trang hoặc tab **Lịch sử thông báo** — `GET /notifications`.

#### 9.5.7 Schema `agent_config` (backend)

`GET/PUT /config` — **không còn** `auto_routing_mode`. Auto routing cấu hình cấp **dịch vụ** (`PUT /scopes/{product}/auto`) và/hoặc **SKU** (`PUT /scopes/.../{sku}/auto`); hiệu lực theo §9.5.2.

`GET/PUT /config` (triển khai phẳng — không nested):

```json
{
  "scheduler_enabled": true,
  "scheduler_interval_min": 5,
  "data_source": "mock",
  "mock_enabled": true,
  "mock_interval_min": 1,
  "mock_scenario": "normal",
  "maintenance_default_duration_min": 60,
  "agent_locale": "vi-VN"
}
```

Schema mở rộng (chưa expose hết trên UI):

```json
{
  "scheduler_enabled": true,
  "scheduler_interval_min": 5,
  "data_source": "mock",
  "mock_data": {
    "enabled": true,
    "interval_minutes": 1,
    "scenario": "esale_degrading",
    "retention_hours": 24
  },
  "maintenance": {
    "default_duration_minutes": 60,
    "auto_enabled": false
  },
  "notification": {
    "enabled": true,
    "recipients": ["ops-team@company.com", "noc@company.com"],
    "on_red_only": true,
    "cooldown_minutes": 60
  },
  "alert_threshold_defaults": {
    "success_rate_min_pct": 80,
    "pending_rate_max_pct": 15,
    "fail_rate_max_pct": 10,
    "fail_txn_count_max": 100,
    "error_event_count_max": 50,
    "pending_txn_count_max": null,
    "metrics_window_min": 15,
    "consecutive_cycles_required": 2
  },
  "product_thresholds": [
    {
      "product_code": "TOPUP_VINA",
      "enabled": true,
      "success_rate_min_pct": 80,
      "pending_rate_max_pct": 15,
      "fail_rate_max_pct": 10,
      "fail_txn_count_max": 120,
      "error_event_count_max": 60,
      "metrics_window_min": 15,
      "consecutive_cycles_required": 2,
      "alert_email_enabled": true
    }
  ],
  "escalation_chat": [
    {
      "provider_code": "ESALE",
      "chat_app_name": "Microsoft Teams",
      "chat_group_name": "[OpsOne] ESALE Support",
      "mention_tags": "@esale-oncall"
    }
  ],
  "updated_at": "2026-06-04T10:00:00+07:00",
  "updated_by": "admin@ops",
  "locale": "vi-VN"
}
```

**Response mẫu `PUT /scopes/GARENA/10000/auto`:**

```json
{
  "product_code": "GARENA",
  "sku_code": "10000",
  "auto_action": "time_window",
  "window_start": "08:00",
  "window_end": "22:00"
}
```

#### 9.5.8 Checklist UI Cấu hình

- [x] Một card compact — scheduler + bảo trì (§9.5.5) + mock; input số hẹp; **select kịch bản mock full-width** (`.settings-row--select`).
- Chỉ **admin** được `PUT /config`; ops read-only.
- Validate: `scheduler_interval_min` 1–60; `maintenance_default_duration_min` 1–255 (API **400** nếu ngoài khoảng); mock interval 1–5 phút.
- Integration test: `TestConfigPutMaintenanceDefaultDuration`.
- Audit log mỗi lần đổi config (ai, khi nào, before/after).
- Ngưỡng dịch vụ — inline Dashboard §9.5.3 (không còn tab riêng trên Cấu hình).
- Badge: *"Đang dùng MOCK DATA"* / *"Auto routing: …"* trên header UI.

### 9.6 (đã bỏ) ~~UI Lịch sử thay đổi & Rollback~~

> **Đã loại bỏ** trang `/changes`, API `GET/POST /agent-changes/*` và rollback. Khôi phục routing: **Mở lại provider** / **Mở lại dịch vụ** trên Dashboard (§8.7).

### 9.7 Checklist triển khai UI

- [x] Web app responsive (PC + điện thoại); Vite + React 18 + TypeScript.
- [x] Dashboard §9.0 — `ServiceOverviewTable` + `IncidentsTable` (**phân trang** 10/trang); refresh **60s** + SSE.
- [x] Top-nav tabs (Dashboard / Sự cố / Bảo trì / Cấu hình); `ChatWidget` dock 4 góc (`useChatDock`).
- [x] Dashboard **tab loại dịch vụ** (Thẻ / Topup / Data) + icon theo **worst per-SKU** + plan chờ 🔴.
- [x] Bảng routing **group SKU** theo `product_code` (`rowspan`); **sort SKU số tăng dần**.
- [x] **Cột provider** — 6 chỉ số/provider (`ProviderMetricCell`); active đỏ/xanh; `routing_pct=0` → muted + metric 0 + **Mở lại** → hàng *Mở lại provider*; Lưu baseline → `restore-baseline` (`manual_baseline`); **BT active** → disabled + metric 0 (`under_maintenance`).
- [x] **Health per provider (live)** — metric đỏ khi vượt ngưỡng; cột TT 🟡/🔴 theo `consecutive_cycles_required`.
- [x] **Auto đề xuất routing/bảo trì (live)** — `scopeSuggestionFromSnapshot` khi `ShouldAct`; **ẩn** khi `ShouldAutoApplyScope=true`; refresh plan DB trên poll (`refreshPendingRoutingFromSnapshot`).
- [x] **Kế hoạch routing — sync draft** — `ServiceOverviewTable` cập nhật `draftRouting` khi API trả `proposed_pct` mới (poll 60s).
- [x] **Auto per dịch vụ + SKU** — `ScopeAutoEditor` cột Dịch vụ (`PUT /scopes/{product}/auto`) + cột SKU; `ResolveEffectiveScopeAuto` (dịch vụ trước); mặc định `recommend_only`.
- [x] **1 sự cố mở / SKU** — Agent không `InsertIncident` trùng khi còn `status=open`.
- [x] **Error event snapshot** — `SumErrorEventsAtRecordedAt` cùng `recorded_at` GetMetrics (§2.3.2); Agent collector + threshold dùng chung quy tắc.
- [x] Header/logo **`useOverallHealth`** — 🔴 khi bất kỳ SKU/tab đỏ hoặc plan chờ.
- [x] Cột TT: **🟡** vi phạm lần 1; **🔴** đủ consecutive hoặc plan chờ; **🟢** khi metric OK.
- [x] **Kế hoạch routing** / **Đề xuất bảo trì** — hàng con dưới SKU; validate **0–100%**, tổng 100%; Duyệt/Từ chối căn phải (plan DB + synthetic qua `/scopes/...`); **Từ chối** hủy plan pending — không treo UI.
- [x] **Nhãn BT active** — 4 dòng dưới tên SKU (`SkuMaintenanceTimeLabel`, §9.0.1); cột Bảo trì chỉ nút hành động.
- [x] **`RedSkuScrollNav`** — ▲ 🔴 1/N ▼ nhảy tới SKU đỏ + flash highlight.
- [x] **Khối SKU** — viền dưới đậm giữa các scope; không viền giữa SKU và hàng con (§9.0.2).
- [x] **Hàng Đề xuất bảo trì** — layout cột 1–2 / 3–4 / 5→cuối; **Từ/Đến** compact; Duyệt/Từ chối cùng hàng (§9.0.3).
- [x] **Toast scope** — `{product_label} · {SKU}` qua `scopeDisplayLabel` (§9.0.4).
- [x] **Gia hạn BT** — **Từ/Đến** + Lưu; lỗi *Thời gian bảo trì không thay đổi* UI + API `no_change` (§9.0.5).
- [x] **Bảo trì dịch vụ thủ công** — `ServiceMaintenanceButton` inline Từ/Đến + Lưu; mặc định N phút (§9.5.5); batch cột Dịch vụ (§9.0.7).
- [x] **BT active cấp dịch vụ** — `ProductMaintenanceActions`: Mở lại / Gia hạn batch SKU **chỉ khi mọi SKU đang BT**; nút ngang grid 2 cột (§9.0.7).
- [x] **Cột Dịch vụ** — hành động batch BT/routing (tên dịch vụ trên hàng Ngưỡng); layout trái/phải; `min-width: 18rem`.
- [x] **Hàng Ngưỡng** — `product_label` · *Ngưỡng cảnh báo* · 5 cụm (%/GD) thẳng cột Provider · Email + Lưu (§9.5.3).
- [x] **Cột điều khiển SKU** — gộp Bảo trì + Chế độ BT/Routing (`scope-controls-cell`, `colspan=2`).
- [x] `GET /dashboard/overview` trả `service_type` mỗi row.
- [x] Duyệt/từ chối inline + validate tổng 100%; banner phản hồi; % làm tròn 1 chữ số.
- [x] **Đóng sự cố theo hành động** — duyệt → `resolved` + audit; từ chối → `acknowledged`; cột **Người xử lý** / **Thời gian xử lý**.
- [x] **Recovery timeline** — +1 chu kỳ agent sau duyệt/auto → 🟡; +1 chu kỳ nữa nếu metric OK → 🟢 (`recovery_apply_cycle_id`).
- [x] `ProductThresholdEditor` **đầu nhóm** — 5 input ngưỡng; **Lưu** (không còn dropdown Auto global).
- [x] Cột **Chế độ BT / Routing** — `ScopeAutoEditor` per SKU + cột Dịch vụ (sku-mode); compact + ⋯ (Chỉ đề xuất / Tự động / Tự động theo khung giờ); overview tách `product_auto_action` / `scope_auto_action` / `auto_action` hiệu lực §9.5.2; Lưu riêng (không dùng Lưu hàng Ngưỡng).
- [x] Bảng metric — scroll ngang nhẹ khi cần; `table-layout: fixed`, truncate + tooltip.
- [x] `GET /incidents?page=&page_size=` — phân trang full-width; **không** route `/incidents/:id` trên UI.
- [x] `ChatWidget` + `useChatDock` + `useVoiceInput` + `useOpsOneWake`; panel lớn (~800px); wake **alo**; voice đóng chat / tắt mic / bye bye; onboarding + avatar §9.2.1; alias + tra metric/BT direct §7.6.5
- [x] Dashboard `health_summary` hero — nhãn product tiếng Việt (`catalog/display.go`, `FormatCycleHealthSummary`)
- [x] `/settings` — card compact §9.5: scheduler + **thời lượng BT mặc định** (§9.5.5) + mock (`normalizeConfig` fallback 60); select kịch bản **full-width**.
- [x] Dev: `web/dev.ps1`, `VITE_DEV_AUTH_BYPASS`, proxy Vite → API.
- [ ] MSAL O365 production (`VITE_AAD_*`); §2.6 JWT middleware đầy đủ.
- [x] Dashboard — `PUT /products/{code}/thresholds` qua `ProductThresholdEditor`; ngưỡng % + số GD fail (mặc định **50**) / pending (mặc định **5**); bật cảnh báo email.
- [ ] Khung chat render markdown; PWA (tuỳ chọn).

---

# Part V — Verification

## 10. Kịch bản end-to-end (dùng khi test)

### 10.1 Kịch bản A — Topup tiền (multi-provider, routing theo provider)

**Tình huống:** TOPUP_VINA — ESALE suy giảm toàn product, IMEDIA/SHOPPAY ổn.

1. Scheduler chạy 10:10.
2. `service_type=topup` → không gọi GetSkus.
3. Tools: metrics ESALE xấu; top error -3004; GetRouting `scope=provider` ESALE 70%; revenue cao.
4. History: 2 chu kỳ success giảm (product × provider).
5. Reasoning: Rule 1 + 5 + 6 → chuyển traffic theo provider.
6. Output: Incident + Recommendation + **Routing Plan** (provider).
7. (Tuỳ chọn) UpdateRouting `{ "scope": "provider", "routing": { "ESALE": 0, "IMEDIA": 67, "SHOPPAY": 33 } }` (hard cut khi fail/pending/txn breach).
8. Ghi history dòng 10:10 (`sku` = null).

**Kết quả kỳ vọng:** Expected success ~97%.

### 10.2 Kịch bản B — ZING chỉ 1 provider active (bảo trì có thời hạn)

**Tình huống:** Thẻ **ZING** chỉ có **ESALE active** — hoặc catalog chỉ còn 1 dòng `product_providers`, hoặc IMEDIA/SHOPPAY bị xóa/inactive. Metric ESALE rất xấu 2 chu kỳ (~10 phút).

**Chuẩn bị (trước khi chạy):** `mysql ... < db/scenario-b-zing-esale-only.sql` hoặc:

```sql
-- ZING chỉ ESALE active — dùng cho demo kịch bản B
DELETE pp FROM product_providers pp
JOIN products p ON p.id = pp.product_id
JOIN providers pr ON pr.id = pp.provider_id
WHERE p.product_code = 'ZING' AND pr.provider_code IN ('IMEDIA', 'SHOPPAY');

DELETE FROM routing_config
WHERE product_code = 'ZING' AND provider_code IN ('IMEDIA', 'SHOPPAY');
```

1. GetProviders(`ZING`) → `active_count = 1`, `active_providers = ["ESALE"]`.
2. Không sinh Kế hoạch routing (không có provider **active** dự phòng).
3. EvaluateThresholds → `suggested_action = "maintenance"`.
4. **10:15** — OpsOne output **Đề xuất bảo trì** (tiếng Việt):
  - `starts_at`: 10:15, `ends_at`: 11:15 (60 phút — `maintenance_default_duration_min`).
  - `status`: `pending_approve` (mode `recommend_only`).
5. Admin approve → `scheduled`/`active`; UI card ZING 🔧 *Bảo trì đến 11:15*.
6. **10:20** — Chu kỳ Agent: GetMaintenance → active → **không** tạo Incident trùng.
7. **11:15** — Tick worker → `completed`; metric đánh giá lại bình thường.

**Kết quả kỳ vọng:** Bảo trì ZING/ESALE có thời gian rõ ràng; ops biết khi nào mở lại dịch vụ. *(Sau demo B, restore seed đủ 3 provider active để chạy kịch bản C.)*

### 10.2b Kịch bản B′ — 2 provider catalog, 1 inactive, provider active còn lại xấu (bảo trì)

**Tình huống:** ZING vẫn có **2 dòng** `product_providers` (ESALE + IMEDIA) nhưng **IMEDIA inactive** (`enabled = 0`). ESALE (provider active duy nhất) metric xấu 2 chu kỳ.

**Chuẩn bị:**

```sql
-- ZING: IMEDIA inactive, ESALE active — edge case §7.4
UPDATE product_providers pp
JOIN products p ON p.id = pp.product_id
JOIN providers pr ON pr.id = pp.provider_id
SET pp.enabled = 0
WHERE p.product_code = 'ZING' AND pr.provider_code = 'IMEDIA';
```

1. GetProviders(`ZING`) → `active_count = 1`, `total_count = 2`, `inactive_providers = ["IMEDIA"]`.
2. EvaluateThresholds → `suggested_action = "maintenance"` — **KHÔNG** routing sang IMEDIA (inactive).
3. Output: Incident + Đề xuất bảo trì ESALE (giống kịch bản B).
4. Verify: **không** có Routing Plan; **không** gọi UpdateRouting.

**Kết quả kỳ vọng:** OpsOne phân biệt đúng **active** vs **catalog** — 2 provider trên giấy nhưng chỉ 1 chạy → bảo trì, không shift traffic sang provider đã tắt.

### 10.3 Kịch bản C — Thẻ ZING (routing theo SKU / mệnh giá)

**Tình huống:** SKU 10.000đ trên ESALE ~98% (ổn), nhưng **SKU 20.000đ** trên ESALE suy giảm.

1. `service_type=card` → GetSkus → `10000`, `20000`, `50000`, …
2. GetMetrics `sku=10000`, ESALE: success ~98% → OK.
3. GetMetrics `sku=20000`, ESALE: success 72%, fail 13% → Rule **7, 8**.
4. GetRouting `scope=sku`: ESALE 80% traffic mệnh giá 20k.
5. GetMetrics cùng SKU trên IMEDIA: success 98%.
6. Reasoning: chỉ routing **SKU 20000** — **không** UpdateRouting `scope=provider`.
7. Output: Incident (scope sku) + Routing Plan SKU 20k + Recommendation.
8. UpdateRouting `{ "scope": "sku", "sku": "20000", "routing": { "ESALE": 10, "IMEDIA": 90 } }`.
9. Ghi history với `sku=20000`.

**Kết quả kỳ vọng:** Success mệnh giá 20k ~96%; mệnh giá 10k không bị ảnh hưởng.

### 10.4 Kịch bản D — Topup data (routing theo SKU / gói)

**Tình huống:** DATA_VINA — gói `VNP20` trên ESALE ~98% (ổn), nhưng **gói `VNP50`** trên ESALE suy giảm.

1. `service_type=topup_data` → GetSkus → `VNP20`, `VNP50`, `V50K`, `V100K`
2. GetMetrics `sku=VNP20`, ESALE: success ~98% → OK.
3. GetMetrics `sku=VNP50`, ESALE: success 72%, fail 13% → Rule **7, 8**.
4. GetRouting `scope=sku`: ESALE 80% traffic gói VNP50.
5. GetMetrics cùng SKU trên IMEDIA: success 98%.
6. Reasoning: chỉ routing **SKU VNP50** — **không** UpdateRouting `scope=provider`.
7. Output: Incident (scope sku) + Routing Plan gói VNP50 + Recommendation.
8. UpdateRouting `{ "scope": "sku", "sku": "VNP50", "routing": { "ESALE": 10, "IMEDIA": 90 } }`.
9. Ghi history với `sku=VNP50`.

**Kết quả kỳ vọng:** Success gói VNP50 ~96%; gói VNP20 không bị ảnh hưởng.

### 10.5 Kịch bản E — UI chat & voice (responsive)

**Tình huống:** Ops đang di chuyển, mở UI trên **điện thoại** sau khi scheduler sinh Incident #20260604-001 (ZING SKU 20k).

1. UI feed hiển thị card Sự cố mức Cao — tap mở chi tiết.
2. **Chat:** gõ *"Tình hình topup mobi?"* → alias `TOPUP_MOBI` §7.6.5 → `get_routing` + `get_metrics` (cả 3 provider) → tóm tắt tiếng Việt. Hoặc *"Tại sao ESALE 20k fail tăng?"* → `get_metrics` + `get_top_errors`. **GD pending:** *"thẻ Mobifone 50.000 quay đơn có banding không"* → direct reply §7.6.5.2 (29 GD pending ESALE khớp cột Dashboard). **Bảo trì:** *"Thẻ Garena có đang bảo trì?"* hoặc *"Thẻ Mobifone 10.000"* (follow-up) → direct reply §7.6.5.1 khớp nhãn BT Dashboard.
3. **Voice:** bật Mic, nói *"Đề xuất routing cho mệnh giá hai mươi nghìn"* → im lặng 2s → tự gửi `/chat` (mic vẫn bật, ô nhập clear) → Agent trả Routing Plan; hỏi tiếp không cần bật Mic lại; nói *"tắt mic"* hoặc bấm Mic để tắt.
4. Trên **desktop** cùng tài khoản: Dashboard §9.0 — bảng routing (hàng con *Kế hoạch routing* dưới SKU) + bảng incidents + `ChatWidget` góc phải dưới (cùng dữ liệu).
5. (Tuỳ chọn) Ops bấm **Duyệt** trên hàng kế hoạch (cột Bảo trì, căn phải) → UpdateRouting thực thi.
6. (Tuỳ chọn, **Admin**) Chat: *"Liệt kê việc chờ duyệt"* → *"Duyệt routing TOPUP_VINA"* — agent gọi `list_pending_actions` + tool duyệt §7.6.5 (cần `LLM_API_KEY` + role Admin).

**Kết quả kỳ vọng:** Chat và voice cho cùng kết quả; UI usable trên mobile và PC; chat thông minh khi cấu hình GreenNode AIP.

### 10.6 Kịch bản F — Scheduler & auto routing per dịch vụ / SKU (UI)

**Tình huống:** Admin cấu hình trước giờ cao điểm.

1. Mở tab **⚙ Cấu hình** → đặt chu kỳ phân tích **3 phút** → Lưu → scheduler chạy 10:00, 10:03, 10:06…
2. Trên **Dashboard**, SKU `TOPUP_VINA` (scope provider): cột **Chế độ BT / Routing** → **Tự động theo khung giờ** `22:00`–`06:00` → Lưu.
3. **22:30** — Agent phát hiện ESALE xấu → **tự** UpdateRouting (trong khung giờ).
4. **14:00** cùng ngày — cùng tình huống → chỉ Routing Plan + nút **Duyệt** (ngoài khung giờ); không auto.
5. SKU `GARENA/10000` đặt **Tự động** → chu kỳ sau Agent tự routing/bảo trì; Dashboard **không** hiện hàng đề xuất (các SKU khác vẫn Chỉ đề xuất).
6. Tắt **scheduler** → không còn phân tích định kỳ; ops vẫn hỏi qua chat/voice.

**Kết quả kỳ vọng:** Chu kỳ từ Cấu hình; hành vi auto **theo dịch vụ (ưu tiên) hoặc SKU**, không global.

### 10.7 Kịch bản G — Mock data realtime + Agent chạy thử

**Tình huống:** Hackathon demo — chưa có API production.

1. UI Cấu hình: `data_source=mock`, bật Mock Generator, chu kỳ **1 phút**, scenario `**esale_degrading`**.
2. **10:00–10:04** — Generator chạy 5 lần; mock store tích lũy; ESALE trên TOPUP_VINA success giảm dần.
3. **10:05** — Scheduler Agent (5 phút) chạy → GetMetrics đọc mock store → phát hiện xu hướng xấu.
4. Reasoning Rule 1 + 5 → Incident + Routing Plan (provider).
5. UI feed hiển thị incident; ops mở chat hỏi *"ESALE TOPUP_VINA sao rồi?"* → Agent trả lời từ mock data.
6. Đổi scenario `**sku_local_fault`** → vài phút sau Agent báo DATA_VINA / VNP50 (kịch bản D).

**Kết quả kỳ vọng:** Không cần production; data đổi mỗi phút; Agent phân tích và sinh output hợp lệ.

### 10.8 Kịch bản H — Icon trạng thái 3 màu

**Tình huống:** Theo dõi UI qua 2 chu kỳ Agent.

1. **10:00** — Mọi SKU ổn → output `health_status: green` → header/logo 🟢 *Hệ thống OK*.
2. **10:05** — Mock `esale_degrading` → Kế hoạch routing chờ duyệt → tab/SKU liên quan 🔴 (plan chờ); header 🟡 hoặc 🔴 tùy worst SKU.
3. **10:10** — Tỷ lệ thành công ≤ 80% **hoặc** GD pending ≥ ngưỡng → scope live health `red` → cột TT 🔴; **Topup metric OK → cột TT vẫn 🟢** (`liveScopeHealth` per scope).
4. Sau UpdateRouting / metric hồi → live health 🟢 ngay khi **cả 5 metric** trong ngưỡng; Agent `RECOVERING` có thể còn 🟡 trên `/health-status` thêm 1 chu kỳ (§3.1).

**Kết quả kỳ vọng:** Ops nhìn icon biết ngay tình trạng; không cần đọc hết báo cáo.

### 10.9 (đã bỏ) ~~Kịch bản I — Lịch sử thay đổi & Rollback routing~~

> Thay bằng **Mở lại provider** / **Mở lại dịch vụ** trên Dashboard (§8.7). Audit vẫn ghi `agent_change_log` khi `UpdateRouting`.

### 10.10 Kịch bản J — Email ops sau routing/bảo trì khi 🔴

**Tình huống A — Routing + pending cao:** TOPUP_VINA ESALE suy giảm; OpsOne tự động điều phối routing; vẫn 🔴 (pending 18%).

1. Cấu hình §9.5.5: `notification.enabled=true`, recipients có `ops-team@company.com`.
2. `provider_chat_escalation` ESALE: Teams / `[OpsOne] ESALE Support` / `@esale-oncall`.
3. **10:10** — UpdateRouting applied; `health_status=red`, pending ≥ 15%.
4. **Verify email** (MailHog hoặc `notification_log`) — **toàn bộ tiếng Việt**:
  - Subject chứa `[OpsOne 🔴] TOPUP_VINA`.
  - Body: chỉ số, routing trước/sau, block **Leo thang** — không có câu tiếng Anh (trừ mã ESALE, TOPUP_VINA).
  - Tin nhắn mẫu copy được vào Teams.
5. Gửi lại trong 30 phút cùng scope → **không** gửi duplicate (cooldown 60 phút).

**Tình huống B — Bảo trì ZING chỉ ESALE:** Sau approve maintenance active, metric vẫn đỏ.

1. Chạy `db/scenario-b-zing-esale-only.sql`; maintenance 10:15–11:15 active.
2. Email subject `[OpsOne 🔴] ZING — bảo trì ESALE`.
3. Body có `starts_at` / `ends_at` + tin nhắn mẫu tag `@esale-oncall`.

**Kết quả kỳ vọng:** Team ops nhận mail đủ ngữ cảnh; biết vào nhóm chat nào, tag ai để nhờ provider check.

### 10.11 Kịch bản K — Ngưỡng per product → routing + email đồng thời

**Tình huống:** TOPUP_VINA cấu hình `fail_rate_max=10%`, `fail_txn_count_max=120`, `consecutive_cycles_required=2`.

1. Mock `esale_degrading` — chu kỳ 1: fail 8% (chưa vượt) → theo dõi thêm.
2. Chu kỳ 2: fail 11%, fail_txn=130 → `breached=true`, `consecutive=2` → 🔴.
3. OpsOne: Routing Plan + `UpdateRouting` (nếu `auto_action` của scope cho phép §9.5.2).
4. **Cùng chu kỳ:** email ops với block `Ngưỡng vượt: fail_rate 11% > 10%; fail_txn 130 > 120`.
5. Đổi ngưỡng product `fail_txn_count_max=200` trên UI → chu kỳ sau không mail duplicate nếu metric hồi.

**Kịch bản phụ — ZING chỉ ESALE:** vượt `success_rate_min=85%` (actual 42%) → `maintenance` + email (không routing).

**Kết quả kỳ vọng:** Mỗi dịch vụ có ngưỡng riêng; vượt ngưỡng → hành động + mail **đồng bộ**.

### 10.12 Kịch bản L — Admin PUT routing: baseline mới vs chỉnh tạm + auto restore

**Tình huống A — `set_as_baseline=false` (chỉnh tạm):**

**Trạng thái ban đầu:** TOPUP_VINA baseline + traffic = ESALE 70 | IMEDIA 20 | SHOPPAY 10.

1. Mock `esale_degrading` — ESALE xấu; admin mở **Cấu hình → Tỉ lệ routing**.
2. Admin sửa traffic: ESALE 50 | IMEDIA 30 | SHOPPAY 20; **không** tick *"Đặt làm baseline mới"* → `set_as_baseline=false`.
3. `PUT /products/TOPUP_VINA/routing` → verify:
   - `baseline_pct` vẫn 70/20/10.
   - `traffic_pct` = 50/30/20.
   - `routing_scope_state.pending_restore = 1`.
   - UI badge *"Chỉnh tạm — chờ khôi phục"*.
4. Mock chuyển `normal` — metric ổn 2 chu kỳ liên tiếp (§8.6.3).
5. Agent recovery → `traffic_pct` về 70/20/10; `pending_restore=0`; badge mất.

**Tình huống B — `set_as_baseline=true` (baseline mới):**

1. Admin tick ☑ *"Đặt làm baseline mới"*; set ESALE 60 | IMEDIA 25 | SHOPPAY 15.
2. Verify: `baseline_pct = traffic_pct = 60/25/15`; `pending_restore=0`; `config_audit_log` có bản ghi.
3. Sau sự cố + recovery → hệ thống trả về **60/25/15** (baseline mới), không về 70/20/10 cũ.

**Kết quả kỳ vọng:** Checkbox phân biệt rõ Business baseline vs can thiệp tạm; restore đúng mốc.

---

## 11. Checklist triển khai tổng

> **Thứ tự chi tiết:** xem **§2.4** (19 bước verify). Phần dưới gom theo **8 phase** — khớp 1:1 với bảng **「Các bước phát triển chính」** ở đầu tài liệu. AI codegen nên triển khai **tuần tự Phase 0 → 7**, mỗi phase xong mới sang phase kế tiếp.

### Ánh xạ Phase ↔ Roadmap ↔ §2.4

| Phase | Roadmap (đầu file) | §2.4 (verify) | Deliverable chính |
| ----- | ------------------ | ------------- | ----------------- |
| **0** | Bước 1 — Chuẩn bị | #1–2 | `db/schema.sql`, `db/seed.sql`, `internal/store` |
| **1** | Bước 2 — Mock + Scheduler | #3, #7 (dry-run) | `cmd/worker-mock`, `agent_analysis_*` |
| **2** | Bước 3 — Tools | #4–5 | `internal/tools`, `internal/threshold`, `internal/notify` |
| **3** | Bước 4 — Agent core | #7–8 | `cmd/worker-agent`, `internal/agent` |
| **4** | Bước 5 — Reasoning + Output | #6, #12 (partial) | `internal/rules`, output §8, `agent_change_log` |
| **5** | Bước 6 — REST API | #9 | `cmd/api`, SSE, `dashboard/overview` ✅ |
| **6** | Bước 7 — Frontend React | #10–11, #13 | `web/` Dashboard §9.0 ✅ |
| **7** | Bước 8 — E2E & DoD | #12–19, §12 | Kịch bản §10, `make test`, demo end-to-end |

### Phase 0 — Chuẩn bị (Setup)

- Khởi tạo monorepo §2.2 (`cmd/`, `internal/`, `web/`, `db/`)
- Tạo database MySQL + chạy DDL **§13** (`db/schema.sql`, `db/seed.sql`).
- Xác định product pilot từ **catalog §1.1** (11 product hoặc subset rollout):
  - **Thẻ (`card`):** ZING, GARENA, VINAPHONE, MOBIFONE, VIETTEL
  - **Topup data (`topup_data`):** DATA_VINA, DATA_MOBI, DATA_VIETTEL
  - **Topup (`topup`):** TOPUP_VINA, TOPUP_MOBI, TOPUP_VIETTEL
- Liệt kê provider cho từng product; **SKU thẻ** theo mệnh giá; **SKU topup data:** VNP20, VNP50, V50K, V100K
- **Business set baseline routing** (§8.6 Pha A): nhập `baseline_pct` cho từng `(product, sku, provider)` — đây là quyết định kinh doanh, KHÔNG do Agent sinh; khi seed `traffic_pct = baseline_pct`.
- Chọn `data_source`: `mock` (chạy thử) hoặc `production`
- Mặc định mọi scope `auto_action=recommend_only` (seed `routing_scope_state` per SKU + topup provider); admin đổi **cấp dịch vụ** (`PUT /scopes/{product}/auto`) hoặc **cấp SKU** trên Dashboard (§9.5.2)

### Phase 1 — Mock Data + Scheduler

- Mock Data Generator: cron **1 phút** → bảng §13.5
- Scenario inject: `normal`, `esale_degrading` (tối thiểu)
- Scheduler Agent: chu kỳ 5 phút (hoặc 3 phút khi demo)
- Schema MySQL đầy đủ **§13** (`agent_analysis_cycles`, `agent_analysis_history`, …)
- Chạy dry-run: chỉ ghi history, chưa reasoning

### Phase 2 — Tools

- GetMetrics, GetTopErrors, GetProviders, GetSkus (`routing_mode=sku`), GetRouting, GetRevenue
- Test: topup tiền `scope=provider`; thẻ / topup data routing sku
- UpdateRouting: topup `scope=provider`; card / topup_data `scope=sku | sku_batch` (mock trước)
- GetMaintenance / SetMaintenance — cửa sổ starts_at / ends_at (§8.5, kịch bản B, B′)
- `internal/threshold` — EvaluateThresholds + `suggested_action` theo `active_provider_count` (§7.4)
- `internal/notify` — email khi vượt ngưỡng + routing/bảo trì (§8.9, J, K)
- Mỗi UpdateRouting ghi `agent_change_log` (audit DB).

### Phase 3 — Agent Core

- Loop theo `routing_mode`: provider-mode → product → provider; sku-mode → product → sku → provider
- Gom context đủ cho Reasoning

### Phase 4 — Reasoning + Output

- Implement rules §7.3 (1–9; rule 7–8 cho `routing_mode=sku`)
- Template Incident / Recommendation / Maintenance / Routing Plan / **Health Status** — **toàn bộ tiếng Việt** (§2.5)
- **1 sự cố mở / SKU** — `FindOpenIncidentForScope` trước `InsertIncident` (§8.3.1)
- LLM summary + fallback template (§7.6); **Rules trước LLM**
- Chạy kịch bản A–D, H (gồm **H — icon trạng thái 3 màu**) — verify qua DB/API trước khi có UI đầy đủ

### Phase 5 — REST API ✅ (triển khai `ai_gen_src`)

- [x] `cmd/api` — `:8080`; CORS `http://localhost:5173`
- [x] Handlers §2.3: … `mock`, **`POST /chat` — LLM agent §7.6.5** (stub keyword khi không có `LLM_API_KEY`), SSE `/events`
- [x] JSON **snake_case** — `internal/api/serialize.go`; config PUT `scheduler_interval_min`, …
- [x] `DEV_AUTH_BYPASS` + header `X-OpsOne-Role` (dev); `POST .../approve`
- [x] Integration test `internal/api/server_test.go` — health, config audit, incident get, `TestProductScopeAutoPutOverview`, `TestConfigPutMaintenanceDefaultDuration` (`maintenance_default_duration_min`)
- [ ] O365 JWT middleware đầy đủ §2.6 (production)
- [x] `PUT /products/{code}/thresholds`; maintenance approve/cancel (partial); `POST /notifications/test` (stub)

### Phase 6 — Frontend (React) ✅ (triển khai `ai_gen_src`)

- [x] Vite 5 + React 18 + TypeScript + React Router + TanStack Query
- [x] Routes §9; `Layout` + SSE `useSSE` (poll 30s fallback)
- [x] **Dashboard §9.0** — `ServiceOverviewTable` (cột provider 6 chỉ số + hàng plan/bảo trì/**Mở lại provider**; Lưu baseline → `restore-baseline`), `IncidentsPage` phân trang full-width
- [x] `ProviderMetricCell`, `scopeMetrics.ts`, `effectiveRowHealth`; `ProductThresholdEditor` ngưỡng inline
- [x] `HealthBadge` compact, `MaintenancePage`, `ChatWidget` + voice
- [x] `Settings` — card compact §9.5 (`PUT /config`: scheduler, `maintenance_default_duration_min`, mock); ngưỡng §9.5.3 + **Auto per dịch vụ / SKU** (`ScopeAutoEditor`) trên Dashboard
- [x] `web/dev.ps1` — chạy UI khi PATH chưa có `npm` (Windows)
- [ ] MSAL.js khi `VITE_AAD_*` (§2.6.4); form routing `set_as_baseline` §9.5.4.1
- [ ] E2E kịch bản **E, F, G, H** đầy đủ (→ Phase 7)

### Phase 7 — E2E & Definition of Done ✅

- [x] `internal/e2e/` — kịch bản cốt lõi §10: **A** (TOPUP routing seed), **H** (health API), approve plan → `agent_change_log`
- [x] **G** — kiểm tra `agent_analysis_cycles` (≥1; `-Full` / `OPSONE_E2E_FULL=1` chạy 2 cycle mới)
- [x] Audit: `config_audit_log` sau PUT config; incidents, dashboard overview
- [x] `make test-e2e`, `scripts/e2e.ps1`, `Invoke-OpsOneE2E`; Makefile seed UTF-8 (`docker cp`)
- [x] Tài liệu vận hành ngắn trong `README.md` (ai approve / auto)
- [ ] Toàn bộ §10 **B–F, J–L** (maintenance ZING, chat/voice + **chat duyệt Admin** E2E, email ngưỡng live, baseline checkbox)
- [ ] Alert worker crash; React test CI
- **DoD §12** — demo end-to-end **không** nối production: `docker compose up` + 3 binary + `.\scripts\e2e.ps1`

---

## 12. Định nghĩa hoàn thành (Definition of Done)

Hệ thống **chạy được** khi đủ các điều kiện sau (stack **Go + MySQL + React**):

**Backend (Go):**

- `docker compose up` → MySQL healthy; `make migrate && make seed` OK
- `cmd/worker-mock` chạy nền — `mock_metrics` tăng mỗi phút
- `cmd/worker-agent` tự chạy theo `scheduler_interval_min` — không cần user
- `cmd/api` — REST §2.3 trả JSON; `go test ./...` pass
- Pipeline §3: 9 tools → rules → output → UpdateRouting / SetMaintenance → `agent_change_log`
- Ghi `agent_analysis_cycles`, `agent_analysis_history`; audit `agent_change_log` §8.7

**Agent output:**

- **Health Status** 🟢🟡🔴 — kịch bản H
- **Routing Plan** SKU (kịch bản C, D) và provider (kịch bản A)
- **Maintenance** ZING — chỉ 1 provider active, có `starts_at` / `ends_at` — kịch bản B, B′ (§10.2b)
- ≥ 2 chu kỳ history để thấy xu hướng (kịch bản G)
- **Email ops** — nội dung tiếng Việt — kịch bản J
- **Ngưỡng per product** → routing/bảo trì + mail đồng thời — kịch bản K

**Frontend (React):**

- [x] `cd web && .\dev.ps1` — Dashboard §9.0: bảng routing/bảo trì/ngưỡng; provider `routing_pct=0` vẫn hiện + **Mở lại**; bảng incidents phân trang (TT + người xử lý)
- [x] Top-nav tabs; refresh 60s + SSE; duyệt routing inline + banner; đóng sự cố kèm audit
- [x] `/incidents` bảng sự cố (handled_by, handled_at, resolution_action — không route detail); Chat dock + voice (E, F partial)
- [ ] Routing `set_as_baseline` §9.5.4.1 (J, L)
- [ ] MSAL O365 production; responsive audit đủ §9.3 trên thiết bị thật

**Dữ liệu:**

- Catalog 11 product + SKU + routing seed trong MySQL

---

# Part VI — Data Layer (MySQL)

> **File triển khai:** `db/schema.sql` (DROP + CREATE toàn bộ bảng) · `db/seed.sql` · `db/clear_runtime.sql`.  
> **Cập nhật DDL:** chỉnh `schema.sql` → `Invoke-OpsOneReset` (xóa dữ liệu runtime + catalog, tạo lại từ đầu).  
> Chạy: `Invoke-OpsOneReset` (Windows) hoặc `make db-up && make migrate && make seed && make db-verify`

## 13. Thiết kế Database (MySQL)

> **Engine:** InnoDB · **Charset:** `utf8mb4` · **Quy ước SKU:** topup tiền dùng `sku = ''` (empty string), không dùng `NULL` — thuận tiện cho UNIQUE index.

### 13.1 Sơ đồ quan hệ (tóm tắt)

```text
products ──┬── product_skus
           ├── product_providers ── providers
           ├── product_alert_thresholds
           └── routing_config
           └── routing_scope_state   (auto_action, window_*, pending_restore — §13.3.1)

agent_settings (1 row)
agent_analysis_cycles ──┬── agent_analysis_history
                        ├── health_status_product
                        ├── incidents
                        ├── routing_plans ── agent_change_log (rollback_of_id)
                        ├── maintenance_windows
                        └── recommendations

mock_generator_run        mock_metrics / mock_error_stats
metrics_snapshot          (production metrics)

config_audit_log          (agent_settings — admin, không rollback routing)
provider_chat_escalation  notification_log
chat_intent_stats         (FAQ intent hit count — §7.6.5.3 ✅)
chat_interaction_log      (mỗi lượt /chat — route, slots, result — §7.6.5.5 ✅ P1–P2)
chat_sessions ── chat_messages   (persist hội thoại — §7.6.5.5 ✅ P1)
chat_command_patterns     (pattern lệnh candidate/approved — §7.6.5.5 📋 P3)
chat_feedback             (up/down/corrected — §7.6.5.5 📋)
chat_few_shot_examples    (ví dụ prompt LLM — §7.6.5.5 📋 P4)
chat_voice_corrections    (STT heard→corrected — §7.6.5.5 📋 P5)
chat_user_prefs           (sync profile ops — §7.6.5.5 📋)
users                     (cache O365/Entra ID profile — §2.6.9)
```

### 13.2 Catalog — sản phẩm & SKU

```sql
CREATE TABLE products (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_code  VARCHAR(32)  NOT NULL COMMENT 'ZING, TOPUP_VINA, ...',
  label         VARCHAR(128) NOT NULL,
  service_type  ENUM('card','topup_data','topup') NOT NULL,
  routing_mode  ENUM('sku','provider') NOT NULL,
  enabled       TINYINT(1)   NOT NULL DEFAULT 1,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_products_code (product_code),
  KEY idx_products_enabled (enabled, service_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE providers (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  provider_code VARCHAR(32) NOT NULL COMMENT 'ESALE, IMEDIA, SHOPPAY',
  label         VARCHAR(64) NOT NULL,
  enabled       TINYINT(1)  NOT NULL DEFAULT 1,
  UNIQUE KEY uk_providers_code (provider_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product_providers (
  product_id    INT UNSIGNED NOT NULL,
  provider_id   INT UNSIGNED NOT NULL,
  enabled       TINYINT(1)   NOT NULL DEFAULT 1 COMMENT '1=active cho product; 0=inactive — §6.3 §7.4',
  PRIMARY KEY (product_id, provider_id),
  KEY idx_pp_product_enabled (product_id, enabled),
  CONSTRAINT fk_pp_product  FOREIGN KEY (product_id)  REFERENCES products(id),
  CONSTRAINT fk_pp_provider FOREIGN KEY (provider_id) REFERENCES providers(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- Provider active = product_providers.enabled=1 AND providers.enabled=1 (§7.4)

CREATE TABLE product_skus (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_id    INT UNSIGNED NOT NULL,
  sku_code      VARCHAR(32)  NOT NULL COMMENT '20000, VNP50, ...',
  label         VARCHAR(128) NOT NULL,
  enabled       TINYINT(1)   NOT NULL DEFAULT 1,
  UNIQUE KEY uk_product_sku (product_id, sku_code),
  KEY idx_skus_product (product_id),
  CONSTRAINT fk_skus_product FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 13.2.1 Ngưỡng cảnh báo theo dịch vụ (§1.2)

```sql
CREATE TABLE product_alert_thresholds (
  id                          INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_code                VARCHAR(32)  NOT NULL,
  enabled                     TINYINT(1)   NOT NULL DEFAULT 1,
  success_rate_min_pct        DECIMAL(5,2) NULL COMMENT 'NULL = dùng default hệ thống',
  pending_rate_max_pct        DECIMAL(5,2) NULL,
  fail_rate_max_pct           DECIMAL(5,2) NULL,
  fail_txn_count_max          INT UNSIGNED NULL COMMENT 'số GD lỗi trong window',
  error_event_count_max       INT UNSIGNED NULL COMMENT 'sum error_count',
  pending_txn_count_max       INT UNSIGNED NULL,
  metrics_window_min          TINYINT UNSIGNED NOT NULL DEFAULT 15,
  consecutive_cycles_required TINYINT UNSIGNED NOT NULL DEFAULT 2,
  alert_email_enabled         TINYINT(1)   NOT NULL DEFAULT 1,
  updated_at                  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by                  VARCHAR(64)  NULL,
  UNIQUE KEY uk_alert_product (product_code),
  CONSTRAINT fk_alert_product FOREIGN KEY (product_code) REFERENCES products(product_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 13.3 Routing hiện tại + Baseline (Business config)

> **Hai trường tách bạch (§8.6.1):**
> - `baseline_pct`: tỉ lệ **Business cấu hình** — chỉ Admin sửa, không bao giờ Agent ghi đè.
> - `traffic_pct`: tỉ lệ **đang chạy** — Agent có thể đổi khi sự cố (Pha B); Recovery sẽ khôi phục `traffic_pct = baseline_pct`.
> Khi seed lần đầu: `traffic_pct = baseline_pct` (chưa có sự cố).

```sql
CREATE TABLE routing_config (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_code    VARCHAR(32)  NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '' COMMENT 'empty = topup provider-level',
  provider_code   VARCHAR(32)  NOT NULL,
  baseline_pct    DECIMAL(5,2) NOT NULL COMMENT 'Business config — chỉ Admin sửa (§8.6 Pha A)',
  traffic_pct     DECIMAL(5,2) NOT NULL COMMENT 'Đang chạy — Agent có thể đổi khi sự cố (Pha B)',
  baseline_updated_at DATETIME NULL,
  baseline_updated_by VARCHAR(64) NULL,
  updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'lần cuối traffic_pct đổi',
  updated_by      VARCHAR(64)  NULL COMMENT 'opsone-agent | admin email',
  UNIQUE KEY uk_routing (product_code, sku_code, provider_code),
  KEY idx_routing_product (product_code, sku_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Quy ước truy cập:**

| Thao tác | Cột được phép ghi | Ai ghi |
|----------|-------------------|--------|
| Seed / Admin PUT `set_as_baseline=true` | `baseline_pct`, `traffic_pct`, `baseline_updated_*`; `pending_restore=0` | Admin — `config_audit_log` + `agent_change_log` `manual_baseline` |
| Admin PUT `set_as_baseline=false` | Chỉ `traffic_pct`; `routing_scope_state.pending_restore=1` | Admin — `agent_change_log` `manual_temp` |
| Agent UpdateRouting (Pha B) | `traffic_pct`, `updated_*` | `opsone-agent` — log `agent_change_log` `auto` |
| Recovery (Pha C) | `traffic_pct = baseline_pct`; `pending_restore=0` | `opsone-agent` — `reason=recovery_to_baseline` hoặc `restore_after_manual_override` |

**Validate (trước commit):**

- `Σ baseline_pct == 100` per `(product_code, sku_code)`.
- `Σ traffic_pct == 100` per `(product_code, sku_code)`.
- Mỗi `traffic_pct` trong **0–100%**; `Σ traffic_pct == 100` (§8.6.3).

### 13.3.1 Trạng thái scope routing — `pending_restore` (§8.6.5)

Theo dõi admin chỉnh **tạm thời** (`set_as_baseline=false`) — Agent recovery trả `traffic_pct` về `baseline_pct` khi ổn định.

```sql
CREATE TABLE routing_scope_state (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_code        VARCHAR(32)  NOT NULL,
  sku_code            VARCHAR(32)  NOT NULL DEFAULT '' COMMENT 'empty = topup provider-level',
  auto_action         ENUM('recommend_only','auto','time_window') NOT NULL DEFAULT 'recommend_only' COMMENT '§9.5.2 — auto routing per scope',
  window_start        TIME         NULL COMMENT 'Bắt đầu khung giờ khi auto_action=time_window',
  window_end          TIME         NULL COMMENT 'Kết thúc khung giờ (có thể qua nửa đêm)',
  pending_restore     TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '1=admin chỉnh tạm — chờ restore baseline khi ổn',
  manual_override_by  VARCHAR(128) NULL COMMENT 'UPN admin — JWT preferred_username',
  manual_override_at  DATETIME     NULL,
  recovery_apply_cycle_id BIGINT UNSIGNED NULL COMMENT 'Chu kỳ apply routing — +1 chu kỳ 🟡, +2 chu kỳ 🟢 nếu OK',
  updated_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_routing_scope (product_code, sku_code),
  KEY idx_pending_restore (pending_restore, product_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Quy ước:**

- `pending_restore=0` — traffic khớp baseline hoặc admin vừa set baseline mới.
- `pending_restore=1` — admin PUT với `set_as_baseline=false`; UI badge *"Chỉnh tạm — chờ khôi phục"*.
- Recovery (§8.6.3 Bước 4): `traffic_pct ← baseline_pct` rồi `pending_restore=0`.
- `recovery_apply_cycle_id`: set khi `UpdateRouting` (`admin_approve` / `auto`); xóa khi health 🟢 sau chu kỳ +2.

### 13.4 Cấu hình Agent (UI Settings)

`GET/PUT /config` (triển khai) expose: `scheduler_*`, `mock_*`, `maintenance_default_duration_min` (§9.5.5). Các cột `notification_*`, `maintenance_auto_enabled` — schema có, UI chưa.

```sql
CREATE TABLE agent_settings (
  id                      TINYINT UNSIGNED NOT NULL PRIMARY KEY DEFAULT 1,
  scheduler_enabled       TINYINT(1)       NOT NULL DEFAULT 1,
  scheduler_interval_min  TINYINT UNSIGNED NOT NULL DEFAULT 5,
  data_source             ENUM('mock','production') NOT NULL DEFAULT 'mock',
  mock_enabled            TINYINT(1)       NOT NULL DEFAULT 1,
  mock_interval_min       TINYINT UNSIGNED NOT NULL DEFAULT 1,
  mock_scenario           ENUM('normal','esale_degrading','sku_local_fault','random_spike') NOT NULL DEFAULT 'normal',
  mock_retention_hours    SMALLINT UNSIGNED NOT NULL DEFAULT 24,
  maintenance_default_duration_min TINYINT UNSIGNED NOT NULL DEFAULT 60 COMMENT 'thời lượng bảo trì mặc định',
  maintenance_auto_enabled TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '1=auto SetMaintenance khi active_provider_count==1 hoặc không có backup healthy',
  notification_enabled  TINYINT(1)       NOT NULL DEFAULT 1,
  notification_recipients JSON             NOT NULL COMMENT '["ops-team@company.com"]',
  notification_on_red_only TINYINT(1)     NOT NULL DEFAULT 1,
  notification_pending_threshold DECIMAL(5,2) NOT NULL DEFAULT 15.00,
  notification_fail_threshold DECIMAL(5,2) NOT NULL DEFAULT 10.00,
  notification_cooldown_min TINYINT UNSIGNED NOT NULL DEFAULT 60,
  default_success_rate_min_pct DECIMAL(5,2) NOT NULL DEFAULT 80.00,
  default_pending_rate_max_pct DECIMAL(5,2) NOT NULL DEFAULT 15.00,
  default_fail_rate_max_pct DECIMAL(5,2) NOT NULL DEFAULT 10.00,
  default_fail_txn_count_max INT UNSIGNED NOT NULL DEFAULT 50,
  default_error_event_count_max INT UNSIGNED NOT NULL DEFAULT 50,
  default_pending_txn_count_max INT UNSIGNED NULL,
  default_metrics_window_min TINYINT UNSIGNED NOT NULL DEFAULT 15,
  default_consecutive_cycles_required TINYINT UNSIGNED NOT NULL DEFAULT 2,
  routing_good_threshold_pct DECIMAL(5,2) NOT NULL DEFAULT 90.00 COMMENT 'success_rate ≥ ngưỡng → coi là healthy backup (§8.6.3)',
  routing_min_improvement_pct DECIMAL(5,2) NOT NULL DEFAULT 5.00 COMMENT 'expected_success phải cải thiện ≥ % này mới apply',
  routing_recovery_cycles  TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT 'chu kỳ ổn định sau vàng trước restore baseline (tổng +2 chu kỳ → xanh)',
  routing_recovery_buffer_pct DECIMAL(5,2) NOT NULL DEFAULT 5.00 COMMENT 'success ≥ threshold + buffer mới tính là hồi',
  agent_locale            VARCHAR(8)       NOT NULL DEFAULT 'vi-VN',
  updated_at              DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by              VARCHAR(64)      NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE config_audit_log (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  changed_by    VARCHAR(64)  NOT NULL,
  change_type   VARCHAR(32)  NOT NULL COMMENT 'agent_settings',
  before_json   JSON         NULL,
  after_json    JSON         NOT NULL,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_config_audit_time (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 13.4.1 Thông báo email & leo thang chat (§8.9)

```sql
CREATE TABLE provider_chat_escalation (
  id              INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  provider_code   VARCHAR(32)  NOT NULL,
  chat_app_name   VARCHAR(64)  NOT NULL COMMENT 'Microsoft Teams, Zalo, Telegram...',
  chat_group_name VARCHAR(128) NOT NULL,
  mention_tags    VARCHAR(256) NOT NULL COMMENT '@esale-oncall @n2-support',
  enabled         TINYINT(1)   NOT NULL DEFAULT 1,
  updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_escalation_provider (provider_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE notification_log (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  dedupe_key          VARCHAR(128) NOT NULL COMMENT 'cooldown — tránh gửi trùng',
  trigger_event       ENUM('routing_applied','maintenance_active','maintenance_scheduled') NOT NULL,
  health_status       ENUM('green','yellow','red') NOT NULL,
  product_code        VARCHAR(32)  NOT NULL,
  provider_code       VARCHAR(32)  NOT NULL,
  sku_code            VARCHAR(32)  NOT NULL DEFAULT '',
  cycle_id            BIGINT UNSIGNED NULL,
  incident_id         VARCHAR(32)  NULL,
  agent_change_id     BIGINT UNSIGNED NULL,
  maintenance_id      VARCHAR(32)  NULL,
  metrics_snapshot    JSON         NOT NULL COMMENT 'rates, counts, breach_reasons[]',
  action_summary      TEXT         NOT NULL,
  chat_escalation_json JSON        NOT NULL COMMENT 'app, group, tags, sample_message',
  recipients          JSON         NOT NULL,
  subject             VARCHAR(256) NOT NULL,
  status              ENUM('pending','sent','failed') NOT NULL DEFAULT 'pending',
  error_message       VARCHAR(512) NULL,
  sent_at             DATETIME     NULL,
  created_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_notification_dedupe (dedupe_key),
  KEY idx_notification_time (created_at),
  KEY idx_notification_product (product_code, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 13.5 Mock data (Generator 1 phút)

```sql
CREATE TABLE mock_generator_run (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  started_at    DATETIME     NOT NULL,
  finished_at   DATETIME     NULL,
  scenario      VARCHAR(32)  NOT NULL,
  rows_metrics  INT UNSIGNED NOT NULL DEFAULT 0,
  status        ENUM('running','success','failed') NOT NULL DEFAULT 'running',
  error_message VARCHAR(512) NULL,
  KEY idx_mock_run_started (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE mock_metrics (
  id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  recorded_at        DATETIME       NOT NULL,
  product_code       VARCHAR(32)    NOT NULL,
  sku_code           VARCHAR(32)    NOT NULL DEFAULT '',
  provider_code      VARCHAR(32)    NOT NULL,
  success_rate       DECIMAL(5,2)   NOT NULL,
  pending_rate       DECIMAL(5,2)   NOT NULL,
  fail_rate          DECIMAL(5,2)   NOT NULL,
  total_transactions INT UNSIGNED   NOT NULL DEFAULT 0,
  revenue_last_hour  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  generator_run_id   BIGINT UNSIGNED NULL,
  KEY idx_mock_metrics_lookup (product_code, provider_code, sku_code, recorded_at),
  KEY idx_mock_metrics_time (recorded_at),
  CONSTRAINT fk_mock_metrics_run FOREIGN KEY (generator_run_id) REFERENCES mock_generator_run(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE mock_error_stats (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  recorded_at     DATETIME     NOT NULL,
  product_code    VARCHAR(32)  NOT NULL,
  provider_code   VARCHAR(32)  NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '',
  error_code      VARCHAR(16)  NOT NULL,
  error_count     INT UNSIGNED NOT NULL,
  generator_run_id BIGINT UNSIGNED NULL,
  KEY idx_mock_errors_lookup (product_code, provider_code, recorded_at),
  KEY idx_mock_errors_time (recorded_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Query GetMetrics (mock, window 15m):**

```sql
SELECT success_rate, pending_rate, fail_rate, total_transactions
FROM mock_metrics
WHERE product_code = ?
  AND provider_code = ?
  AND sku_code = ?
  AND recorded_at >= NOW() - INTERVAL 15 MINUTE
ORDER BY recorded_at DESC
LIMIT 1;
```

### 13.6 Production metrics (tuỳ chọn — khi `data_source=production`)

```sql
CREATE TABLE metrics_snapshot (
  id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  recorded_at        DATETIME       NOT NULL,
  product_code       VARCHAR(32)    NOT NULL,
  sku_code           VARCHAR(32)    NOT NULL DEFAULT '',
  provider_code      VARCHAR(32)    NOT NULL,
  success_rate       DECIMAL(5,2)   NOT NULL,
  pending_rate       DECIMAL(5,2)   NOT NULL,
  fail_rate          DECIMAL(5,2)   NOT NULL,
  total_transactions INT UNSIGNED   NOT NULL DEFAULT 0,
  revenue_last_hour  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  KEY idx_metrics_lookup (product_code, provider_code, sku_code, recorded_at),
  KEY idx_metrics_time (recorded_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 13.7 Lịch sử phân tích Agent

```sql
CREATE TABLE agent_analysis_cycles (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_started   DATETIME     NOT NULL,
  cycle_finished  DATETIME     NULL,
  data_source     ENUM('mock','production') NOT NULL,
  health_status   ENUM('green','yellow','red') NOT NULL DEFAULT 'green',
  health_summary  VARCHAR(512) NULL,
  decision        VARCHAR(32)  NULL COMMENT 'monitor, incident, shift_traffic, maintenance',
  status          ENUM('running','success','failed') NOT NULL DEFAULT 'running',
  KEY idx_cycles_started (cycle_started),
  KEY idx_cycles_health (health_status, cycle_started)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE agent_analysis_history (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id        BIGINT UNSIGNED NOT NULL,
  recorded_at     DATETIME     NOT NULL,
  product_code    VARCHAR(32)  NOT NULL,
  service_type    ENUM('card','topup_data','topup') NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '',
  provider_code   VARCHAR(32)  NOT NULL,
  success_rate    DECIMAL(5,2) NOT NULL,
  pending_rate    DECIMAL(5,2) NOT NULL,
  fail_rate       DECIMAL(5,2) NOT NULL,
  KEY idx_history_cycle (cycle_id),
  KEY idx_history_lookup (product_code, provider_code, sku_code, recorded_at),
  CONSTRAINT fk_history_cycle FOREIGN KEY (cycle_id) REFERENCES agent_analysis_cycles(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE health_status_product (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id        BIGINT UNSIGNED NOT NULL,
  product_code    VARCHAR(32)  NOT NULL,
  health_status   ENUM('green','yellow','red') NOT NULL,
  health_summary  VARCHAR(256) NULL,
  UNIQUE KEY uk_health_cycle_product (cycle_id, product_code),
  CONSTRAINT fk_health_cycle FOREIGN KEY (cycle_id) REFERENCES agent_analysis_cycles(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- State machine §3.1 — mỗi chu kỳ 1 dòng per (product, sku)
CREATE TABLE agent_state_history (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id          BIGINT UNSIGNED NOT NULL,
  product_code      VARCHAR(32)  NOT NULL,
  sku_code          VARCHAR(32)  NOT NULL DEFAULT '',
  state             ENUM('NORMAL','WARNING','INCIDENT',
                         'ROUTING_PROPOSED','MAINTENANCE_PROPOSED',
                         'ROUTING_APPLIED','MAINTENANCE_ACTIVE','RECOVERING') NOT NULL,
  prev_state        ENUM('NORMAL','WARNING','INCIDENT',
                         'ROUTING_PROPOSED','MAINTENANCE_PROPOSED',
                         'ROUTING_APPLIED','MAINTENANCE_ACTIVE','RECOVERING') NULL,
  transition_reason VARCHAR(256) NULL COMMENT 'VD: breached_consecutive_2; routing_apply_ok; recovery_2_cycles',
  recorded_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_state_lookup (product_code, sku_code, recorded_at),
  KEY idx_state_cycle (cycle_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 13.8 Output — Incident, Routing Plan, Recommendation

```sql
CREATE TABLE incidents (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  incident_id     VARCHAR(32)  NOT NULL COMMENT '20260604-001',
  cycle_id        BIGINT UNSIGNED NULL,
  severity        ENUM('low','medium','high') NOT NULL,
  product_code    VARCHAR(32)  NOT NULL,
  provider_code   VARCHAR(32)  NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '',
  sku_label       VARCHAR(64)  NULL,
  success_before  DECIMAL(5,2) NULL,
  success_after   DECIMAL(5,2) NULL,
  fail_before     DECIMAL(5,2) NULL,
  fail_after      DECIMAL(5,2) NULL,
  summary         TEXT         NULL,
  status          ENUM('open','acknowledged','resolved') NOT NULL DEFAULT 'open',
  handled_by      VARCHAR(64)  NULL COMMENT 'Actor xử lý sự cố',
  handled_at      DATETIME     NULL COMMENT 'Thời điểm xử lý',
  resolution_action ENUM('admin_approve','admin_reject','auto') NULL COMMENT 'Hành động đóng sự cố',
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_incident_id (incident_id),
  KEY idx_incidents_product (product_code, created_at),
  KEY idx_incidents_status (status, severity)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE routing_plans (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id        BIGINT UNSIGNED NULL,
  product_code    VARCHAR(32)  NOT NULL,
  scope           ENUM('provider','sku','sku_batch') NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '',
  plan_json       JSON         NOT NULL COMMENT 'current, performance, suggested, expected',
  status          ENUM('draft','pending_approve','approved','rejected','executed') NOT NULL DEFAULT 'draft',
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  approved_by     VARCHAR(64)  NULL,
  approved_at     DATETIME     NULL,
  KEY idx_routing_plans_product (product_code, created_at),
  KEY idx_routing_plans_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE recommendations (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id        BIGINT UNSIGNED NULL,
  incident_id     VARCHAR(32)  NULL,
  product_code    VARCHAR(32)  NOT NULL,
  action_type     VARCHAR(32)  NOT NULL COMMENT 'shift_traffic, monitor, escalate, maintenance',
  action_detail   TEXT         NOT NULL,
  monitor_until   DATETIME     NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_recommendations_product (product_code, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE maintenance_windows (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  maintenance_id    VARCHAR(32)  NOT NULL COMMENT '20260604-M001',
  product_code      VARCHAR(32)  NOT NULL,
  provider_code     VARCHAR(32)  NOT NULL,
  sku_code          VARCHAR(32)  NOT NULL DEFAULT '',
  starts_at         DATETIME     NOT NULL COMMENT 'thời gian bắt đầu bảo trì',
  ends_at           DATETIME     NOT NULL COMMENT 'thời gian kết thúc bảo trì',
  status            ENUM('pending_approve','scheduled','active','completed','cancelled') NOT NULL DEFAULT 'pending_approve',
  trigger_type      ENUM('opsone_recommend','opsone_auto','admin_manual') NOT NULL,
  severity          ENUM('low','medium','high') NULL,
  reason            TEXT         NULL,
  cycle_id          BIGINT UNSIGNED NULL,
  incident_id       VARCHAR(32)  NULL,
  approved_by       VARCHAR(64)  NULL,
  approved_at       DATETIME     NULL,
  cancelled_by      VARCHAR(64)  NULL,
  cancelled_at      DATETIME     NULL,
  created_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_maintenance_id (maintenance_id),
  KEY idx_maintenance_product (product_code, status, starts_at),
  KEY idx_maintenance_active (status, ends_at),
  KEY idx_maintenance_window (product_code, provider_code, sku_code, starts_at, ends_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Query GetMaintenance / ListActiveMaintenance — cửa sổ theo scope:**

```sql
SELECT maintenance_id, product_code, provider_code, sku_code, starts_at, ends_at, status, reason
FROM maintenance_windows
WHERE product_code = ?
  /* AND provider_code = ?  -- optional */
  /* AND sku_code = ?       -- optional */
  AND status IN ('scheduled', 'active')
ORDER BY sku_code ASC, provider_code ASC, starts_at ASC;
```

**Lọc in-window (Go — giống `GET /dashboard/overview`):** `!ends_at.Before(now) && !starts_at.After(now)` → **active**; `starts_at.After(now)` → **scheduled**; hết hạn → bỏ qua.

**Chat §7.6.5.2:** `metricsForChat` dùng `GetMetricsInWindow` per provider (cửa sổ 15m) + `pending_txn`/`fail_txn` như `provider_metrics` overview.

**Chat §7.6.5.1:** `maintenanceForChat` dùng `ListMaintenanceWindows(product, 'active')` + cùng bộ lọc Go như overview (không dùng `ends_at > NOW()` trong SQL).

**Query tick lifecycle (worker mỗi phút):**

```sql
-- scheduled → active
UPDATE maintenance_windows
SET status = 'active'
WHERE status = 'scheduled'
  AND starts_at <= NOW()
  AND ends_at > NOW();

-- active → completed
UPDATE maintenance_windows
SET status = 'completed'
WHERE status = 'active'
  AND ends_at <= NOW();
```

```sql
CREATE TABLE agent_change_log (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  change_type       ENUM('routing') NOT NULL DEFAULT 'routing' COMMENT 'mở rộng: maintenance',
  product_code      VARCHAR(32)  NOT NULL,
  scope             ENUM('provider','sku','sku_batch') NOT NULL,
  sku_code          VARCHAR(32)  NOT NULL DEFAULT '',
  routing_before    JSON         NOT NULL,
  routing_after     JSON         NOT NULL,
  trigger_type      ENUM('auto','admin_approve','manual_baseline','manual_temp','rollback') NOT NULL,
  change_status     ENUM('applied','rolled_back','superseded') NOT NULL DEFAULT 'applied',
  cycle_id          BIGINT UNSIGNED NULL,
  routing_plan_id   BIGINT UNSIGNED NULL,
  incident_id       VARCHAR(32)  NULL,
  rollback_of_id    BIGINT UNSIGNED NULL COMMENT 'bản ghi rollback trỏ về change gốc',
  reason            TEXT         NULL,
  executed_by       VARCHAR(64)  NULL COMMENT 'opsone-agent | admin email',
  rolled_back_at    DATETIME     NULL,
  rolled_back_by    VARCHAR(64)  NULL,
  created_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_agent_changes_product (product_code, created_at),
  KEY idx_agent_changes_status (change_status, product_code, scope, sku_code, created_at),
  KEY idx_agent_changes_cycle (cycle_id),
  CONSTRAINT fk_agent_change_rollback FOREIGN KEY (rollback_of_id) REFERENCES agent_change_log(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

> **Cột legacy:** `rollback_of_id` — giữ trong schema; không dùng sau khi bỏ tính năng rollback.

```sql
SELECT id, routing_before, routing_after
FROM agent_change_log
WHERE product_code = ?
  AND scope = ?
  AND sku_code = ?
  AND change_type = 'routing'
  AND change_status = 'applied'
ORDER BY created_at DESC
LIMIT 1;
```

### 13.9 Chat (UI on-demand)

**Triển khai:** `chat_intent_stats` ✅ (kèm `route_key`, success/fail) · `chat_sessions` / `chat_messages` / `chat_interaction_log` ✅ P1–P2 · `chat_command_patterns` / `chat_feedback` / few-shot / voice corrections 📋 P3–P5.

```sql
CREATE TABLE chat_intent_stats (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  intent_key      VARCHAR(64)  NOT NULL COMMENT 'maintenance, metrics, set_maintenance, ...',
  pattern_hash    CHAR(24)     NOT NULL,
  sample_message  VARCHAR(512) NOT NULL,
  hit_count       INT UNSIGNED NOT NULL DEFAULT 1,
  route_key       VARCHAR(64)  NULL COMMENT 'route thực tế — §7.6.5.5 P2',
  success_count   INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '§7.6.5.5 P2',
  fail_count      INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '§7.6.5.5 P2',
  last_seen_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_chat_intent_pattern (intent_key, pattern_hash),
  KEY idx_chat_intent_hits (intent_key, hit_count DESC, last_seen_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_sessions (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  session_uuid    CHAR(36)     NULL COMMENT 'UUID client — map session_id POST /chat',
  user_id         VARCHAR(64)  NOT NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_chat_sessions_user (user_id, updated_at),
  KEY idx_chat_sessions_uuid (session_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_messages (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  session_id      BIGINT UNSIGNED NOT NULL,
  role            ENUM('user','assistant') NOT NULL,
  content         TEXT         NOT NULL,
  input_source    ENUM('text','voice') NOT NULL DEFAULT 'text',
  stt_raw         VARCHAR(1024) NULL COMMENT 'transcript STT gốc (voice)',
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_chat_messages_session (session_id, created_at),
  CONSTRAINT fk_chat_messages_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- §7.6.5.5 P1–P2 ✅ (P3+ bảng dưới: candidate patterns, feedback, few-shot)
CREATE TABLE chat_interaction_log (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  session_uuid    CHAR(36)     NOT NULL,
  user_id         VARCHAR(64)  NULL,
  user_message    VARCHAR(1024) NOT NULL,
  message_norm    VARCHAR(1024) NOT NULL COMMENT 'CommandKey / normalize',
  input_source    ENUM('text','voice') NOT NULL DEFAULT 'text',
  stt_raw         VARCHAR(1024) NULL,
  route           VARCHAR(64)  NOT NULL COMMENT 'command_set_maintenance, direct_maintenance, llm, ...',
  intent_key      VARCHAR(64)  NULL,
  slots_json      JSON         NULL COMMENT 'product, sku, provider, ...',
  tools_called    JSON         NULL,
  action_result   ENUM('success','error','no_op','wrong_route') NOT NULL DEFAULT 'no_op',
  reply_preview   VARCHAR(512) NULL,
  latency_ms      INT UNSIGNED NULL,
  is_admin        TINYINT(1)   NOT NULL DEFAULT 0,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_chat_log_session (session_uuid, created_at),
  KEY idx_chat_log_route (route, action_result, created_at),
  KEY idx_chat_log_norm (message_norm(120), created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_command_patterns (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  command_key     VARCHAR(64)  NOT NULL COMMENT 'set_maintenance, approve, ...',
  pattern_type    ENUM('regex','keywords') NOT NULL DEFAULT 'keywords',
  pattern_def     JSON         NOT NULL,
  default_slots   JSON         NULL,
  hit_count       INT UNSIGNED NOT NULL DEFAULT 0,
  success_count   INT UNSIGNED NOT NULL DEFAULT 0,
  fail_count      INT UNSIGNED NOT NULL DEFAULT 0,
  status          ENUM('candidate','approved','deprecated') NOT NULL DEFAULT 'candidate',
  min_role        ENUM('ops','admin') NOT NULL DEFAULT 'ops',
  approved_by     VARCHAR(64)  NULL,
  approved_at     DATETIME     NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_cmd_pattern (command_key, status, hit_count DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_feedback (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  interaction_id  BIGINT UNSIGNED NOT NULL,
  rating          ENUM('up','down','corrected') NOT NULL,
  user_correction VARCHAR(1024) NULL,
  expected_command VARCHAR(64) NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_feedback_interaction (interaction_id),
  CONSTRAINT fk_chat_feedback_log FOREIGN KEY (interaction_id) REFERENCES chat_interaction_log(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_few_shot_examples (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  command_key     VARCHAR(64)  NOT NULL,
  user_example    VARCHAR(512) NOT NULL,
  assistant_example VARCHAR(1024) NOT NULL,
  success_rate    DECIMAL(5,2) NULL,
  priority        INT          NOT NULL DEFAULT 0,
  status          ENUM('candidate','approved','deprecated') NOT NULL DEFAULT 'candidate',
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_few_shot (command_key, status, priority DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_voice_corrections (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  heard_norm      VARCHAR(512) NOT NULL,
  corrected_norm  VARCHAR(512) NOT NULL,
  hit_count       INT UNSIGNED NOT NULL DEFAULT 1,
  last_seen_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_voice_correction (heard_norm(120), corrected_norm(120)),
  KEY idx_voice_hits (hit_count DESC, last_seen_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_user_prefs (
  user_id         VARCHAR(64)  NOT NULL PRIMARY KEY,
  display_name    VARCHAR(64)  NULL,
  honorific       VARCHAR(16)  NULL,
  preferred_brevity ENUM('short','normal') NOT NULL DEFAULT 'normal',
  favorite_products JSON       NULL,
  updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 13.9.1 Users — cache profile O365/Entra ID (§2.6.9)

```sql
CREATE TABLE users (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  azure_oid     VARCHAR(64)  NOT NULL COMMENT 'claim oid — stable user id từ Entra ID',
  upn           VARCHAR(128) NOT NULL COMMENT 'preferred_username — user@company.com',
  display_name  VARCHAR(128) NULL,
  role_cached   ENUM('admin','ops') NOT NULL DEFAULT 'ops' COMMENT 'role gần nhất (cache hiển thị; verify thật vẫn từ JWT mỗi request)',
  last_login_at DATETIME     NULL,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_users_oid (azure_oid),
  UNIQUE KEY uk_users_upn (upn),
  KEY idx_users_role (role_cached)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Query upsert trong middleware (sau verify JWT thành công):**

```sql
INSERT INTO users (azure_oid, upn, display_name, role_cached, last_login_at)
VALUES (?, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
  upn = VALUES(upn),
  display_name = VALUES(display_name),
  role_cached = VALUES(role_cached),
  last_login_at = NOW();
```

### 13.10 Seed mẫu (rút gọn)

```sql
INSERT INTO agent_settings (id, notification_recipients, agent_locale) VALUES (1, '["ops-team@company.com"]', 'vi-VN');

INSERT INTO providers (provider_code, label) VALUES
  ('ESALE','ESALE'), ('IMEDIA','IMEDIA'), ('SHOPPAY','SHOPPAY');

INSERT INTO products (product_code, label, service_type, routing_mode) VALUES
  ('ZING','Thẻ Zing','card','sku'),
  ('GARENA','Thẻ Garena','card','sku'),
  ('VINAPHONE','Thẻ Vinaphone','card','sku'),
  ('MOBIFONE','Thẻ Mobifone','card','sku'),
  ('VIETTEL','Thẻ Viettel','card','sku'),
  ('DATA_VINA','Data Vinaphone','topup_data','sku'),
  ('DATA_MOBI','Data Mobifone','topup_data','sku'),
  ('DATA_VIETTEL','Data Viettel','topup_data','sku'),
  ('TOPUP_VINA','Topup Vinaphone','topup','provider'),
  ('TOPUP_MOBI','Topup Mobifone','topup','provider'),
  ('TOPUP_VIETTEL','Topup Viettel','topup','provider');

-- product_skus: thẻ mệnh giá + topup data VNP20/VNP50/V50K/V100K — seed theo product_id
-- product_providers: mỗi product × 3 provider, enabled=1 (seed mặc định — §6.3)

-- routing_config — BUSINESS BASELINE (§8.6 Pha A). Khi seed: traffic_pct = baseline_pct.
-- Topup tiền (provider-level, sku_code=''):
INSERT INTO routing_config (product_code, sku_code, provider_code, baseline_pct, traffic_pct, baseline_updated_by, updated_by) VALUES
  ('TOPUP_VINA',    '', 'ESALE',   70.00, 70.00, 'business', 'business'),
  ('TOPUP_VINA',    '', 'IMEDIA',  20.00, 20.00, 'business', 'business'),
  ('TOPUP_VINA',    '', 'SHOPPAY', 10.00, 10.00, 'business', 'business'),
  ('TOPUP_MOBI',    '', 'ESALE',   60.00, 60.00, 'business', 'business'),
  ('TOPUP_MOBI',    '', 'IMEDIA',  30.00, 30.00, 'business', 'business'),
  ('TOPUP_MOBI',    '', 'SHOPPAY', 10.00, 10.00, 'business', 'business'),
  ('TOPUP_VIETTEL', '', 'ESALE',   50.00, 50.00, 'business', 'business'),
  ('TOPUP_VIETTEL', '', 'IMEDIA',  30.00, 30.00, 'business', 'business'),
  ('TOPUP_VIETTEL', '', 'SHOPPAY', 20.00, 20.00, 'business', 'business');

-- Thẻ / Topup data: per-SKU baseline. Ví dụ ZING SKU 20000:
INSERT INTO routing_config (product_code, sku_code, provider_code, baseline_pct, traffic_pct, baseline_updated_by, updated_by) VALUES
  ('ZING', '20000', 'ESALE',   80.00, 80.00, 'business', 'business'),
  ('ZING', '20000', 'IMEDIA',  20.00, 20.00, 'business', 'business'),
  ('DATA_VINA', 'VNP50', 'ESALE',  70.00, 70.00, 'business', 'business'),
  ('DATA_VINA', 'VNP50', 'IMEDIA', 30.00, 30.00, 'business', 'business');
-- (Seed đầy đủ các SKU còn lại trong db/seed.sql)

-- Kịch bản B (ZING chỉ ESALE): chạy SQL thu hẹp provider — xem §10.2

INSERT INTO provider_chat_escalation (provider_code, chat_app_name, chat_group_name, mention_tags) VALUES
  ('ESALE', 'Microsoft Teams', '[OpsOne] ESALE Support', '@esale-oncall'),
  ('IMEDIA', 'Microsoft Teams', '[OpsOne] IMEDIA Support', '@imedia-ops'),
  ('SHOPPAY', 'Zalo', 'OpsOne x SHOPPAY', '@shopay-support @n2-lead');

INSERT INTO product_alert_thresholds (
  product_code, success_rate_min_pct, pending_rate_max_pct, fail_rate_max_pct,
  fail_txn_count_max, error_event_count_max, metrics_window_min, consecutive_cycles_required
)
SELECT product_code, 80.00, 15.00, 10.00, 100, 50, 15, 2 FROM products;
```

### 13.11 Checklist DB

- Schema trong repo: `db/schema.sql`
- Seed trong repo: `db/seed.sql` — **UTF-8**, nhãn tiếng Việt đọc được (`Thẻ Zing`, `10.000đ`, …)
- **Windows:** dùng `Invoke-OpsOneReset` (`scripts/dev.ps1`) — `docker cp` file SQL vào container + `mysql source` + `--default-character-set=utf8mb4` (không pipe stdin PowerShell — tránh `Th???`)
- MySQL Docker: `TZ=Asia/Ho_Chi_Minh`, DSN Go `loc=Asia/Ho_Chi_Minh` (§2.1)
- `make db-up && make migrate && make seed` — verify `products` = 11
- Job retention: `DELETE FROM mock_metrics WHERE recorded_at < NOW() - INTERVAL 24 HOUR` (theo `mock_retention_hours`)
- User app MySQL least privilege (SELECT/INSERT/UPDATE)
- Index `idx_agent_changes_status` — truy vấn audit theo product/scope/status

---

# Part VII — Tool Contracts (AI Codegen / MCP-ready)

> **Mục đích:** Định nghĩa **JSON Schema** đầy đủ cho 9 tool ở §6 để:
> 1. AI codegen tự sinh Go struct + handler không phải đoán.
> 2. Compatible với **Model Context Protocol (MCP)** — đăng ký tool cho LLM dùng trực tiếp.
> 3. Test contract: `internal/tools/*_test.go` validate request/response qua schema.
>
> File triển khai: `internal/tools/contracts.json` (1 file gom 9 tool) + Go binding `internal/tools/contracts.go`.

## 14. Tool Contracts — JSON Schema 9 tools

### 14.1 Quy ước chung

- **`product`**: enum 11 product code (xem §1.1) — `ZING | GARENA | VINAPHONE | MOBIFONE | VIETTEL | DATA_VINA | DATA_MOBI | DATA_VIETTEL | TOPUP_VINA | TOPUP_MOBI | TOPUP_VIETTEL`.
- **`provider`**: enum `ESALE | IMEDIA | SHOPPAY`.
- **`sku`**: string; bắt buộc khi `routing_mode=sku` (card, topup_data); rỗng `""` khi `routing_mode=provider` (topup).
- **`window`**: ISO 8601 duration (`PT15M`, `PT1H`) hoặc dạng tắt `"15m"`, `"1h"`. Mặc định `15m`.
- **Tất cả response** kèm `meta`: `{ "data_source": "mock|production", "queried_at": ISO8601 }`.
- **Error envelope**: `{ "error": { "code": "string", "message_vi": "string" } }` HTTP 4xx/5xx.

### 14.2 Tool registry (overview)

```yaml
tools:
  - name: GetMetrics       # §6.1 — đọc only
  - name: GetTopErrors     # §6.2 — đọc only
  - name: GetProviders     # §6.3 — đọc only
  - name: GetSkus          # §6.4 — đọc only
  - name: GetRouting       # §6.5 — đọc only
  - name: GetRevenue       # §6.6 — đọc only
  - name: UpdateRouting    # §6.7 — MUTATING (Admin auth + audit log)
  - name: GetMaintenance   # §6.8 — đọc only
  - name: SetMaintenance   # §6.9 — MUTATING (Admin auth + audit log)
```

### 14.3 GetMetrics

```json
{
  "name": "GetMetrics",
  "description": "Lấy success/pending/fail rate + tổng GD trong cửa sổ.",
  "input_schema": {
    "type": "object",
    "required": ["product", "provider"],
    "properties": {
      "product":  { "type": "string", "enum": ["ZING","GARENA","VINAPHONE","MOBIFONE","VIETTEL","DATA_VINA","DATA_MOBI","DATA_VIETTEL","TOPUP_VINA","TOPUP_MOBI","TOPUP_VIETTEL"] },
      "provider": { "type": "string", "enum": ["ESALE","IMEDIA","SHOPPAY"] },
      "sku":      { "type": "string", "default": "", "description": "rỗng khi routing_mode=provider" },
      "window":   { "type": "string", "pattern": "^[0-9]+[mh]$", "default": "15m" }
    }
  },
  "output_schema": {
    "type": "object",
    "required": ["success_rate","pending_rate","fail_rate","total_transactions","meta"],
    "properties": {
      "success_rate":       { "type": "number", "minimum": 0, "maximum": 100 },
      "pending_rate":       { "type": "number", "minimum": 0, "maximum": 100 },
      "fail_rate":          { "type": "number", "minimum": 0, "maximum": 100 },
      "total_transactions": { "type": "integer", "minimum": 0 },
      "fail_txn_count":     { "type": "integer", "minimum": 0, "description": "= total * fail_rate / 100 (§7.4)" },
      "meta": { "$ref": "#/definitions/Meta" }
    }
  }
}
```

### 14.3.1 GetTopErrors

```json
{
  "name": "GetTopErrors",
  "input_schema": {
    "type": "object",
    "required": ["product","provider"],
    "properties": {
      "product":  { "type": "string" },
      "provider": { "type": "string" },
      "sku":      { "type": "string", "default": "" },
      "window":   { "type": "string", "default": "15m" },
      "top_n":    { "type": "integer", "default": 5, "minimum": 1, "maximum": 20 }
    }
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "errors": {
        "type": "array",
        "items": {
          "type": "object",
          "required": ["code","count"],
          "properties": {
            "code":  { "type": "string", "example": "-3004" },
            "count": { "type": "integer" },
            "label_vi": { "type": "string", "description": "Diễn giải tiếng Việt (tuỳ chọn)" }
          }
        }
      },
      "error_event_count_total": { "type": "integer", "description": "SUM count — dùng cho §1.2 error_event_count_max" },
      "meta": { "$ref": "#/definitions/Meta" }
    }
  }
}
```

### 14.3.2 GetProviders

```json
{
  "name": "GetProviders",
  "description": "Trả active/inactive providers cho product. §6.3 §7.4.",
  "input_schema": {
    "type": "object",
    "required": ["product"],
    "properties": { "product": { "type": "string" } }
  },
  "output_schema": {
    "type": "object",
    "required": ["product","active_providers","inactive_providers","active_count","total_count"],
    "properties": {
      "product":            { "type": "string" },
      "active_providers":   { "type": "array", "items": { "type": "string" } },
      "inactive_providers": { "type": "array", "items": { "type": "string" } },
      "active_count":       { "type": "integer" },
      "total_count":        { "type": "integer" },
      "meta": { "$ref": "#/definitions/Meta" }
    }
  }
}
```

### 14.3.3 GetSkus

```json
{
  "name": "GetSkus",
  "description": "Danh sách SKU — chỉ cho routing_mode=sku (card, topup_data).",
  "input_schema": {
    "type": "object",
    "required": ["product"],
    "properties": { "product": { "type": "string" } }
  },
  "output_schema": {
    "type": "object",
    "required": ["product","service_type","skus"],
    "properties": {
      "product":      { "type": "string" },
      "service_type": { "type": "string", "enum": ["card","topup_data","topup"] },
      "skus": {
        "type": "array",
        "items": {
          "type": "object",
          "required": ["sku","label","enabled"],
          "properties": {
            "sku":     { "type": "string" },
            "label":   { "type": "string" },
            "enabled": { "type": "boolean" }
          }
        }
      },
      "meta": { "$ref": "#/definitions/Meta" }
    },
    "errors": {
      "TOPUP_NOT_APPLICABLE": "Gọi GetSkus với product service_type=topup → trả 422"
    }
  }
}
```

### 14.3.4 GetRouting

```json
{
  "name": "GetRouting",
  "input_schema": {
    "type": "object",
    "required": ["product"],
    "properties": {
      "product": { "type": "string" },
      "sku":     { "type": "string", "default": "" }
    }
  },
  "output_schema": {
    "oneOf": [
      {
        "type": "object",
        "description": "scope=provider (topup)",
        "required": ["product","service_type","scope","routing"],
        "properties": {
          "product":      { "type": "string" },
          "service_type": { "type": "string", "enum": ["topup"] },
          "scope":        { "type": "string", "enum": ["provider"] },
          "baseline":     { "type": "object", "additionalProperties": { "type": "number" } },
          "routing":      { "type": "object", "additionalProperties": { "type": "number" } },
          "pending_restore": { "type": "boolean" },
          "meta": { "$ref": "#/definitions/Meta" }
        }
      },
      {
        "type": "object",
        "description": "scope=sku (card, topup_data)",
        "required": ["product","service_type","scope","routing_by_sku"],
        "properties": {
          "product":      { "type": "string" },
          "service_type": { "type": "string", "enum": ["card","topup_data"] },
          "scope":        { "type": "string", "enum": ["sku"] },
          "routing_by_sku": {
            "type": "object",
            "additionalProperties": {
              "type": "object",
              "properties": {
                "baseline":        { "type": "object", "additionalProperties": { "type": "number" } },
                "routing":         { "type": "object", "additionalProperties": { "type": "number" } },
                "pending_restore": { "type": "boolean" }
              }
            }
          },
          "meta": { "$ref": "#/definitions/Meta" }
        }
      }
    ]
  }
}
```

### 14.3.5 GetRevenue

```json
{
  "name": "GetRevenue",
  "input_schema": {
    "type": "object",
    "required": ["product","provider"],
    "properties": {
      "product":  { "type": "string" },
      "provider": { "type": "string" },
      "sku":      { "type": "string", "default": "" },
      "window":   { "type": "string", "default": "1h" }
    }
  },
  "output_schema": {
    "type": "object",
    "required": ["product","provider","revenue_amount","currency"],
    "properties": {
      "product":         { "type": "string" },
      "provider":        { "type": "string" },
      "sku":             { "type": "string" },
      "revenue_amount":  { "type": "integer", "description": "VD VND" },
      "revenue_last_hour": { "type": "integer", "deprecated": true, "description": "alias revenue_amount window=1h" },
      "currency":        { "type": "string", "default": "VND" },
      "window":          { "type": "string" },
      "meta": { "$ref": "#/definitions/Meta" }
    }
  }
}
```

### 14.3.6 UpdateRouting (MUTATING)

```json
{
  "name": "UpdateRouting",
  "description": "Đổi traffic_pct. Validate scope theo service_type. Ghi agent_change_log §8.7.",
  "auth": { "role": "Admin", "audit_log": true },
  "input_schema": {
    "type": "object",
    "required": ["product","service_type","scope","reason"],
    "properties": {
      "product":      { "type": "string" },
      "service_type": { "type": "string", "enum": ["card","topup_data","topup"] },
      "scope":        { "type": "string", "enum": ["provider","sku","sku_batch"] },
      "sku":          { "type": "string" },
      "set_as_baseline": { "type": "boolean", "default": false, "description": "true=baseline mới (Business); false=temp override §8.6.5" },
      "routing": {
        "type": "object",
        "additionalProperties": { "type": "number", "minimum": 0, "maximum": 100 },
        "description": "Map provider→% — Σ=100"
      },
      "updates": {
        "type": "array",
        "description": "Chỉ dùng khi scope=sku_batch",
        "items": {
          "type": "object",
          "required": ["sku","routing"],
          "properties": {
            "sku":     { "type": "string" },
            "routing": { "type": "object", "additionalProperties": { "type": "number" } }
          }
        }
      },
      "reason":        { "type": "string", "minLength": 5 },
      "trigger_type":  { "type": "string", "enum": ["auto","admin_approve","manual_baseline","manual_temp"] },
      "cycle_id":      { "type": "integer" },
      "routing_plan_id": { "type": "integer" }
    },
    "constraints": [
      "service_type=topup → scope MUST = provider",
      "service_type IN [card,topup_data] → scope MUST IN [sku,sku_batch]",
      "Σ(routing.values) == 100 per scope/sku",
      "each value in [0, 100]"
    ]
  },
  "output_schema": {
    "type": "object",
    "required": ["change_id","product","routing_before","routing_after"],
    "properties": {
      "change_id":      { "type": "integer", "description": "agent_change_log.id" },
      "product":        { "type": "string" },
      "scope":          { "type": "string" },
      "sku":            { "type": "string" },
      "routing_before": { "type": "object" },
      "routing_after":  { "type": "object" },
      "applied_at":     { "type": "string", "format": "date-time" },
      "executed_by":    { "type": "string", "description": "opsone-agent | UPN admin" },
      "meta": { "$ref": "#/definitions/Meta" }
    }
  },
  "errors": {
    "SCOPE_MISMATCH":          "scope không khớp service_type",
    "SUM_NOT_100":             "Σ routing ≠ 100",
    "GUARDRAIL_RANGE":         "value not in [0, 100]",
    "MAINTENANCE_ACTIVE":      "Đang có maintenance_windows active — không UpdateRouting",
    "PERMISSION_DENIED":       "Caller không có role Admin"
  }
}
```

### 14.3.7 GetMaintenance

```json
{
  "name": "GetMaintenance",
  "input_schema": {
    "type": "object",
    "required": ["product"],
    "properties": {
      "product":  { "type": "string" },
      "provider": { "type": "string" },
      "sku":      { "type": "string", "default": "" }
    }
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "active":    { "type": "array", "items": { "$ref": "#/definitions/MaintenanceWindow" } },
      "scheduled": { "type": "array", "items": { "$ref": "#/definitions/MaintenanceWindow" } },
      "meta": { "$ref": "#/definitions/Meta" }
    },
    "definitions": {
      "MaintenanceWindow": {
        "type": "object",
        "required": ["maintenance_id","starts_at","ends_at","status"],
        "properties": {
          "maintenance_id":     { "type": "string" },
          "starts_at":          { "type": "string", "format": "date-time" },
          "ends_at":            { "type": "string", "format": "date-time" },
          "status":             { "type": "string", "enum": ["pending_approve","scheduled","active","completed","cancelled"] },
          "remaining_minutes":  { "type": "integer" },
          "reason":             { "type": "string" }
        }
      }
    }
  }
}
```

### 14.3.8 SetMaintenance (MUTATING)

```json
{
  "name": "SetMaintenance",
  "auth": { "role": "Admin", "audit_log": true },
  "input_schema": {
    "type": "object",
    "required": ["product","provider","starts_at","ends_at","reason","trigger_type"],
    "properties": {
      "product":      { "type": "string" },
      "provider":     { "type": "string" },
      "sku":          { "type": "string", "default": "" },
      "starts_at":    { "type": "string", "format": "date-time" },
      "ends_at":      { "type": "string", "format": "date-time" },
      "trigger_type": { "type": "string", "enum": ["opsone_recommend","opsone_auto","admin_manual"] },
      "reason":       { "type": "string", "minLength": 5 },
      "cycle_id":     { "type": "integer" },
      "incident_id":  { "type": "string" }
    },
    "constraints": [
      "ends_at > starts_at",
      "Không overlap maintenance scheduled/active cùng (product,provider,sku)"
    ]
  },
  "output_schema": {
    "type": "object",
    "required": ["maintenance_id","status","starts_at","ends_at"],
    "properties": {
      "maintenance_id": { "type": "string", "example": "20260604-M001" },
      "status":         { "type": "string", "enum": ["pending_approve","scheduled","active"] },
      "starts_at":      { "type": "string", "format": "date-time" },
      "ends_at":        { "type": "string", "format": "date-time" },
      "meta": { "$ref": "#/definitions/Meta" }
    }
  },
  "errors": {
    "INVALID_DURATION": "ends_at <= starts_at",
    "OVERLAP_WINDOW":   "Đã có maintenance scheduled/active overlap",
    "PERMISSION_DENIED": "Caller không có role Admin"
  }
}
```

### 14.4 Shared definitions

```json
{
  "definitions": {
    "Meta": {
      "type": "object",
      "required": ["data_source","queried_at"],
      "properties": {
        "data_source": { "type": "string", "enum": ["mock","production"] },
        "queried_at":  { "type": "string", "format": "date-time" },
        "window":      { "type": "string" }
      }
    }
  }
}
```

### 14.5 Đăng ký tool cho LLM (OpenAI-compatible function calling)

```ts
// Auto-generate từ 14.x — internal/reasoning/tools_registry.go (Go) / FE TypeScript
export const TOOLS_OPENAI = [
  {
    type: "function",
    function: {
      name: "GetMetrics",
      description: "Lấy success/pending/fail rate + tổng GD",
      parameters: /* input_schema từ 14.3 */
    }
  },
  // ... 8 tool còn lại
];
```

**Lưu ý cho chat agent (§7.6.5):** Thứ tự routing: metric (§7.6.5.2) → **lệnh trực tiếp** (§7.6.5.4) → tra cứu BT (§7.6.5.1) → LLM. Phân biệt *"bảo trì giúp tôi"* (lệnh) vs *"có đang bảo trì không"* (tra cứu). Intent ghi `chat_intent_stats` (§7.6.5.3); mỗi lượt persist `chat_interaction_log` + `chat_messages` (§7.6.5.5 P1–P2). Args qua `chatresolve.NormalizeToolArgs`. Admin: tool duyệt + `set_maintenance` / `reopen_service` / `set_scope_auto`.

### 14.6 Checklist Tool Contracts

- [ ] File `internal/tools/contracts.json` chứa 9 tool đầy đủ (theo 14.3.x).
- [ ] Generator `make gen-tools` — đọc JSON → sinh Go struct (`*Request`, `*Response`) + handler skeleton.
- [ ] Unit test mỗi tool: input invalid → trả error envelope đúng `errors`.
- [ ] Validator middleware: validate request body theo JSON Schema trước khi vào handler.
- [ ] Doc auto-gen: `make gen-tool-docs` → README per tool (Markdown từ JSON).
- [ ] Tool registry expose qua endpoint `GET /api/v1/tools` (cho debug / MCP discovery).

---

# Part VIII — Deployment

> **Mục tiêu:** Một AI codegen / DevOps có thể đọc Part này → tự sinh `docker-compose.yml`, `Dockerfile`, GitHub Actions, hoặc deploy lên Vercel + Railway + Aiven mà không phải đoán.

## 15. Deployment

### 15.1 Topology

```text
┌────────────────────────────────────────────────────────────────┐
│                       Internet / Ops users                     │
└─────────────────┬──────────────────────────────┬───────────────┘
                  │ HTTPS (443)                  │ HTTPS
                  ▼                              ▼
         ┌────────────────┐              ┌────────────────┐
         │  Frontend SPA  │              │   API Gateway  │
         │  (React build  │              │   (Nginx /     │
         │   static)      │              │    Traefik)    │
         └────────────────┘              └────────┬───────┘
                                                  │ /api/v1/*
                                                  ▼
                       ┌──────────────────────────────────────┐
                       │       Backend services (Go)          │
                       │  cmd/api   cmd/worker-mock           │
                       │            cmd/worker-agent          │
                       └────────────────┬─────────────────────┘
                                        │ TCP 3306
                                        ▼
                                ┌─────────────┐
                                │  MySQL 8    │
                                └─────────────┘
                                        ▲
                                        │
                  ┌────────────────────────────────────────┐
                  │ External: SMTP (MailHog dev / SES prod),│
                  │           OpenAI-compat LLM API,        │
                  │           Microsoft Entra ID            │
                  └────────────────────────────────────────┘
```

### 15.2 Ba đường deploy chính

#### 15.2.1 Option A — Self-hosted (Docker Compose / VPS / On-prem)

Cho hackathon demo và môi trường nội bộ. **Khuyến nghị cho phiên bản hiện tại.**

```yaml
# deployment:
#   target: docker-compose
#   db:       mysql:8.0
#   backend:  alpine + go binary
#   frontend: nginx:alpine serving static React build
#   reverse_proxy: nginx (cùng container frontend) hoặc traefik bên ngoài
#   smtp:     mailhog (dev) | external SMTP (prod)
```

**`docker-compose.yml`** (file `infra/docker-compose.yml`):

```yaml
version: "3.9"
services:
  mysql:
    image: mysql:8.0.39
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: opsone
      MYSQL_USER: opsone
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
    volumes:
      - mysql_data:/var/lib/mysql
      - ../db/schema.sql:/docker-entrypoint-initdb.d/01-schema.sql:ro
      - ../db/seed.sql:/docker-entrypoint-initdb.d/02-seed.sql:ro
    ports: ["3306:3306"]
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      retries: 5

  api:
    build: { context: .., dockerfile: infra/Dockerfile.go, args: { BINARY: api } }
    environment:
      MYSQL_DSN: opsone:${MYSQL_PASSWORD}@tcp(mysql:3306)/opsone?parseTime=true
      API_ADDR: ":8080"
      AAD_TENANT_ID: ${AAD_TENANT_ID}
      AAD_API_CLIENT_ID: ${AAD_API_CLIENT_ID}
      AAD_ISSUER: https://login.microsoftonline.com/${AAD_TENANT_ID}/v2.0
      SMTP_HOST: mailhog
      SMTP_PORT: "1025"
      LLM_API_URL: ${LLM_API_URL}
      LLM_API_KEY: ${LLM_API_KEY}
    depends_on:
      mysql: { condition: service_healthy }
    ports: ["8080:8080"]

  worker-mock:
    build: { context: .., dockerfile: infra/Dockerfile.go, args: { BINARY: worker-mock } }
    environment:
      MYSQL_DSN: opsone:${MYSQL_PASSWORD}@tcp(mysql:3306)/opsone?parseTime=true
    depends_on:
      mysql: { condition: service_healthy }

  worker-agent:
    build: { context: .., dockerfile: infra/Dockerfile.go, args: { BINARY: worker-agent } }
    environment:
      MYSQL_DSN: opsone:${MYSQL_PASSWORD}@tcp(mysql:3306)/opsone?parseTime=true
      LLM_API_URL: ${LLM_API_URL}
      LLM_API_KEY: ${LLM_API_KEY}
    depends_on:
      mysql: { condition: service_healthy }

  web:
    build: { context: ../web, dockerfile: ../infra/Dockerfile.web }
    ports: ["80:80"]
    depends_on: [api]

  mailhog:
    image: mailhog/mailhog:v1.0.1
    ports: ["1025:1025", "8025:8025"]   # SMTP, Web UI

volumes:
  mysql_data:
```

**`infra/Dockerfile.go`** (multi-stage, 1 file cho 3 binary):

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.21-alpine AS builder
ARG BINARY
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${BINARY}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
```

**`infra/Dockerfile.web`** (Nginx serving React build):

```dockerfile
FROM node:20.13-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY ../infra/nginx.conf /etc/nginx/conf.d/default.conf
HEALTHCHECK CMD wget -q -O - http://localhost/ >/dev/null || exit 1
```

**`infra/nginx.conf`** (reverse proxy + SPA fallback):

```nginx
upstream opsone_api { server api:8080; }

server {
  listen 80;
  server_name _;
  root /usr/share/nginx/html;
  index index.html;

  # SPA fallback
  location / {
    try_files $uri $uri/ /index.html;
  }

  # API proxy
  location /api/ {
    proxy_pass http://opsone_api;
    proxy_http_version 1.1;
    proxy_set_header X-Forwarded-For $remote_addr;
    proxy_set_header Authorization $http_authorization;
  }

  # SSE
  location /api/v1/events {
    proxy_pass http://opsone_api;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 24h;
  }
}
```

**Lệnh chạy:**

```bash
cd infra
cp .env.example .env       # điền secrets
docker compose up -d --build
docker compose logs -f api worker-agent
```

#### 15.2.2 Option B — GreenNode AgentBase (VNG Cloud) ✅ triển khai hiện tại

**4 Custom Agent runtimes** (PUBLIC), mỗi runtime = 1 Docker image + flavor (`runtime-s2-general-2x4`):

| Runtime | Image | Mô tả | URL người dùng |
|---------|-------|--------|----------------|
| `opsone-api` | `Dockerfile` | REST + SSE + chat LLM `:8080` | DEFAULT endpoint — API backend |
| `opsone-worker-mock` | `Dockerfile.worker-mock` | Mock tick 1 phút | Chỉ `GET /health` |
| `opsone-worker-agent` | `Dockerfile.worker-agent` | Scheduler 5 phút | Chỉ `GET /health` |
| `opsone-web` | `web/Dockerfile` | React SPA (nginx `:8080`) | DEFAULT endpoint — **mở dashboard** |

**Hai lớp env (bắt buộc tách):**

| File | Mục đích |
|------|----------|
| `opsone/.env` (repo root) | `GREENNODE_CLIENT_ID` / `GREENNODE_CLIENT_SECRET` — IAM deploy từ máy dev |
| `ai_gen_src/.env.greennode` | Env **trong container** api/workers: `MYSQL_DSN`, `LLM_*`, `CORS_ORIGIN=*`, `DEV_AUTH_BYPASS=true`, … |

> **MySQL:** GreenNode **không** cấp DB. Dùng vDB MySQL (public endpoint) hoặc MySQL khác reachable từ runtime PUBLIC.  
> **`MYSQL_DSN` (Go):** **không** thêm `allowPublicKeyRetrieval=true` (chỉ dùng cho DBeaver/JDBC — Go driver lỗi 1193). `config.sanitizeMySQLDSN` tự strip nếu copy nhầm.  
> Runtime tự inject `GREENNODE_CLIENT_ID`, `GREENNODE_AGENT_IDENTITY`, … — **không** ghi trong `.env.greennode`.

**Deploy (PowerShell, từ `ai_gen_src`):**

```powershell
copy ..\.env.example ..\.env                    # IAM
copy .env.greennode.example .env.greennode      # MYSQL_DSN + LLM
.\scripts\prepare-greennode-env.ps1             # tuỳ chọn: từ .env local

.\scripts\deploy-greennode.ps1 -Target api
.\scripts\deploy-greennode.ps1 -Target worker-mock
.\scripts\deploy-greennode.ps1 -Target worker-agent
.\scripts\deploy-greennode.ps1 -Target web      # cần opsone-api ACTIVE; auto lấy API endpoint

# hoặc một lệnh:
.\scripts\deploy-greennode-all.ps1
```

**Web build:** `VITE_API_BASE_URL=https://<opsone-api-endpoint>/api/v1` (build-arg lúc `docker build`); `VITE_DEV_AUTH_BYPASS=true` trên GreenNode demo. Local dev: proxy Vite `/api` → `localhost:8080` (để trống `VITE_API_BASE_URL`).

**Health:** mọi image lắng nghe `:8080`, `GET /health` → `200 {"status":"ok"}`. API/workers: health server trước, `OpenWithRetry` MySQL tối đa 2 phút.

**Console:** [Agent Runtime](https://aiplatform.console.vngcloud.vn/agent-runtime?tab=runtime) · script in `Endpoint:` / `Dashboard:` sau deploy.

**Bash:** `TARGET=api|worker-mock|worker-agent|web bash scripts/deploy-greennode.sh`

#### 15.2.3 Option C — Managed cloud (Vercel + Railway + Aiven)

Cho production rollout nhẹ, không tự quản hạ tầng. **Tách 3 layer ra 3 nhà cung cấp.**

```yaml
# deployment:
#   target: managed-cloud
#   frontend: Vercel
#   backend:  Railway (api + worker-mock + worker-agent — 3 services)
#   db:       Aiven MySQL
#   smtp:     SendGrid / AWS SES
#   llm:      OpenAI / Azure OpenAI
```

| Layer | Service | Cấu hình |
|-------|---------|----------|
| **Frontend** | **Vercel** | Connect Git repo `web/`; framework: Vite; build cmd `npm run build`; output `dist`; ENV: `VITE_AAD_TENANT_ID`, `VITE_AAD_WEB_CLIENT_ID`, `VITE_API_BASE_URL=https://opsone-api.up.railway.app` |
| **Backend API** | **Railway** service `opsone-api` | Dockerfile `infra/Dockerfile.go` ARG `BINARY=api`; expose 8080; ENV đầy đủ từ §15.5 |
| **Worker mock** | **Railway** service `opsone-worker-mock` | Cùng image; ARG `BINARY=worker-mock`; chỉ ENV MySQL |
| **Worker agent** | **Railway** service `opsone-worker-agent` | Cùng image; ARG `BINARY=worker-agent`; ENV MySQL + LLM |
| **Database** | **Aiven MySQL** plan startup | MySQL 8; copy `DATABASE_URL` vào 3 Railway services |
| **SMTP** | **SendGrid** (hoặc AWS SES) | API key → ENV `SMTP_*` |
| **LLM (chat)** | **GreenNode AIP** (MaaS OpenAI-compatible) | `LLM_API_KEY`, `LLM_MODEL=minimax/minimax-m2.5` (hoặc id từ `/v1/models`) |

**Railway `railway.json`** (tuỳ chọn — IaC):

```json
{
  "$schema": "https://railway.app/railway.schema.json",
  "services": [
    { "name": "api",          "dockerfile": "infra/Dockerfile.go", "build": { "buildArgs": { "BINARY": "api" } },          "tcpProxyPort": 8080 },
    { "name": "worker-mock",  "dockerfile": "infra/Dockerfile.go", "build": { "buildArgs": { "BINARY": "worker-mock" } } },
    { "name": "worker-agent", "dockerfile": "infra/Dockerfile.go", "build": { "buildArgs": { "BINARY": "worker-agent" } } }
  ]
}
```

### 15.3 CI/CD — GitHub Actions

File: `.github/workflows/ci.yml`

```yaml
name: CI
on:
  push:    { branches: [main, develop] }
  pull_request:

jobs:
  test-backend:
    runs-on: ubuntu-22.04
    services:
      mysql:
        image: mysql:8.0.39
        env:
          MYSQL_ROOT_PASSWORD: root
          MYSQL_DATABASE: opsone_test
        ports: ["3306:3306"]
        options: --health-cmd "mysqladmin ping" --health-interval 10s --health-timeout 5s --health-retries 10
    steps:
      - uses: actions/checkout@v4.1.7
      - uses: actions/setup-go@v5.0.2
        with: { go-version: "1.21.13" }
      - name: Apply schema
        run: |
          mysql -h 127.0.0.1 -uroot -proot opsone_test < db/schema.sql
          mysql -h 127.0.0.1 -uroot -proot opsone_test < db/seed.sql
      - name: Test
        env:
          MYSQL_DSN: root:root@tcp(127.0.0.1:3306)/opsone_test?parseTime=true
          DEV_AUTH_BYPASS: "true"
          LLM_ENABLED: "false"
        run: go test ./... -race -cover

  test-frontend:
    runs-on: ubuntu-22.04
    steps:
      - uses: actions/checkout@v4.1.7
      - uses: actions/setup-node@v4.0.3
        with: { node-version: "20.13.1", cache: "npm", cache-dependency-path: web/package-lock.json }
      - run: npm ci
        working-directory: web
      - run: npm run lint && npm run build
        working-directory: web

  build-docker:
    needs: [test-backend, test-frontend]
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-22.04
    steps:
      - uses: actions/checkout@v4.1.7
      - uses: docker/setup-buildx-action@v3.6.1
      - uses: docker/login-action@v3.3.0
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build & push 3 binaries
        run: |
          for bin in api worker-mock worker-agent; do
            docker buildx build --push \
              --build-arg BINARY=$bin \
              -t ghcr.io/${{ github.repository }}/opsone-$bin:${{ github.sha }} \
              -t ghcr.io/${{ github.repository }}/opsone-$bin:latest \
              -f infra/Dockerfile.go .
          done
      - name: Build & push web
        run: |
          docker buildx build --push \
            -t ghcr.io/${{ github.repository }}/opsone-web:${{ github.sha }} \
            -t ghcr.io/${{ github.repository }}/opsone-web:latest \
            -f infra/Dockerfile.web web/
```

> **Pinning version:** mọi action pin theo SHA hoặc semver `>=2 tháng` để tuân thủ rule sandbox. Các phiên bản trên (checkout@v4.1.7 — 07/2024, setup-go@v5.0.2 — 06/2024, setup-node@v4.0.3 — 07/2024, buildx@v3.6.1 — 08/2024, login@v3.3.0 — 08/2024) đều ổn định >2 tháng tại thời điểm spec.

### 15.4 Migration & seed runbook

```bash
# Local
make db-up           # docker compose up -d mysql
make migrate         # apply db/schema.sql
make seed            # apply db/seed.sql
make db-verify       # SELECT COUNT(*) FROM products == 11

# Railway / Aiven (production)
mysql "$DATABASE_URL" < db/schema.sql
mysql "$DATABASE_URL" < db/seed.sql
# Job retention (cron daily)
mysql "$DATABASE_URL" -e "DELETE FROM mock_metrics WHERE recorded_at < NOW() - INTERVAL 24 HOUR"
```

### 15.5 Bảng biến môi trường tổng (production)

| Tên | Bắt buộc | Mô tả | Ví dụ |
|-----|----------|-------|-------|
| `MYSQL_DSN` | ✓ | DSN MySQL | `user:pass@tcp(host:3306)/opsone?parseTime=true` |
| `API_ADDR` | ✓ | API listen addr | `:8080` |
| `DATA_SOURCE` | ✓ | mock vs prod | `production` |
| `CORS_ORIGIN` | ✓ | FE origin | `https://opsone.vercel.app` |
| `AAD_TENANT_ID` | ✓ | Entra tenant | UUID |
| `AAD_API_CLIENT_ID` | ✓ | App registration backend | UUID |
| `AAD_ISSUER` | ✓ | OIDC issuer | `https://login.microsoftonline.com/{tid}/v2.0` |
| `AAD_JWKS_URL` | tự suy | JWKS endpoint | (default từ issuer) |
| `DEV_AUTH_BYPASS` | ✗ | **Phải `false` ở prod** | `false` |
| `SMTP_HOST` | ✓ | SMTP | `smtp.sendgrid.net` |
| `SMTP_PORT` | ✓ | | `587` |
| `SMTP_USER` / `SMTP_PASS` | ✓ | | |
| `SMTP_FROM` | ✓ | | `opsone@company.com` |
| `NOTIFICATION_MOCK` | ✗ | Bật → chỉ log, không gửi | `false` |
| `LLM_API_URL` | có nếu chat LLM | MaaS base URL; trống → `https://maas-llm-aiplatform-hcm.api.vngcloud.vn/v1` | |
| `LLM_API_KEY` | có nếu chat LLM | Alias `AIP_API_KEY`; trống → chat stub | |
| `LLM_MODEL` | có nếu chat LLM | id từ `GET /v1/models` | `minimax/minimax-m2.5` |
| `LLM_TIMEOUT_SECONDS` | ✗ | Timeout HTTP chat agent | `30` |
| `AGENT_LOCALE` | ✓ | | `vi-VN` |

### 15.6 Health checks & observability

| Endpoint | Mục đích |
|----------|----------|
| `GET /api/v1/health` (public) | Liveness — trả `{ "status": "ok", "version": "<git_sha>" }` |
| `GET /api/v1/health/ready` (public) | Readiness — ping MySQL, kiểm tra worker-agent có tick gần đây không |
| `GET /api/v1/metrics` (Ops auth) | Metric Prometheus (cycles_completed, llm_call_latency_ms, …) — tuỳ chọn |

**Logs:** structured JSON (`zerolog` hoặc `slog`); fields bắt buộc `time`, `level`, `msg`, `cycle_id` (nếu trong chu kỳ), `product_code`.

### 15.7 Backup & DR

- **Aiven MySQL:** bật automatic backup daily, retention 14 ngày.
- **Self-hosted:** cron `mysqldump opsone | gzip > /backups/$(date +%F).sql.gz`; sync S3.
- **Restore drill:** quý 1 lần — restore vào staging, chạy `make db-verify` + E2E §10.

### 15.8 Security checklist

- [ ] HTTPS bắt buộc (Vercel auto / Let's Encrypt với Nginx).
- [ ] MySQL user app: chỉ `SELECT/INSERT/UPDATE/DELETE` trên schema `opsone`; **không** `GRANT ALL`.
- [ ] Secret qua env / vault — **không** commit `.env` (gitignore mặc định).
- [ ] CORS chỉ allow origin FE đã biết.
- [ ] Rate limit `/api/v1/chat` (5 req/min/user).
- [ ] Audit log: mọi mutate có `executed_by` từ JWT (§2.6.8).
- [ ] `DEV_AUTH_BYPASS=true` chỉ ở dev; CI assert chặn build prod nếu env này bật.

### 15.9 Checklist Deployment

- [ ] `infra/` có: `docker-compose.yml`, `Dockerfile.go`, `Dockerfile.web`, `nginx.conf`, `.env.example`.
- [ ] `make deploy-local` chạy được trong < 3 phút.
- [ ] CI GitHub Actions xanh trên `main`; build & push GHCR image có tag SHA.
- [ ] Doc 1 trang: *"Deploy lên Railway + Vercel + Aiven"* (link 3 dashboard).
- [ ] Postmortem template: incident production → file `runbooks/POSTMORTEM_TEMPLATE.md`.

---

