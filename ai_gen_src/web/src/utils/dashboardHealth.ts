import type { DashboardOverviewRow, HealthStatus, ProductThreshold } from '../types/api';
import { isSkuUnderActiveMaintenance } from './maintenanceDisplay';
import { shouldShowManualApproval } from './scopeAuto';
import { isAnyProviderBreached } from './scopeMetrics';

const HEALTH_RANK: Record<string, number> = { green: 0, yellow: 1, red: 2 };

const HEALTH_LABEL: Record<string, string> = {
  green: 'Hệ thống OK',
  yellow: 'Đang theo dõi / xử lý',
  red: 'Đang có vấn đề',
};

export function normalizeHealth(status: HealthStatus | undefined): HealthStatus {
  const key = (status ?? 'green').toLowerCase();
  if (key in HEALTH_RANK) return key;
  return 'green';
}

export function worstHealth(statuses: Iterable<HealthStatus>): HealthStatus {
  let worst: HealthStatus = 'green';
  let rank = 0;
  for (const raw of statuses) {
    const key = normalizeHealth(raw);
    const r = HEALTH_RANK[key] ?? 0;
    if (r > rank) {
      rank = r;
      worst = key;
    }
  }
  return worst;
}

export function rowPendingApprove(row: DashboardOverviewRow): boolean {
  if (!shouldShowManualApproval(row)) return false;
  const plan = row.pending_plan;
  if (
    plan &&
    (plan.suggested || plan.status === 'pending_approve' || plan.status === 'draft')
  ) {
    return true;
  }
  return Boolean(row.pending_maintenance && !row.maintenance);
}

/** Trạng thái hiển thị cột TT — dùng health_status từ API (đã tính consecutive_cycles). */
export function effectiveRowHealth(
  row: DashboardOverviewRow,
  threshold?: ProductThreshold,
): HealthStatus {
  if (isSkuUnderActiveMaintenance(row)) return 'green';
  if (rowPendingApprove(row)) return 'red';
  if (row.health_status) {
    return normalizeHealth(row.health_status);
  }
  if (isAnyProviderBreached(row.provider_metrics, threshold)) return 'yellow';
  return 'green';
}

export function overallHealthFromOverview(
  rows: DashboardOverviewRow[],
  thresholdsByProduct?: Record<string, ProductThreshold>,
): HealthStatus {
  if (rows.length === 0) return 'green';
  return worstHealth(
    rows.map((row) => effectiveRowHealth(row, thresholdsByProduct?.[row.product_code])),
  );
}

export function healthLabel(status: HealthStatus): string {
  return HEALTH_LABEL[normalizeHealth(status)] ?? String(status);
}

export function overallHealthSummary(
  status: HealthStatus,
  apiSummary?: string,
  overviewRows?: DashboardOverviewRow[],
): string | undefined {
  const key = normalizeHealth(status);
  if (key === 'red') {
    const fromOverview = incidentSummaryFromOverview(overviewRows);
    if (fromOverview) return fromOverview;
    const formatted = formatLegacyIncidentSummary(apiSummary);
    if (formatted) return formatted;
    return 'Có SKU hoặc loại dịch vụ đang báo đỏ';
  }
  if (key === 'yellow') {
    return apiSummary || 'Một số sản phẩm vượt ngưỡng — đang theo dõi';
  }
  return apiSummary || 'Hệ thống ổn định';
}

const PRODUCT_CODE_RE = /^[A-Z][A-Z0-9_]+$/;

const FALLBACK_LABELS: Record<string, string> = {
  MOBIFONE: 'Thẻ Mobifone',
  VINAPHONE: 'Thẻ Vinaphone',
  VIETTEL: 'Thẻ Viettel',
  ZING: 'Thẻ Zing',
  GARENA: 'Thẻ Garena',
  TOPUP_MOBI: 'Topup Mobifone',
  TOPUP_VINA: 'Topup Vinaphone',
  TOPUP_VIETTEL: 'Topup Viettel',
  DATA_MOBI: 'Data Mobifone',
  DATA_VINA: 'Data Vinaphone',
  DATA_VIETTEL: 'Data Viettel',
};

function productLabel(code: string, rowLabel?: string): string {
  if (rowLabel) return rowLabel;
  return FALLBACK_LABELS[code] ?? code.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function incidentSummaryFromOverview(rows?: DashboardOverviewRow[]): string | undefined {
  if (!rows?.length) return undefined;
  const seen = new Set<string>();
  const labels: string[] = [];
  for (const row of rows) {
    if (normalizeHealth(row.health_status) !== 'red') continue;
    if (seen.has(row.product_code)) continue;
    seen.add(row.product_code);
    labels.push(productLabel(row.product_code, row.product_label));
  }
  labels.sort((a, b) => a.localeCompare(b, 'vi'));
  if (labels.length === 0) return undefined;
  return `Sự cố: ${labels.join(', ')}`;
}

function formatLegacyIncidentSummary(apiSummary?: string): string | undefined {
  if (!apiSummary?.startsWith('Sự cố:')) return apiSummary;
  const rest = apiSummary.slice('Sự cố:'.length).trim().replace(/^\[|\]$/g, '');
  const tokens = rest.split(/[\s,;]+/).map((t) => t.trim()).filter(Boolean);
  const codes = tokens.filter((t) => PRODUCT_CODE_RE.test(t));
  if (codes.length === 0) return apiSummary;
  const labels = [...new Set(codes.map((c) => productLabel(c)))].sort((a, b) => a.localeCompare(b, 'vi'));
  return `Sự cố: ${labels.join(', ')}`;
}
