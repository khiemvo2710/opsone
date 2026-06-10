import { useEffect, useMemo, useState } from 'react';
import type { DashboardOverviewRow } from '../types/api';
import { DateTimeLocalField } from './DateTimeLocalField';
import {
  maintenanceWindowError,
  maintenanceWindowFromISO,
  maintenanceWindowISO,
  maintenanceWindowUnchanged,
} from '../utils/maintenanceWindow';

export interface ScopeMaintenanceActionPayload {
  productCode: string;
  skuCode: string;
  maintenanceIds?: string[];
}

export interface ExtendMaintenancePayload extends ScopeMaintenanceActionPayload {
  startsAt: string;
  endsAt: string;
}

interface Props {
  row: DashboardOverviewRow;
  busy?: boolean;
  onReopen?: (payload: ScopeMaintenanceActionPayload) => void;
  onExtend?: (payload: ExtendMaintenancePayload) => void;
}

export function ActiveMaintenanceCell({ row, busy = false, onReopen, onExtend }: Props) {
  const m = row.maintenance;
  const [editing, setEditing] = useState(false);
  const baseDraft = useMemo(
    () =>
      m
        ? maintenanceWindowFromISO(String(m.starts_at ?? ''), String(m.ends_at ?? ''))
        : { startsAt: '', endsAt: '' },
    [m?.starts_at, m?.ends_at],
  );
  const [editDraft, setEditDraft] = useState(baseDraft);
  const [extendHint, setExtendHint] = useState<string | null>(null);

  useEffect(() => {
    if (!m) {
      setEditing(false);
      return;
    }
    setEditDraft(baseDraft);
    setEditing(false);
    setExtendHint(null);
  }, [m, baseDraft]);

  if (!m) return <span className="muted">—</span>;

  const draft = editing ? editDraft : baseDraft;
  const windowErr = editing ? maintenanceWindowError(draft.startsAt, draft.endsAt) : null;
  const extendErr = windowErr ?? extendHint;

  const startEdit = () => {
    setEditDraft(baseDraft);
    setExtendHint(null);
    setEditing(true);
  };

  const cancelEdit = () => {
    setEditing(false);
    setEditDraft(baseDraft);
    setExtendHint(null);
  };

  const saveExtend = () => {
    if (maintenanceWindowUnchanged(draft, baseDraft)) {
      setExtendHint('Thời gian bảo trì không thay đổi');
      return;
    }
    setExtendHint(null);
    const iso = maintenanceWindowISO(draft.startsAt, draft.endsAt);
    if (!iso) return;
    onExtend?.({
      productCode: row.product_code,
      skuCode: row.sku_code,
      startsAt: iso.starts_at,
      endsAt: iso.ends_at,
    });
  };

  return (
    <div className="maint-active-bar">
      {!editing ? (
        <div className="maint-active-bar__actions">
          {onReopen && (
            <button
              type="button"
              className="btn btn--primary btn--xs"
              disabled={busy}
              title="Kết thúc bảo trì và trả routing về baseline biz"
              onClick={() =>
                onReopen({
                  productCode: row.product_code,
                  skuCode: row.sku_code,
                  maintenanceIds: m.maintenance_ids,
                })
              }
            >
              {busy ? '…' : 'Mở lại dịch vụ'}
            </button>
          )}
          {onExtend && (
            <button
              type="button"
              className="btn btn--ghost btn--xs"
              disabled={busy}
              onClick={startEdit}
            >
              Gia hạn bảo trì
            </button>
          )}
        </div>
      ) : (
        <div className="maint-active-bar__edit">
          <DateTimeLocalField
            label="Từ"
            className="maint-active-bar__field"
            value={draft.startsAt}
            disabled={busy}
            onChange={(startsAt) => {
              setExtendHint(null);
              setEditDraft((prev) => ({ ...prev, startsAt }));
            }}
          />
          <DateTimeLocalField
            label="Đến"
            className="maint-active-bar__field"
            value={draft.endsAt}
            disabled={busy}
            onChange={(endsAt) => {
              setExtendHint(null);
              setEditDraft((prev) => ({ ...prev, endsAt }));
            }}
          />
          <div className="maint-active-bar__edit-actions">
            <button
              type="button"
              className="btn btn--primary btn--xs"
              disabled={busy || Boolean(windowErr)}
              onClick={saveExtend}
            >
              {busy ? '…' : 'Lưu'}
            </button>
            <button
              type="button"
              className="btn btn--ghost btn--xs"
              disabled={busy}
              onClick={cancelEdit}
            >
              Hủy
            </button>
          </div>
          {extendErr && <span className="maint-active-bar__error">{extendErr}</span>}
        </div>
      )}
    </div>
  );
}
