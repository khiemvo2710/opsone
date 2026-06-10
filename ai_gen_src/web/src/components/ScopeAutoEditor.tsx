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
  { value: 'time_window', label: 'Theo khung giờ' },
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

export function ScopeAutoEditor({ productCode, skuCode, initial }: Props) {
  const qc = useQueryClient();
  const { showToast } = useToast();
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
  }, [initial.auto_action, initial.window_start, initial.window_end]);

  const save = useMutation({
    mutationFn: (body: ScopeAutoState) =>
      api<ScopeAutoState>(scopeAutoPath(productCode, skuCode), {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
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

  return (
    <div className="scope-auto-editor">
      <label className="scope-auto-editor__field">
        <span className="scope-auto-editor__label muted">Chế độ</span>
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

      <button
        type="button"
        className="btn btn--primary btn--xs scope-auto-editor__save"
        disabled={save.isPending || Boolean(windowErr)}
        title="Lưu chế độ Auto cho SKU"
        onClick={onSave}
      >
        {save.isPending ? '…' : 'Lưu'}
      </button>
    </div>
  );
}
