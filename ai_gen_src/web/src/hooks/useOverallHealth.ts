import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { DashboardOverview, HealthStatusResponse } from '../types/api';
import { useProductThresholds } from './useProductThresholds';
import {
  healthLabel,
  normalizeHealth,
  overallHealthFromOverview,
  overallHealthSummary,
  worstHealth,
} from '../utils/dashboardHealth';

export function useOverallHealth() {
  const { data: health } = useQuery({
    queryKey: ['health-status'],
    queryFn: () => api<HealthStatusResponse>('/health-status'),
    refetchInterval: 30_000,
  });

  const { data: overview } = useQuery({
    queryKey: ['dashboard-overview'],
    queryFn: () => api<DashboardOverview>('/dashboard/overview'),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const productCodes = useMemo(
    () => (overview?.rows ?? []).map((r) => r.product_code),
    [overview?.rows],
  );
  const thresholdsByProduct = useProductThresholds(productCodes, overview?.thresholds);

  return useMemo(() => {
    const fromOverview = overallHealthFromOverview(overview?.rows ?? [], thresholdsByProduct);
    const fromApi = normalizeHealth(health?.health_status);
    const status = worstHealth([fromOverview, fromApi]);
    return {
      status,
      label: healthLabel(status),
      summary: overallHealthSummary(status, health?.health_summary, overview?.rows),
      cycleId: health?.cycle_id,
    };
  }, [health?.health_status, health?.health_summary, health?.cycle_id, overview?.rows, thresholdsByProduct]);
}
