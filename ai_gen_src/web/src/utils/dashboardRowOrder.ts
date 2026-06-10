import type { DashboardOverviewRow } from '../types/api';

export interface ProductGroup {
  productCode: string;
  productLabel: string;
  rows: DashboardOverviewRow[];
}

/** Tên hiển thị dịch vụ (vd. Data Mobifone) — dùng toast/UI, không lộ product_code. */
export function productLabelForCode(
  rows: DashboardOverviewRow[] | undefined,
  productCode: string,
): string {
  const row = rows?.find((r) => r.product_code === productCode);
  const label = row?.product_label?.trim();
  return label || productCode;
}

export function fmtSkuCode(sku: string): string {
  return sku === '' ? '—' : sku;
}

/** Tên scope cho toast: "Data Mobifone · VNP20". */
export function scopeDisplayLabel(
  rows: DashboardOverviewRow[] | undefined,
  productCode: string,
  skuCode: string,
): string {
  return `${productLabelForCode(rows, productCode)} · ${fmtSkuCode(skuCode)}`;
}

export function compareSkuCode(a: string, b: string): number {
  const aNum = /^\d+$/.test(a);
  const bNum = /^\d+$/.test(b);
  if (aNum && bNum) {
    return Number(a) - Number(b);
  }
  return a.localeCompare(b, undefined, { numeric: true });
}

/** Cùng thứ tự nhóm/sắp xếp như ServiceOverviewTable. */
export function groupRowsByProduct(rows: DashboardOverviewRow[]): ProductGroup[] {
  const groups: ProductGroup[] = [];
  const index = new Map<string, number>();
  for (const row of rows) {
    const existing = index.get(row.product_code);
    if (existing === undefined) {
      index.set(row.product_code, groups.length);
      groups.push({
        productCode: row.product_code,
        productLabel: row.product_label || row.product_code,
        rows: [row],
      });
    } else {
      groups[existing].rows.push(row);
    }
  }
  for (const group of groups) {
    group.rows.sort((a, b) => compareSkuCode(a.sku_code, b.sku_code));
  }
  return groups;
}

/** Danh sách SKU theo thứ tự hiển thị trên bảng (trên → dưới). */
export function rowsInDisplayOrder(rows: DashboardOverviewRow[]): DashboardOverviewRow[] {
  return groupRowsByProduct(rows).flatMap((group) => group.rows);
}
