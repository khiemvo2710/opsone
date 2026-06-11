import { useEffect, useState } from 'react';
import type { ProductMaintenancePayload, ScopeMaintenancePayload } from './ServiceOverviewTable';
import { DateTimeLocalField } from './DateTimeLocalField';
import { useMaintenanceDefaultDurationMin } from '../hooks/useMaintenanceDefaultDurationMin';
import {
  defaultMaintenanceWindow,
  maintenanceWindowError,
  maintenanceWindowISO,
  manualServiceMaintenanceReason,
} from '../utils/maintenanceWindow';

interface Props {
  productCode: string;
  /** Bảo trì một SKU (mặc định). */
  skuCode?: string;
  /** Bảo trì toàn bộ SKU của sản phẩm — dùng cùng onStartProduct. */
  skuCodes?: string[];
  busy?: boolean;
  onStart?: (payload: ScopeMaintenancePayload) => void;
  onStartProduct?: (payload: ProductMaintenancePayload) => void;
}

export function ServiceMaintenanceButton({
  productCode,
  skuCode,
  skuCodes,
  busy = false,
  onStart,
  onStartProduct,
}: Props) {
  const durationMin = useMaintenanceDefaultDurationMin();
  const isProductScope = Boolean(skuCodes && skuCodes.length > 0 && onStartProduct);
  const targetSkus = isProductScope ? skuCodes! : skuCode != null ? [skuCode] : [];
  const [open, setOpen] = useState(false);
  const [window, setWindow] = useState(() => defaultMaintenanceWindow(durationMin));

  useEffect(() => {
    if (!busy) {
      setOpen(false);
    }
  }, [busy]);

  const windowErr = open ? maintenanceWindowError(window.startsAt, window.endsAt) : null;
  const iso = windowErr ? null : maintenanceWindowISO(window.startsAt, window.endsAt);

  const startEdit = () => {
    setWindow(defaultMaintenanceWindow(durationMin));
    setOpen(true);
  };

  const cancelEdit = () => {
    setOpen(false);
    setWindow(defaultMaintenanceWindow(durationMin));
  };

  const confirm = () => {
    if (!iso || targetSkus.length === 0) return;
    const reason = manualServiceMaintenanceReason();
    if (isProductScope) {
      onStartProduct!({
        productCode,
        skuCodes: targetSkus,
        reason,
        startsAt: iso.starts_at,
        endsAt: iso.ends_at,
      });
      return;
    }
    onStart!({
      productCode,
      skuCode: targetSkus[0],
      reason,
      startsAt: iso.starts_at,
      endsAt: iso.ends_at,
    });
  };

  const triggerTitle = isProductScope
    ? `Lên lịch bảo trì toàn bộ ${targetSkus.length} SKU`
    : 'Lên lịch bảo trì SKU';

  if (!open) {
    return (
      <div className={`service-maint-btn${isProductScope ? ' service-maint-btn--product' : ''}`}>
        <button
          type="button"
          className="btn btn--ghost btn--xs service-maint-btn__trigger"
          disabled={busy || targetSkus.length === 0}
          title={targetSkus.length === 0 ? 'Tất cả SKU đang bảo trì' : triggerTitle}
          onClick={startEdit}
        >
          Bảo trì dịch vụ
        </button>
      </div>
    );
  }

  return (
    <div
      className={`service-maint-btn service-maint-btn--open${isProductScope ? ' service-maint-btn--product' : ''}`}
    >
      <span className="service-maint-btn__label muted">
        {isProductScope ? `Bảo trì dịch vụ (${targetSkus.length} SKU)` : 'Bảo trì dịch vụ'}
      </span>
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
