import type { MaintenanceWindow } from '../types/api';

export interface GroupedMaintenanceWindow extends MaintenanceWindow {
  maintenance_ids: string[];
  provider_codes: string[];
}

export function groupMaintenanceWindows(items: MaintenanceWindow[]): GroupedMaintenanceWindow[] {
  const map = new Map<string, GroupedMaintenanceWindow>();
  for (const mw of items) {
    const key = `${mw.product_code}\x00${mw.sku_code}\x00${mw.starts_at}\x00${mw.ends_at}`;
    const existing = map.get(key);
    if (!existing) {
      map.set(key, {
        ...mw,
        maintenance_ids: [mw.maintenance_id],
        provider_codes: [mw.provider_code],
        provider_code: mw.provider_code,
      });
      continue;
    }
    existing.maintenance_ids.push(mw.maintenance_id);
    if (!existing.provider_codes.includes(mw.provider_code)) {
      existing.provider_codes.push(mw.provider_code);
      existing.provider_code = existing.provider_codes.join(' + ');
    }
  }
  return [...map.values()];
}
