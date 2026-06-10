import { useEffect, useMemo, useState } from 'react';
import type { DashboardOverviewRow, ProductThreshold } from '../types/api';
import { HealthBadge } from './HealthBadge';
import { effectiveRowHealth } from '../utils/dashboardHealth';
import { rowsInDisplayOrder } from '../utils/dashboardRowOrder';

interface Props {
  rows: DashboardOverviewRow[];
  thresholdsByProduct?: Record<string, ProductThreshold>;
}

function scopeKey(row: DashboardOverviewRow): string {
  return `${row.product_code}:${row.sku_code}`;
}

export function RedSkuScrollNav({ rows, thresholdsByProduct = {} }: Props) {
  const redKeys = useMemo(
    () =>
      rowsInDisplayOrder(rows)
        .filter((row) => effectiveRowHealth(row, thresholdsByProduct[row.product_code]) === 'red')
        .map(scopeKey),
    [rows, thresholdsByProduct],
  );

  const [activeIndex, setActiveIndex] = useState(0);

  useEffect(() => {
    setActiveIndex(0);
  }, [redKeys.join('|')]);

  if (redKeys.length === 0) {
    return null;
  }

  const scrollTo = (index: number) => {
    const key = redKeys[index];
    if (!key) return;
    const el = document.getElementById(`overview-scope-${key}`);
    if (!el) return;
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    el.classList.add('overview-table__sku-row--flash');
    window.setTimeout(() => el.classList.remove('overview-table__sku-row--flash'), 1200);
  };

  const go = (delta: number) => {
    const next = (activeIndex + delta + redKeys.length) % redKeys.length;
    setActiveIndex(next);
    scrollTo(next);
  };

  return (
    <div className="sku-scroll-nav" aria-label="Điều hướng SKU đỏ">
      <button
        type="button"
        className="sku-scroll-nav__btn"
        title="SKU đỏ trước"
        aria-label="SKU đỏ trước"
        onClick={() => go(-1)}
      >
        ▲
      </button>
      <div className="sku-scroll-nav__status" title="SKU đang đỏ — điều hướng nhanh">
        <span className="sku-scroll-nav__count" aria-live="polite">
          {activeIndex + 1}/{redKeys.length}
        </span>
        <HealthBadge status="red" compact size="sm" label="SKU đang đỏ" />
      </div>
      <button
        type="button"
        className="sku-scroll-nav__btn"
        title="SKU đỏ tiếp theo"
        aria-label="SKU đỏ tiếp theo"
        onClick={() => go(1)}
      >
        ▼
      </button>
    </div>
  );
}
