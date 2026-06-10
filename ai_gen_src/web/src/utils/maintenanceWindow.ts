import {
  formatDatetimeVi,
  isoToDatetimeLocalValue,
  toDatetimeLocalValue,
} from './datetimeLocal';

export { toDatetimeLocalValue };

export function isSkuWideMaintenance(reason?: string): boolean {
  return Boolean(reason?.includes('Tất cả provider đang routing'));
}

/** Lý do bảo trì SKU thủ công từ nút「Bảo trì dịch vụ」— khớp maintenanceTargetsForScope backend. */
export function manualServiceMaintenanceReason(): string {
  return 'Tất cả provider đang routing — Bảo trì dịch vụ thủ công';
}

export function maintenanceSuggestLabel(pm: {
  provider_code?: string;
  reason?: string;
  scope_level?: boolean;
}): string {
  if (isSkuWideMaintenance(pm.reason) || pm.scope_level) {
    return pm.reason ?? 'Đề xuất bảo trì SKU';
  }
  return pm.reason ?? 'Vượt ngưỡng';
}

export function maintenanceActiveTimes(
  startsAt: string,
  endsAt: string,
): { start: string; end: string } | null {
  const start = formatDatetimeVi(isoToDatetimeLocalValue(startsAt));
  const end = formatDatetimeVi(isoToDatetimeLocalValue(endsAt));
  if (!start || !end) return null;
  return { start, end };
}

export function maintenanceActiveLabel(startsAt: string, endsAt: string): string {
  const times = maintenanceActiveTimes(startsAt, endsAt);
  if (!times) return 'Bảo trì đang hoạt động';
  return `Bảo trì từ ${times.start} - ${times.end}`;
}

export function defaultMaintenanceWindow(durationMin = 60): { startsAt: string; endsAt: string } {
  const now = new Date();
  const end = new Date(now.getTime() + durationMin * 60 * 1000);
  return { startsAt: toDatetimeLocalValue(now), endsAt: toDatetimeLocalValue(end) };
}

export function maintenanceWindowFromISO(startsAt: string, endsAt: string): { startsAt: string; endsAt: string } {
  return {
    startsAt: isoToDatetimeLocalValue(startsAt),
    endsAt: isoToDatetimeLocalValue(endsAt),
  };
}

export function maintenanceWindowError(startsAt: string, endsAt: string): string | null {
  const s = new Date(startsAt).getTime();
  const e = new Date(endsAt).getTime();
  if (!Number.isFinite(s) || !Number.isFinite(e)) {
    return 'Thời gian không hợp lệ';
  }
  if (e <= s) {
    return 'Kết thúc phải sau bắt đầu';
  }
  return null;
}

export function maintenanceWindowISO(
  startsAt: string,
  endsAt: string,
): { starts_at: string; ends_at: string } | null {
  if (maintenanceWindowError(startsAt, endsAt)) {
    return null;
  }
  return {
    starts_at: new Date(startsAt).toISOString(),
    ends_at: new Date(endsAt).toISOString(),
  };
}

export function maintenanceWindowUnchanged(
  a: { startsAt: string; endsAt: string },
  b: { startsAt: string; endsAt: string },
): boolean {
  const isoA = maintenanceWindowISO(a.startsAt, a.endsAt);
  const isoB = maintenanceWindowISO(b.startsAt, b.endsAt);
  if (!isoA || !isoB) return false;
  return isoA.starts_at === isoB.starts_at && isoA.ends_at === isoB.ends_at;
}
