import type { DashboardOverviewRow } from '../types/api';

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
