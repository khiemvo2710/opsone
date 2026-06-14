import { useEffect, useMemo, useState } from 'react';
import type {
  ProductExtendMaintenancePayload,
  ProductReopenMaintenancePayload,
} from './ServiceOverviewTable';
import { DateTimeLocalField } from './DateTimeLocalField';
import {
  maintenanceWindowError,
  maintenanceWindowFromISO,
  maintenanceWindowISO,
  maintenanceWindowUnchanged,
} from '../utils/maintenanceWindow';

export interface ProductMaintainedScope {
  skuCode: string;
  maintenanceIds?: string[];
  startsAt: string;
  endsAt: string;
}

interface Props {
  productCode: string;
  scopes: ProductMaintainedScope[];
  busy?: boolean;
  onReopenProduct?: (payload: ProductReopenMaintenancePayload) => void;
  onExtendProduct?: (payload: ProductExtendMaintenancePayload) => void;
}

function initialExtendDraft(scopes: ProductMaintainedScope[]): { startsAt: string; endsAt: string } {
  const first = scopes[0];
  if (!first) return { startsAt: '', endsAt: '' };
  return maintenanceWindowFromISO(first.startsAt, first.endsAt);
}

export function ProductMaintenanceActions({
  productCode,
  scopes,
  busy = false,
  onReopenProduct,
  onExtendProduct,
}: Props) {
  const [editing, setEditing] = useState(false);
  // Use primitive values as deps so polling (new array references) doesn't close the form.
  const firstStartsAt = scopes[0]?.startsAt ?? '';
  const firstEndsAt = scopes[0]?.endsAt ?? '';
  const baseDraft = useMemo(
    () => initialExtendDraft(scopes),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [firstStartsAt, firstEndsAt, scopes.length],
  );
  const [editDraft, setEditDraft] = useState(baseDraft);
  const [extendHint, setExtendHint] = useState<string | null>(null);

  useEffect(() => {
    setEditDraft(baseDraft);
    setEditing(false);
    setExtendHint(null);
  }, [baseDraft]);

  if (scopes.length === 0) return null;

  const draft = editing ? editDraft : baseDraft;
  const windowErr = editing ? maintenanceWindowError(draft.startsAt, draft.endsAt) : null;
  const extendErr = windowErr ?? extendHint;

  const reopenAll = () => {
    onReopenProduct?.({
      productCode,
      scopes: scopes.map((s) => ({
        skuCode: s.skuCode,
        maintenanceIds: s.maintenanceIds,
      })),
    });
  };

  const saveExtend = () => {
    if (maintenanceWindowUnchanged(draft, baseDraft)) {
      setExtendHint('Thời gian bảo trì không thay đổi');
      return;
    }
    setExtendHint(null);
    const iso = maintenanceWindowISO(draft.startsAt, draft.endsAt);
    if (!iso) return;
    onExtendProduct?.({
      productCode,
      scopes: scopes.map((s) => ({
        skuCode: s.skuCode,
        startsAt: iso.starts_at,
        endsAt: iso.ends_at,
      })),
    });
  };

  return (
    <div className="product-maint-actions">
      {!editing ? (
        <div className="product-maint-actions__buttons">
          {onReopenProduct && (
            <button
              type="button"
              className="btn btn--primary btn--xs product-maint-actions__btn"
              disabled={busy}
              title={`Mở lại dịch vụ toàn bộ ${scopes.length} SKU đang bảo trì`}
              onClick={reopenAll}
            >
              {busy ? '…' : 'Mở lại dịch vụ'}
            </button>
          )}
          {onExtendProduct && (
            <button
              type="button"
              className="btn btn--ghost btn--xs product-maint-actions__btn"
              disabled={busy}
              onClick={() => {
                setEditDraft(baseDraft);
                setExtendHint(null);
                setEditing(true);
              }}
            >
              Gia hạn bảo trì
            </button>
          )}
        </div>
      ) : (
        <div className="product-maint-actions__extend">
          <span className="product-maint-actions__hint muted">
            Gia hạn {scopes.length} SKU
          </span>
          <DateTimeLocalField
            label="Từ"
            compact
            className="product-maint-actions__field"
            value={draft.startsAt}
            disabled={busy}
            onChange={(startsAt) => {
              setExtendHint(null);
              setEditDraft((prev) => ({ ...prev, startsAt }));
            }}
          />
          <DateTimeLocalField
            label="Đến"
            compact
            className="product-maint-actions__field"
            value={draft.endsAt}
            disabled={busy}
            onChange={(endsAt) => {
              setExtendHint(null);
              setEditDraft((prev) => ({ ...prev, endsAt }));
            }}
          />
          <div className="product-maint-actions__edit-actions">
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
              onClick={() => {
                setEditing(false);
                setEditDraft(baseDraft);
                setExtendHint(null);
              }}
            >
              Hủy
            </button>
          </div>
          {extendErr && <span className="product-maint-actions__error">{extendErr}</span>}
        </div>
      )}
    </div>
  );
}
