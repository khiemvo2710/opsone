import { useEffect, useState } from 'react';
import type { DashboardOverviewRow } from '../types/api';
import { DateTimeLocalField } from './DateTimeLocalField';
import {
  maintenanceActiveLabel,
  maintenanceWindowError,
  maintenanceWindowFromISO,
  maintenanceWindowISO,
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
  const [draft, setDraft] = useState(() =>
    m ? maintenanceWindowFromISO(m.starts_at, m.ends_at) : { startsAt: '', endsAt: '' },
  );

  useEffect(() => {
    if (!m) return;
    setDraft(maintenanceWindowFromISO(m.starts_at, m.ends_at));
    setEditing(false);
  }, [m?.starts_at, m?.ends_at]);

  if (!m) return <span className="muted">—</span>;

  const windowErr = editing ? maintenanceWindowError(draft.startsAt, draft.endsAt) : null;
  const iso = maintenanceWindowISO(draft.startsAt, draft.endsAt);

  const startEdit = () => {
    setDraft(maintenanceWindowFromISO(m.starts_at, m.ends_at));
    setEditing(true);
  };

  const cancelEdit = () => {
    setEditing(false);
    setDraft(maintenanceWindowFromISO(m.starts_at, m.ends_at));
  };

  return (
    <div className="maint-active-bar">
      <span className="maint-active-bar__label" title={m.reason}>
        {maintenanceActiveLabel(m.starts_at, m.ends_at)}
      </span>
      {!editing ? (
        <div className="maint-active-bar__actions">
          {onReopen && (
            <button
              type="button"
              className="btn btn--ghost btn--xs"
              disabled={busy}
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
              Gia hạn
            </button>
          )}
        </div>
      ) : (
        <div className="maint-active-bar__edit">
          <DateTimeLocalField
            label="Bắt đầu"
            className="maint-active-bar__field"
            value={draft.startsAt}
            disabled={busy}
            onChange={(startsAt) => setDraft((prev) => ({ ...prev, startsAt }))}
          />
          <DateTimeLocalField
            label="Kết thúc"
            className="maint-active-bar__field"
            value={draft.endsAt}
            disabled={busy}
            onChange={(endsAt) => setDraft((prev) => ({ ...prev, endsAt }))}
          />
          <div className="maint-active-bar__edit-actions">
            <button
              type="button"
              className="btn btn--primary btn--xs"
              disabled={busy || Boolean(windowErr)}
              onClick={() =>
                onExtend?.({
                  productCode: row.product_code,
                  skuCode: row.sku_code,
                  startsAt: iso.starts_at,
                  endsAt: iso.ends_at,
                })
              }
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
          {windowErr && <span className="maint-active-bar__error">{windowErr}</span>}
        </div>
      )}
    </div>
  );
}
