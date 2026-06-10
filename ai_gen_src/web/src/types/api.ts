export type HealthStatus = 'green' | 'yellow' | 'red' | string;

export interface HealthStatusResponse {
  cycle_id?: number;
  cycle_started?: string;
  health_status: HealthStatus;
  health_label?: string;
  health_summary?: string;
  products?: ProductHealth[];
}

export interface ProductHealth {
  product_code: string;
  product_label?: string;
  health_status: HealthStatus;
  health_summary?: string;
}

export type ServiceType = 'card' | 'topup' | 'topup_data' | string;

export interface ScopeLiveMetrics {
  success_pct: number;
  pending_pct: number;
  fail_pct: number;
  pending_txn: number;
  fail_txn: number;
  total_txn?: number;
}

export interface ProviderLiveMetrics {
  routing_pct?: number;
  success_pct?: number;
  pending_pct?: number;
  fail_pct?: number;
  pending_txn?: number;
  fail_txn?: number;
  under_maintenance?: boolean;
}

export interface DashboardOverviewRow {
  product_code: string;
  product_label: string;
  service_type?: ServiceType;
  sku_code: string;
  health_status: HealthStatus;
  routing_pct: Record<string, number>;
  /** Tỷ lệ routing mặc định theo biz (baseline). */
  baseline_pct?: Record<string, number>;
  live_metrics?: ScopeLiveMetrics;
  provider_metrics?: Record<string, ProviderLiveMetrics>;
  maintenance?: {
    maintenance_id: string;
    maintenance_ids?: string[];
    provider_code: string;
    provider_codes?: string[];
    scope_level?: boolean;
    starts_at: string;
    ends_at: string;
    remaining_min: number;
    label_vi?: string;
    reason?: string;
  };
  pending_plan?: RoutingPlan & { suggested?: boolean };
  pending_maintenance?: {
    id?: number;
    sku_code?: string;
    provider_code?: string;
    scope_level?: boolean;
    reason?: string;
    action_type?: string;
    suggested?: boolean;
  };
  /** Chế độ auto per SKU (recommend_only | auto | time_window). */
  auto_action?: string;
  /** datetime-local value, e.g. 2026-06-10T08:00 */
  window_start?: string;
  window_end?: string;
}

export interface DashboardOverview {
  updated_at: string;
  providers: string[];
  /** Ngưỡng per product — gom trong overview để tránh N request riêng. */
  thresholds?: Record<string, ProductThreshold>;
  rows: DashboardOverviewRow[];
}

export interface AgentConfig {
  scheduler_enabled: boolean;
  scheduler_interval_min: number;
  data_source: string;
  mock_enabled: boolean;
  mock_interval_min: number;
  mock_scenario: string;
  agent_locale: string;
}

export interface Product {
  id: number;
  product_code: string;
  label: string;
  service_type: string;
  routing_mode: string;
  enabled: boolean;
}

export interface IncidentsListResponse {
  items: Incident[];
  total: number;
  page: number;
  page_size: number;
}

export interface Incident {
  id: number;
  incident_id: string;
  cycle_id?: number;
  severity: string;
  product_code: string;
  provider_code: string;
  sku_code: string;
  summary?: string;
  status: string;
  handled_by?: string;
  handled_at?: string;
  resolution_action?: string;
  created_at: string;
}

export interface RoutingPlan {
  id: number;
  cycle_id?: number;
  product_code: string;
  scope: string;
  sku_code: string;
  status: string;
  created_at: string;
  plan?: {
    proposed_pct?: Record<string, number>;
    current_pct?: Record<string, number>;
    reason_vi?: string;
  };
}

export interface MaintenanceWindow {
  maintenance_id: string;
  product_code: string;
  provider_code: string;
  sku_code: string;
  starts_at: string;
  ends_at: string;
  status: string;
  reason?: string;
}

export interface ProductThreshold {
  product_code: string;
  enabled: boolean;
  success_rate_min_pct: number;
  pending_rate_max_pct: number;
  fail_rate_max_pct: number;
  fail_txn_count_max: number;
  pending_txn_count_max: number;
  error_event_count_max: number;
  metrics_window_min: number;
  consecutive_cycles_required: number;
  alert_email_enabled: boolean;
}

export interface ApiError {
  code: string;
  message_vi: string;
}
