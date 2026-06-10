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
): string | undefined {
  const key = normalizeHealth(status);
  if (key === 'red') {
    return apiSummary && apiSummary.includes('Sự cố') ? apiSummary : 'Có SKU hoặc loại dịch vụ đang báo đỏ';
  }
  if (key === 'yellow') {
    return apiSummary || 'Một số sản phẩm vượt ngưỡng — đang theo dõi';
  }
  return apiSummary || 'Hệ thống ổn định';
}
