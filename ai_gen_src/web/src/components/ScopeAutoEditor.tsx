import { useEffect, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import { useToast } from '../context/ToastContext';
import type { DashboardOverview } from '../types/api';
import { DateTimeLocalField } from './DateTimeLocalField';
import {
  datetimeRangeError,
  defaultScopeAutoWindow,
  fromApiDateTime,
} from '../utils/datetimeLocal';

const AUTO_ACTIONS = [
  { value: 'recommend_only', label: 'Chỉ đề xuất' },
  { value: 'auto', label: 'Tự động' },
  { value: 'time_window', label: 'Tự động theo khung giờ' },
] as const;

export interface ScopeAutoState {
  auto_action: string;
  window_start?: string;
  window_end?: string;
}

interface Props {
  productCode: string;
  skuCode: string;
  initial: ScopeAutoState;
  /** product = cấu hình toàn dịch vụ; sku = cấu hình từng SKU */
  level?: 'product' | 'sku';
}

function scopeAutoPath(productCode: string, skuCode: string): string {
  if (skuCode === '') {
    return `/scopes/${encodeURIComponent(productCode)}/auto`;
  }
  return `/scopes/${encodeURIComponent(productCode)}/${encodeURIComponent(skuCode)}/auto`;
}

function normalizeAction(v: string | undefined): string {
  if (v === 'auto' || v === 'time_window') return v;
  return 'recommend_only';
}

function actionDisplayLabel(action: string): string {
  return AUTO_ACTIONS.find((a) => a.value === action)?.label ?? 'Chỉ đề xuất';
}

function summaryTitle(initial: ScopeAutoState): string | undefined {
  if (initial.auto_action !== 'time_window') return undefined;
  const start = fromApiDateTime(initial.window_start, 8, 0);
  const end = fromApiDateTime(initial.window_end, 18, 0);
  if (!start || !end) return undefined;
  return `${start} → ${end}`;
}

export function ScopeAutoEditor({
  productCode,
  skuCode,
  initial,
  level = 'sku',
}: Props) {
  const qc = useQueryClient();
  const { showToast } = useToast();
  const [editing, setEditing] = useState(false);
  const [action, setAction] = useState(normalizeAction(initial.auto_action));
  const defaults = defaultScopeAutoWindow();
  const [windowStart, setWindowStart] = useState(
    fromApiDateTime(initial.window_start, 8, 0) || defaults.windowStart,
  );
  const [windowEnd, setWindowEnd] = useState(
    fromApiDateTime(initial.window_end, 18, 0) || defaults.windowEnd,
  );

  useEffect(() => {
    setAction(normalizeAction(initial.auto_action));
    setWindowStart(fromApiDateTime(initial.window_start, 8, 0));
    setWindowEnd(fromApiDateTime(initial.window_end, 18, 0));
    setEditing(false);
  }, [initial.auto_action, initial.window_start, initial.window_end]);

  const patchOverviewCache = (saved: ScopeAutoState) => {
    qc.setQueryData<DashboardOverview>(['dashboard-overview'], (old) => {
      if (!old?.rows?.length) return old;
      const rows = old.rows.map((row) => {
        if (row.product_code !== productCode) return row;
        if (skuCode === '') {
          return {
            ...row,
            product_auto_action: saved.auto_action,
            product_window_start: saved.window_start,
            product_window_end: saved.window_end,
            auto_action: saved.auto_action,
            window_start: saved.window_start,
            window_end: saved.window_end,
          };
        }
        if (row.sku_code !== skuCode) return row;
        return {
          ...row,
          scope_auto_action: saved.auto_action,
          scope_window_start: saved.window_start,
          scope_window_end: saved.window_end,
        };
      });
      return { ...old, rows };
    });
  };

  const save = useMutation({
    mutationFn: (body: ScopeAutoState) =>
      api<ScopeAutoState & { product_code?: string; sku_code?: string }>(
        scopeAutoPath(productCode, skuCode),
        {
          method: 'PUT',
          body: JSON.stringify(body),
        },
      ),
    onSuccess: async (saved) => {
      setEditing(false);
      const normalized: ScopeAutoState = {
        auto_action: saved.auto_action,
        window_start: saved.window_start,
        window_end: saved.window_end,
      };
      patchOverviewCache(normalized);
      await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      showToast('ok', 'Đã lưu chế độ Auto');
    },
    onError: (err: Error) => {
      showToast('err', err instanceof ApiClientError ? err.message : 'Lưu thất bại');
    },
  });

  const windowErr = action === 'time_window' ? datetimeRangeError(windowStart, windowEnd) : null;

  const onSave = () => {
    if (windowErr) {
      showToast('err', windowErr);
      return;
    }
    const body: ScopeAutoState = { auto_action: action };
    if (action === 'time_window') {
      body.window_start = windowStart;
      body.window_end = windowEnd;
    }
    save.mutate(body);
  };

  const startEdit = () => {
    setAction(normalizeAction(initial.auto_action));
    setWindowStart(fromApiDateTime(initial.window_start, 8, 0) || defaults.windowStart);
    setWindowEnd(fromApiDateTime(initial.window_end, 18, 0) || defaults.windowEnd);
    setEditing(true);
  };

  const cancelEdit = () => {
    setAction(normalizeAction(initial.auto_action));
    setWindowStart(fromApiDateTime(initial.window_start, 8, 0) || defaults.windowStart);
    setWindowEnd(fromApiDateTime(initial.window_end, 18, 0) || defaults.windowEnd);
    setEditing(false);
  };

  const savedAction = normalizeAction(initial.auto_action);
  const savedLabel = actionDisplayLabel(savedAction);
  const scopeHint =
    level === 'product'
      ? 'Chế độ xử lý tự động cho toàn bộ dịch vụ'
      : 'Chế độ xử lý tự động cho SKU này';
  const saveHint =
    level === 'product' ? 'Lưu chế độ Auto cho dịch vụ' : 'Lưu chế độ Auto cho SKU';

  if (!editing) {
    return (
      <div className="scope-auto-editor scope-auto-editor--compact">
        <span className="scope-auto-editor__summary-label muted">Chế độ BT / Routing</span>
        <div className="scope-auto-editor__summary-actions">
          <span
            className="scope-auto-editor__summary-value"
            title={
              level === 'product'
                ? `${savedLabel} — chế độ mặc định cho toàn dịch vụ (SKU có thể cấu hình riêng)`
                : (summaryTitle(initial) ?? undefined)
            }
          >
            {savedLabel}
          </span>
          <button
            type="button"
            className="btn btn--ghost btn--xs scope-auto-editor__more"
            aria-label="Chỉnh sửa chế độ bảo trì / routing"
            title="Chỉnh sửa"
            onClick={startEdit}
          >
            ⋯
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="scope-auto-editor scope-auto-editor--editing">
      <label className="scope-auto-editor__field">
        <span className="scope-auto-editor__label muted">Chế độ BT / Routing</span>
        <select
          className="scope-auto-editor__select"
          value={action}
          title={scopeHint}
          onChange={(e) => setAction(e.target.value)}
        >
          {AUTO_ACTIONS.map((a) => (
            <option key={a.value} value={a.value}>
              {a.label}
            </option>
          ))}
        </select>
      </label>

      {action === 'time_window' && (
        <div className="scope-auto-editor__window">
          <DateTimeLocalField
            label="Từ"
            compact
            className="scope-auto-editor__datetime"
            value={windowStart}
            onChange={setWindowStart}
          />
          <DateTimeLocalField
            label="Đến"
            compact
            className="scope-auto-editor__datetime"
            value={windowEnd}
            onChange={setWindowEnd}
          />
          {windowErr && <span className="scope-auto-editor__error">{windowErr}</span>}
        </div>
      )}

      <div className="scope-auto-editor__actions">
        <button
          type="button"
          className="btn btn--primary btn--xs"
          disabled={save.isPending || Boolean(windowErr)}
          title={saveHint}
          onClick={onSave}
        >
          {save.isPending ? '…' : 'Lưu'}
        </button>
        <button
          type="button"
          className="btn btn--ghost btn--xs"
          disabled={save.isPending}
          onClick={cancelEdit}
        >
          Hủy
        </button>
      </div>
    </div>
  );
}
