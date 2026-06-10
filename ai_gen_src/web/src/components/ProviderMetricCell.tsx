import type { ProductThreshold, ProviderLiveMetrics } from '../types/api';
import {
  fmtMetricPct,
  fmtMetricTxn,
  metricCellClass,
  providerMetricValue,
  type MetricKind,
} from '../utils/scopeMetrics';

const PCT_LINES: { key: MetricKind; label: string }[] = [
  { key: 'success', label: 'Success' },
  { key: 'pending', label: 'Pending' },
  { key: 'fail', label: 'Fail' },
];

const COUNT_LINES: { key: MetricKind; label: string }[] = [
  { key: 'pending_txn', label: 'Pending' },
  { key: 'fail_txn', label: 'Fail' },
];

function zeroInactiveMetrics(routingPct: number): ProviderLiveMetrics {
  return {
    routing_pct: routingPct,
    success_pct: 0,
    pending_pct: 0,
    fail_pct: 0,
    pending_txn: 0,
    fail_txn: 0,
  };
}

function formatLine(kind: MetricKind | 'routing', pm: ProviderLiveMetrics | undefined, forceZero = false): string {
  if (!pm && !forceZero) return '—';
  if (kind === 'routing') {
    const v = pm?.routing_pct ?? 0;
    if (!forceZero && (v == null || Number.isNaN(v))) return '—';
    return `${Math.round(v * 10) / 10}%`;
  }
  if (forceZero) {
    if (kind === 'pending_txn' || kind === 'fail_txn') return fmtMetricTxn(0);
    return fmtMetricPct(0);
  }
  const value = providerMetricValue(pm!, kind);
  if (value == null || Number.isNaN(value)) return '—';
  if (kind === 'pending_txn' || kind === 'fail_txn') {
    return fmtMetricTxn(value);
  }
  return fmtMetricPct(value);
}

function lineClass(
  kind: MetricKind | 'routing',
  pm: ProviderLiveMetrics | undefined,
  th?: ProductThreshold,
  inactive = false,
): string {
  if (inactive || !pm || (pm.routing_pct ?? 0) <= 0) {
    return 'provider-metric-line provider-metric-line--na';
  }
  if (kind === 'routing') {
    return 'provider-metric-line provider-metric-line--route';
  }
  const value = providerMetricValue(pm, kind);
  const base = metricCellClass(kind, value, th);
  if (base.includes('--bad')) return 'provider-metric-line provider-metric-line--bad';
  if (base.includes('--ok')) return 'provider-metric-line provider-metric-line--ok';
  return 'provider-metric-line provider-metric-line--na';
}

function MetricLines({
  lines,
  pm,
  threshold,
  inactive = false,
}: {
  lines: { key: MetricKind; label: string }[];
  pm: ProviderLiveMetrics;
  threshold?: ProductThreshold;
  inactive?: boolean;
}) {
  return (
    <ul className="provider-metric-list">
      {lines.map(({ key, label }) => (
        <li key={key} className={lineClass(key, pm, threshold, inactive)}>
          <span className="provider-metric-list__label muted">{label}</span>
          <span className="provider-metric-list__value">{formatLine(key, pm, inactive)}</span>
        </li>
      ))}
    </ul>
  );
}

interface Props {
  provider: string;
  metrics?: Record<string, ProviderLiveMetrics>;
  routingPct?: number;
  supported?: boolean;
  threshold?: ProductThreshold;
  inactive?: boolean;
  maintained?: boolean;
  reopenDisabled?: boolean;
  reopenBusy?: boolean;
  scopeRestoring?: boolean;
  onReopen?: () => void;
}

export function ProviderMetricCell({
  provider,
  metrics,
  routingPct,
  supported = true,
  threshold,
  inactive = false,
  maintained = false,
  reopenDisabled,
  reopenBusy,
  scopeRestoring,
  onReopen,
}: Props) {
  if (!supported) {
    return <td className="provider-metric-cell muted">—</td>;
  }

  const pmRaw = metrics?.[provider];
  const pct = pmRaw?.routing_pct ?? routingPct ?? 0;
  const disabled = inactive || maintained;
  const displayPct = maintained ? 0 : pct;
  const pm = disabled ? zeroInactiveMetrics(displayPct) : pmRaw;

  if (!pm && !disabled) {
    return <td className="provider-metric-cell muted">—</td>;
  }

  const displayPm = pm ?? zeroInactiveMetrics(displayPct);
  const cellClass = maintained
    ? ' provider-metric-cell--maint'
    : inactive
      ? ' provider-metric-cell--inactive'
      : '';

  return (
    <td className={`provider-metric-cell${cellClass}`}>
      <div className={`provider-metric-zones${maintained ? ' provider-metric-zones--disabled' : ''}`}>
        <div className="provider-metric-zone provider-metric-zone--routing">
          <span className={`provider-metric-route-line ${lineClass('routing', displayPm, threshold, disabled)}`}>
            <span className="provider-metric-route-line__name">{provider}</span>
            <span className="provider-metric-route-line__pct">{formatLine('routing', displayPm, disabled)}</span>
          </span>
        </div>
        <div className="provider-metric-zone provider-metric-zone--pct">
          <span className="provider-metric-zone__head muted">%</span>
          <MetricLines lines={PCT_LINES} pm={displayPm} threshold={threshold} inactive={disabled} />
        </div>
        <div className="provider-metric-zone provider-metric-zone--count">
          <span className="provider-metric-zone__head muted">GD</span>
          <MetricLines lines={COUNT_LINES} pm={displayPm} threshold={threshold} inactive={disabled} />
        </div>
      </div>
      {inactive && !maintained && !scopeRestoring && onReopen && (
        <button
          type="button"
          className="btn btn--ghost btn--xs provider-metric-cell__reopen"
          disabled={reopenDisabled || reopenBusy}
          title="Mở lại provider — chỉnh % routing"
          onClick={onReopen}
        >
          {reopenBusy ? '…' : 'Mở lại'}
        </button>
      )}
    </td>
  );
}
