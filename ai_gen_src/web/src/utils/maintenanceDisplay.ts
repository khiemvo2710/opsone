import type { DashboardOverviewRow } from '../types/api';

/** Scope has an active maintenance window (full SKU or per-provider). */
export function isSkuUnderActiveMaintenance(row: DashboardOverviewRow): boolean {
  if (row.maintenance) {
    return true;
  }
  const routing = row.routing_pct ?? {};
  let active = 0;
  let maintained = 0;
  for (const [provider, pct] of Object.entries(routing)) {
    if (pct <= 0) {
      continue;
    }
    active += 1;
    if (isProviderUnderMaintenance(row, provider)) {
      maintained += 1;
    }
  }
  return active > 0 && maintained >= active;
}

/** Provider column disabled when active maintenance covers this provider (§2.3.2). */
export function isProviderUnderMaintenance(row: DashboardOverviewRow, provider: string): boolean {
  const pm = row.provider_metrics?.[provider];
  if (pm?.under_maintenance) {
    return true;
  }

  const maint = row.maintenance;
  if (!maint) {
    return false;
  }

  if (maint.provider_codes?.includes(provider)) {
    return true;
  }

  if (maint.scope_level) {
    const pct = row.routing_pct?.[provider] ?? pm?.routing_pct ?? 0;
    return pct > 0;
  }

  if (maint.provider_code === provider) {
    return true;
  }

  return false;
}
