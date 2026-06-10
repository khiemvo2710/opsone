import type { ProductThreshold, ScopeLiveMetrics } from '../types/api';
import {
  METRIC_HEADERS,
  fmtMetricPct,
  fmtMetricTxn,
  liveMetricValue,
  metricCellClass,
  type MetricKind,
} from '../utils/scopeMetrics';

function formatValue(kind: MetricKind, value: number | undefined): string {
  if (kind === 'pending_txn' || kind === 'fail_txn') {
    return fmtMetricTxn(value);
  }
  return fmtMetricPct(value);
}

interface Props {
  live?: ScopeLiveMetrics;
  threshold?: ProductThreshold;
}

export function ScopeMetricCells({ live, threshold }: Props) {
  return (
    <>
      {METRIC_HEADERS.map(({ key }) => {
        const value = liveMetricValue(live, key);
        return (
          <td key={key} className={`mono nowrap ${metricCellClass(key, value, threshold)}`}>
            {formatValue(key, value)}
          </td>
        );
      })}
    </>
  );
}
