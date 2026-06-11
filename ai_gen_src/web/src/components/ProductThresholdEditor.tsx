import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import { useToast } from '../context/ToastContext';
import type { ProductThreshold } from '../types/api';

export interface ThresholdPutBody {
  success_rate_min_pct: number;
  pending_rate_max_pct: number;
  fail_rate_max_pct: number;
  fail_txn_count_max: number;
  pending_txn_count_max: number;
  error_event_count_max: number;
  alert_email_enabled: boolean;
}

function normalizeThreshold(data: ProductThreshold): ProductThreshold {
  return {
    ...data,
    fail_txn_count_max: data.fail_txn_count_max ?? 50,
    pending_txn_count_max: data.pending_txn_count_max ?? 5,
    error_event_count_max: data.error_event_count_max ?? 50,
  };
}

function buildThresholdPutBody(draft: ProductThreshold): ThresholdPutBody {
  return {
    success_rate_min_pct: draft.success_rate_min_pct,
    pending_rate_max_pct: draft.pending_rate_max_pct,
    fail_rate_max_pct: draft.fail_rate_max_pct,
    fail_txn_count_max: draft.fail_txn_count_max ?? 50,
    pending_txn_count_max: draft.pending_txn_count_max ?? 5,
    error_event_count_max: draft.error_event_count_max ?? 50,
    alert_email_enabled: draft.alert_email_enabled,
  };
}

interface Props {
  productCode: string;
  providers: string[];
  /** Từ overview.thresholds — bỏ qua GET riêng khi đã có. */
  threshold?: ProductThreshold;
  productLabel?: string;
}

export function ProductThresholdEditor({
  productCode,
  providers,
  threshold,
  productLabel,
}: Props) {
  const qc = useQueryClient();
  const { showToast } = useToast();
  const [draft, setDraft] = useState<ProductThreshold | null>(
    threshold ? normalizeThreshold(threshold) : null,
  );

  const { data, isLoading } = useQuery({
    queryKey: ['threshold', productCode],
    queryFn: () => api<ProductThreshold>(`/products/${productCode}/thresholds`),
    enabled: !threshold,
    staleTime: 120_000,
  });

  useEffect(() => {
    if (threshold) {
      setDraft(normalizeThreshold(threshold));
      return;
    }
    if (data) setDraft(normalizeThreshold(data));
  }, [data, threshold]);

  const save = useMutation({
    mutationFn: (body: ThresholdPutBody) =>
      api<ProductThreshold>(`/products/${productCode}/thresholds`, {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: (saved) => {
      setDraft(normalizeThreshold(saved));
      void qc.invalidateQueries({ queryKey: ['threshold', productCode] });
      showToast('ok', 'Đã lưu ngưỡng');
    },
    onError: (err: Error) => {
      showToast('err', err instanceof ApiClientError ? err.message : 'Lưu thất bại');
    },
  });

  const labelColSpan = 2;
  /** Dùng thêm cột maint (không đổi width cột provider trong colgroup). */
  const fieldsColSpan = providers.length + 1;
  const actionsColSpan = 1;
  const label = productLabel?.trim() || productCode;

  const productTitleCell = (
    <td className="overview-table__threshold-spacer">
      <span className="overview-table__threshold-product-name" title={label}>
        {label}
      </span>
    </td>
  );

  const thresholdLabelCell = (
    <td colSpan={labelColSpan} className="overview-table__threshold-label-cell">
      <span className="overview-threshold-band__title" title="Ngưỡng cảnh báo">
        Ngưỡng cảnh báo
      </span>
    </td>
  );

  if ((!threshold && isLoading) || !draft) {
    return (
      <>
        {productTitleCell}
        {thresholdLabelCell}
        <td colSpan={fieldsColSpan + actionsColSpan} className="overview-table__threshold-loading">
          <span className="muted">Đang tải ngưỡng…</span>
        </td>
      </>
    );
  }

  const update = <K extends keyof ProductThreshold>(key: K, value: ProductThreshold[K]) => {
    setDraft((t) => (t ? { ...t, [key]: value } : t));
  };

  const onSave = () => {
    save.mutate(buildThresholdPutBody(draft));
  };

  return (
    <>
      {productTitleCell}
      {thresholdLabelCell}
      <td colSpan={fieldsColSpan} className="overview-table__threshold-fields-cell">
        <div className="overview-threshold-band__fields">
          <label className="overview-threshold-field" title="Đỏ khi % Success ≤">
            <span className="overview-threshold-field__label">% Success</span>
            <span className="overview-threshold-field__input-wrap">
              <span className="overview-threshold-field__op muted">≤</span>
              <input
                type="number"
                min={0}
                max={100}
                step={0.1}
                value={draft.success_rate_min_pct}
                onChange={(e) => update('success_rate_min_pct', Number(e.target.value))}
              />
            </span>
          </label>
          <label className="overview-threshold-field" title="Đỏ khi % Pending ≥">
            <span className="overview-threshold-field__label">% Pending</span>
            <span className="overview-threshold-field__input-wrap">
              <span className="overview-threshold-field__op muted">≥</span>
              <input
                type="number"
                min={0}
                max={100}
                step={0.1}
                value={draft.pending_rate_max_pct}
                onChange={(e) => update('pending_rate_max_pct', Number(e.target.value))}
              />
            </span>
          </label>
          <label className="overview-threshold-field" title="Đỏ khi % Fail ≥">
            <span className="overview-threshold-field__label">% Fail</span>
            <span className="overview-threshold-field__input-wrap">
              <span className="overview-threshold-field__op muted">≥</span>
              <input
                type="number"
                min={0}
                max={100}
                step={0.1}
                value={draft.fail_rate_max_pct}
                onChange={(e) => update('fail_rate_max_pct', Number(e.target.value))}
              />
            </span>
          </label>
          <label className="overview-threshold-field" title="Đỏ khi Pending ≥">
            <span className="overview-threshold-field__label">Pending</span>
            <span className="overview-threshold-field__input-wrap">
              <span className="overview-threshold-field__op muted">≥</span>
              <input
                type="number"
                min={1}
                value={draft.pending_txn_count_max}
                onChange={(e) => update('pending_txn_count_max', Number(e.target.value))}
              />
            </span>
          </label>
          <label className="overview-threshold-field" title="Đỏ khi Fail ≥">
            <span className="overview-threshold-field__label">Fail</span>
            <span className="overview-threshold-field__input-wrap">
              <span className="overview-threshold-field__op muted">≥</span>
              <input
                type="number"
                min={1}
                value={draft.fail_txn_count_max}
                onChange={(e) => update('fail_txn_count_max', Number(e.target.value))}
              />
            </span>
          </label>
        </div>
      </td>
      <td colSpan={actionsColSpan} className="overview-table__threshold-actions-cell">
        <div className="overview-threshold-band__actions">
          <label className="overview-threshold-actions__check" title="Cảnh báo email">
            <input
              type="checkbox"
              checked={draft.alert_email_enabled}
              onChange={(e) => update('alert_email_enabled', e.target.checked)}
            />
            <span>Email</span>
          </label>
          <button
            type="button"
            className="btn btn--primary btn--xs"
            disabled={save.isPending}
            onClick={onSave}
          >
            {save.isPending ? '…' : 'Lưu'}
          </button>
        </div>
      </td>
    </>
  );
}
