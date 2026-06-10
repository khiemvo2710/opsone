import { useEffect, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import { useToast } from '../context/ToastContext';
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

export function ScopeAutoEditor({ productCode, skuCode, initial }: Props) {
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

  const save = useMutation({
    mutationFn: (body: ScopeAutoState) =>
      api<ScopeAutoState>(scopeAutoPath(productCode, skuCode), {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      setEditing(false);
      void qc.invalidateQueries({ queryKey: ['dashboard-overview'] });
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

  if (!editing) {
    return (
      <div className="scope-auto-editor scope-auto-editor--compact">
        <div className="scope-auto-editor__summary" title={summaryTitle(initial)}>
          <span className="scope-auto-editor__summary-label muted">Chế độ BT / Routing</span>
          <span className="scope-auto-editor__summary-value">{savedLabel}</span>
        </div>
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
    );
  }

  return (
    <div className="scope-auto-editor scope-auto-editor--editing">
      <label className="scope-auto-editor__field">
        <span className="scope-auto-editor__label muted">Chế độ BT / Routing</span>
        <select
          className="scope-auto-editor__select"
          value={action}
          title="Chế độ xử lý tự động cho SKU này"
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
          title="Lưu chế độ Auto cho SKU"
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
