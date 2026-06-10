import { useEffect, useState } from 'react';
import type { ScopeMaintenancePayload } from './ServiceOverviewTable';
import { DateTimeLocalField } from './DateTimeLocalField';
import {
  defaultMaintenanceWindow,
  maintenanceWindowError,
  maintenanceWindowISO,
  manualServiceMaintenanceReason,
} from '../utils/maintenanceWindow';

interface Props {
  productCode: string;
  skuCode: string;
  busy?: boolean;
  onStart: (payload: ScopeMaintenancePayload) => void;
}

export function ServiceMaintenanceButton({ productCode, skuCode, busy = false, onStart }: Props) {
  const [open, setOpen] = useState(false);
  const [window, setWindow] = useState(defaultMaintenanceWindow);

  useEffect(() => {
    if (!busy) {
      setOpen(false);
    }
  }, [busy]);

  const windowErr = open ? maintenanceWindowError(window.startsAt, window.endsAt) : null;
  const iso = windowErr ? null : maintenanceWindowISO(window.startsAt, window.endsAt);

  const startEdit = () => {
    setWindow(defaultMaintenanceWindow());
    setOpen(true);
  };

  const cancelEdit = () => {
    setOpen(false);
    setWindow(defaultMaintenanceWindow());
  };

  const confirm = () => {
    if (!iso) return;
    onStart({
      productCode,
      skuCode,
      reason: manualServiceMaintenanceReason(),
      startsAt: iso.starts_at,
      endsAt: iso.ends_at,
    });
  };

  if (!open) {
    return (
      <div className="service-maint-btn">
        <button
          type="button"
          className="btn btn--ghost btn--xs service-maint-btn__trigger"
          disabled={busy}
          title="Lên lịch bảo trì SKU"
          onClick={startEdit}
        >
          Bảo trì dịch vụ
        </button>
      </div>
    );
  }

  return (
    <div className="service-maint-btn service-maint-btn--open">
      <span className="service-maint-btn__label muted">Bảo trì dịch vụ</span>
      <div className="service-maint-btn__window">
        <DateTimeLocalField
          label="Từ"
          compact
          className="service-maint-btn__field"
          value={window.startsAt}
          disabled={busy}
          onChange={(startsAt) => setWindow((prev) => ({ ...prev, startsAt }))}
        />
        <DateTimeLocalField
          label="Đến"
          compact
          className="service-maint-btn__field"
          value={window.endsAt}
          disabled={busy}
          onChange={(endsAt) => setWindow((prev) => ({ ...prev, endsAt }))}
        />
      </div>
      {windowErr && <span className="service-maint-btn__error">{windowErr}</span>}
      <div className="service-maint-btn__actions">
        <button
          type="button"
          className="btn btn--primary btn--xs"
          disabled={busy || Boolean(windowErr) || !iso}
          title="Áp dụng bảo trì theo khung giờ đã chọn"
          onClick={confirm}
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
    </div>
  );
}
