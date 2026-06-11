import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { AgentConfig } from '../types/api';

const FALLBACK_MIN = 60;

export function useMaintenanceDefaultDurationMin(): number {
  const { data } = useQuery({
    queryKey: ['config'],
    queryFn: () => api<AgentConfig>('/config'),
    staleTime: 60_000,
  });
  const v = data?.maintenance_default_duration_min;
  return v != null && v > 0 ? v : FALLBACK_MIN;
}
