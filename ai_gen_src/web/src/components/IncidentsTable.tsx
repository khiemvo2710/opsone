import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { Incident, IncidentsListResponse } from '../types/api';
import { incidentStatusLabel, incidentResolutionLabel, formatHandledBy } from '../utils/incidentStatus';
import { HealthBadge } from './HealthBadge';

const SEVERITY_STATUS: Record<string, string> = {
  critical: 'red',
  high: 'red',
  medium: 'yellow',
  low: 'yellow',
};

const DEFAULT_PAGE_SIZE = 10;

function formatIncidentTime(iso: string): string {
  return new Date(iso).toLocaleString('vi-VN', {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatHandledCell(inc: Incident): string {
  const by = formatHandledBy(inc.handled_by, inc.resolution_action);
  if (inc.status === 'open' || !inc.handled_at) return by;
  return `${by} · ${formatIncidentTime(inc.handled_at)}`;
}

interface Props {
  /** When set, fetch paginated list from API (Dashboard). Omit to pass static items. */
  paginated?: boolean;
  pageSize?: number;
  items?: Incident[];
}

export function IncidentsTable({ paginated = false, pageSize = DEFAULT_PAGE_SIZE, items: staticItems }: Props) {
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ['incidents', page, pageSize],
    queryFn: () =>
      api<IncidentsListResponse>(`/incidents?page=${page}&page_size=${pageSize}`),
    enabled: paginated,
    refetchInterval: paginated ? 60_000 : false,
  });

  const items = paginated ? (data?.items ?? []) : (staticItems ?? []);
  const total = paginated ? (data?.total ?? 0) : items.length;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="incidents-table-wrap">
      <div className="table-scroll">
        <table className="data-table incidents-table">
          <colgroup>
            <col className="incidents-table__col-time" />
            <col className="incidents-table__col-id" />
            <col className="incidents-table__col-sev" />
            <col className="incidents-table__col-product" />
            <col className="incidents-table__col-provider" />
            <col className="incidents-table__col-sku" />
            <col className="incidents-table__col-summary" />
            <col className="incidents-table__col-status" />
            <col className="incidents-table__col-handled" />
          </colgroup>
          <thead>
            <tr>
              <th>Thời gian</th>
              <th>Mã</th>
              <th>Mức</th>
              <th>Sản phẩm</th>
              <th>Provider</th>
              <th>SKU</th>
              <th>Tóm tắt</th>
              <th>TT</th>
              <th>Xử lý</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && paginated ? (
              <tr>
                <td colSpan={9} className="muted">
                  Đang tải…
                </td>
              </tr>
            ) : (
              items.map((inc) => (
                <tr key={inc.id}>
                  <td className="incidents-table__time">
                    {formatIncidentTime(inc.created_at)}
                  </td>
                  <td className="mono incidents-table__id">#{inc.incident_id}</td>
                  <td>
                    <HealthBadge
                      status={SEVERITY_STATUS[inc.severity] ?? 'yellow'}
                      label={inc.severity}
                      size="sm"
                      compact
                    />
                  </td>
                  <td>{inc.product_code}</td>
                  <td>{inc.provider_code || '—'}</td>
                  <td>{inc.sku_code || '—'}</td>
                  <td className="incidents-table__summary">{inc.summary || '—'}</td>
                  <td className="incidents-table__status">
                    <span className="incidents-table__status-line">
                      <span className={`badge badge--${inc.status}`}>{incidentStatusLabel(inc.status)}</span>
                      {inc.resolution_action && inc.status !== 'open' && (
                        <span className="incidents-table__action muted" title="Hành động">
                          {' '}
                          · {incidentResolutionLabel(inc.resolution_action)}
                        </span>
                      )}
                    </span>
                  </td>
                  <td className="incidents-table__handled">{formatHandledCell(inc)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
        {!isLoading && items.length === 0 && <p className="muted">Không có sự cố.</p>}
      </div>
      {paginated && !isLoading && (
        <nav className="incidents-pagination" aria-label="Phân trang sự cố">
          <button
            type="button"
            className="btn btn--ghost btn--xs"
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            Trước
          </button>
          <span className="incidents-pagination__info muted">
            Trang {page}/{totalPages} · {total} sự cố
          </span>
          <button
            type="button"
            className="btn btn--ghost btn--xs"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          >
            Sau
          </button>
        </nav>
      )}
    </div>
  );
}
