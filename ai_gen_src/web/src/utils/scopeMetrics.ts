import type { ProductThreshold, ProviderLiveMetrics, ScopeLiveMetrics } from '../types/api';

export type MetricKind = 'success' | 'pending' | 'fail' | 'pending_txn' | 'fail_txn';

export function fmtMetricPct(v: number | undefined): string {
  if (v == null || Number.isNaN(v)) return '—';
  return `${Math.round(v * 10) / 10}%`;
}

export function fmtMetricTxn(v: number | undefined): string {
  if (v == null || Number.isNaN(v)) return '—';
  return String(Math.round(v));
}

/** true = vượt ngưỡng (hiển thị đỏ). Khớp dấu trên hàng Ngưỡng: S ≤ ngưỡng; các cột còn lại ≥ ngưỡng. */
export function isMetricBreached(
  kind: MetricKind,
  value: number | undefined,
  th: ProductThreshold | undefined,
): boolean {
  if (value == null || !th) return false;
  switch (kind) {
    case 'success':
      return value <= th.success_rate_min_pct;
    case 'pending':
      return value >= th.pending_rate_max_pct;
    case 'fail':
      return value >= th.fail_rate_max_pct;
    case 'pending_txn':
      return value >= (th.pending_txn_count_max ?? 5);
    case 'fail_txn':
      return value >= (th.fail_txn_count_max ?? 50);
    default:
      return false;
  }
}

/** true nếu bất kỳ metric live nào vượt ngưỡng (5 điều kiện OR). */
export function isScopeMetricsBreached(
  live: ScopeLiveMetrics | undefined,
  th: ProductThreshold | undefined,
): boolean {
  if (!live || !th) return false;
  return METRIC_HEADERS.some(({ key }) =>
    isMetricBreached(key, liveMetricValue(live, key), th),
  );
}

export function providerMetricValue(
  pm: ProviderLiveMetrics | undefined,
  kind: MetricKind,
): number | undefined {
  if (!pm) return undefined;
  switch (kind) {
    case 'success':
      return pm.success_pct;
    case 'pending':
      return pm.pending_pct;
    case 'fail':
      return pm.fail_pct;
    case 'pending_txn':
      return pm.pending_txn;
    case 'fail_txn':
      return pm.fail_txn;
  }
}

/** true nếu bất kỳ provider active (routing > 0) có metric vượt ngưỡng. */
export function isAnyProviderBreached(
  metrics: Record<string, ProviderLiveMetrics> | undefined,
  th: ProductThreshold | undefined,
): boolean {
  if (!metrics || !th) return false;
  for (const pm of Object.values(metrics)) {
    if ((pm.routing_pct ?? 0) <= 0) continue;
    if (
      METRIC_HEADERS.some(({ key }) =>
        isMetricBreached(key, providerMetricValue(pm, key), th),
      )
    ) {
      return true;
    }
  }
  return false;
}

export const PROVIDER_METRIC_LINES: { key: MetricKind | 'routing'; label: string }[] = [
  { key: 'routing', label: '% Routing' },
  { key: 'success', label: 'Success' },
  { key: 'pending', label: 'Pending' },
  { key: 'fail', label: 'Fail' },
  { key: 'pending_txn', label: 'Pending' },
  { key: 'fail_txn', label: 'Fail' },
];

export function metricCellClass(
  kind: MetricKind,
  value: number | undefined,
  th: ProductThreshold | undefined,
): string {
  if (value == null) return 'metric-cell metric-cell--na';
  return isMetricBreached(kind, value, th) ? 'metric-cell metric-cell--bad' : 'metric-cell metric-cell--ok';
}

export function liveMetricValue(
  m: ScopeLiveMetrics | undefined,
  kind: MetricKind,
): number | undefined {
  if (!m) return undefined;
  switch (kind) {
    case 'success':
      return m.success_pct;
    case 'pending':
      return m.pending_pct;
    case 'fail':
      return m.fail_pct;
    case 'pending_txn':
      return m.pending_txn;
    case 'fail_txn':
      return m.fail_txn;
  }
}

export const METRIC_HEADERS = [
  { key: 'success' as const, label: 'S (%)' },
  { key: 'pending' as const, label: 'P (%)' },
  { key: 'fail' as const, label: 'F (%)' },
  { key: 'pending_txn' as const, label: 'P' },
  { key: 'fail_txn' as const, label: 'F' },
];
