import { useParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { Incident } from '../types/api';
import { incidentStatusLabel, incidentResolutionLabel } from '../utils/incidentStatus';
import { HealthBadge } from '../components/HealthBadge';

const SEVERITY_STATUS: Record<string, string> = {
  critical: 'red',
  high: 'red',
  medium: 'yellow',
  low: 'yellow',
};

export function IncidentDetail() {
  const { id } = useParams<{ id: string }>();

  const { data, isLoading, error } = useQuery({
    queryKey: ['incident', id],
    queryFn: () => api<Incident>(`/incidents/${id}`),
    enabled: Boolean(id),
  });

  if (isLoading) {
    return <p className="loading">Đang tải sự cố...</p>;
  }

  if (error || !data) {
    return (
      <div className="page">
        <p className="error">Không tìm thấy sự cố #{id}.</p>
        <Link to="/">← Dashboard</Link>
      </div>
    );
  }

  const health = SEVERITY_STATUS[data.severity] ?? 'yellow';

  return (
    <div className="page incident-detail">
      <Link to="/" className="back-link">
        ← Dashboard
      </Link>

      <time className="incident-detail__time" dateTime={data.created_at}>
        {new Date(data.created_at).toLocaleString('vi-VN')}
      </time>

      <h1>Sự cố #{data.incident_id}</h1>

      <div className="incident-detail__grid">
        <div className="detail-field">
          <span className="detail-field__label">Mức độ</span>
          <HealthBadge status={health} label={data.severity} />
        </div>
        <div className="detail-field">
          <span className="detail-field__label">Trạng thái</span>
          <span className={`badge badge--${data.status}`}>{incidentStatusLabel(data.status)}</span>
        </div>
        {data.resolution_action && (
          <div className="detail-field">
            <span className="detail-field__label">Hành động xử lý</span>
            <span>{incidentResolutionLabel(data.resolution_action)}</span>
          </div>
        )}
        {data.handled_by && (
          <div className="detail-field">
            <span className="detail-field__label">Người xử lý</span>
            <span>{data.handled_by}</span>
          </div>
        )}
        {data.handled_at && (
          <div className="detail-field">
            <span className="detail-field__label">Thời gian xử lý</span>
            <span>{new Date(data.handled_at).toLocaleString('vi-VN')}</span>
          </div>
        )}
        <div className="detail-field">
          <span className="detail-field__label">Sản phẩm</span>
          <span>{data.product_code}</span>
        </div>
        <div className="detail-field">
          <span className="detail-field__label">Provider</span>
          <span>{data.provider_code || '—'}</span>
        </div>
        <div className="detail-field">
          <span className="detail-field__label">SKU</span>
          <span>{data.sku_code || '—'}</span>
        </div>
        {data.cycle_id != null && (
          <div className="detail-field">
            <span className="detail-field__label">Chu kỳ phân tích</span>
            <span>#{data.cycle_id}</span>
          </div>
        )}
      </div>

      <section className="incident-detail__summary card">
        <h2>Mô tả</h2>
        <p>{data.summary || 'Không có mô tả chi tiết.'}</p>
      </section>
    </div>
  );
}
