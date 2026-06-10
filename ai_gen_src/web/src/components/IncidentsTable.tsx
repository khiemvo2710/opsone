import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { Incident, IncidentsListResponse } from '../types/api';
import { incidentStatusLabel, incidentResolutionLabel, formatHandledBy, formatHandledAt } from '../utils/incidentStatus';
import { HealthBadge } from './HealthBadge';

const SEVERITY_STATUS: Record<string, string> = {
  critical: 'red',
  high: 'red',
  medium: 'yellow',
  low: 'yellow',
};

const DEFAULT_PAGE_SIZE = 10;

function trunc(text: string, max = 56): string {
  if (text.length <= max) return text;
  return `${text.slice(0, max)}…`;
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
            <col className="incidents-table__col-handler" />
            <col className="incidents-table__col-handled" />
          </colgroup>
          <thead>
            <tr>
              <th>Thời gian</th>
              <th>Mã</th>
              <th>Mức độ</th>
              <th>Sản phẩm</th>
              <th>Provider</th>
              <th>SKU</th>
              <th>Tóm tắt</th>
              <th>TT</th>
              <th>Người xử lý</th>
              <th>Thời gian xử lý</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && paginated ? (
              <tr>
                <td colSpan={10} className="muted">
                  Đang tải…
                </td>
              </tr>
            ) : (
              items.map((inc) => (
                <tr key={inc.id}>
                  <td className="incidents-table__time">
                    {new Date(inc.created_at).toLocaleString('vi-VN')}
                  </td>
                  <td className="mono incidents-table__id">
                    <Link to={`/incidents/${inc.incident_id}`} className="table-link incidents-table__id-link">
                      #{inc.incident_id}
                    </Link>
                  </td>
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
                  <td className="incidents-table__summary" title={inc.summary}>
                    {inc.summary ? trunc(inc.summary) : '—'}
                  </td>
                  <td>
                    <span className={`badge badge--${inc.status}`}>{incidentStatusLabel(inc.status)}</span>
                    {inc.resolution_action && inc.status !== 'open' && (
                      <span className="incidents-table__action muted" title="Hành động">
                        {incidentResolutionLabel(inc.resolution_action)}
                      </span>
                    )}
                  </td>
                  <td className="incidents-table__handled-by">
                    {formatHandledBy(inc.handled_by, inc.resolution_action)}
                  </td>
                  <td className="incidents-table__handled-at">
                    {formatHandledAt(inc.handled_at, inc.status)}
                  </td>
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
