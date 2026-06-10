import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { AgentChange } from '../types/api';

function formatRouting(r?: Record<string, number>): string {
  if (!r) return '—';
  return Object.entries(r)
    .map(([k, v]) => `${k}: ${v}%`)
    .join(' · ');
}

export function AgentChanges() {
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['agent-changes'],
    queryFn: () => api<{ items: AgentChange[] }>('/agent-changes'),
  });

  const rollback = useMutation({
    mutationFn: (id: number) =>
      api(`/agent-changes/${id}/rollback`, {
        method: 'POST',
        body: JSON.stringify({ reason: 'rollback từ UI' }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['agent-changes'] });
      void qc.invalidateQueries({ queryKey: ['health-status'] });
    },
  });

  if (isLoading) {
    return <p className="loading">Đang tải lịch sử...</p>;
  }

  return (
    <div className="page changes-page">
      <h1>Lịch sử thay đổi & Rollback</h1>
      <p className="muted">Mọi thay đổi routing do OpsOne — hoàn tác về cấu hình trước (§8.7).</p>

      <div className="table-scroll">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Sản phẩm</th>
              <th>Trước</th>
              <th>Sau</th>
              <th>Trạng thái</th>
              <th>Thời gian</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {(data?.items ?? []).map((ch) => (
              <tr key={ch.id}>
                <td>{ch.id}</td>
                <td>
                  {ch.product_code}
                  {ch.sku_code ? ` / ${ch.sku_code}` : ''}
                </td>
                <td className="mono">{formatRouting(ch.routing_before)}</td>
                <td className="mono">{formatRouting(ch.routing_after)}</td>
                <td>
                  <span className={`badge badge--${ch.change_status}`}>{ch.change_status}</span>
                </td>
                <td>{new Date(ch.created_at).toLocaleString('vi-VN')}</td>
                <td>
                  {ch.change_status === 'applied' && (
                    <button
                      type="button"
                      className="btn btn--ghost"
                      disabled={rollback.isPending}
                      onClick={() => rollback.mutate(ch.id)}
                    >
                      Rollback
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {(data?.items ?? []).length === 0 && <p className="muted">Chưa có thay đổi.</p>}
    </div>
  );
}
