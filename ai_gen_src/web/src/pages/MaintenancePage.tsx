import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { MaintenanceWindow } from '../types/api';
import { MaintenanceCard } from '../components/MaintenanceCard';
import { groupMaintenanceWindows } from '../utils/groupMaintenance';

export function MaintenancePage() {
  const { data, isLoading } = useQuery({
    queryKey: ['maintenance', 'all'],
    queryFn: () => api<{ items: MaintenanceWindow[] }>('/maintenance'),
  });

  const grouped = groupMaintenanceWindows(data?.items ?? []);

  if (isLoading) {
    return <p className="loading">Đang tải bảo trì...</p>;
  }

  return (
    <div className="page maintenance-page">
      <h1>Cửa sổ bảo trì</h1>
      <div className="card-grid">
        {grouped.map((mw) => (
          <MaintenanceCard key={mw.maintenance_ids.join('-')} window={mw} />
        ))}
      </div>
      {grouped.length === 0 && <p className="muted">Không có cửa sổ bảo trì.</p>}
    </div>
  );
}
