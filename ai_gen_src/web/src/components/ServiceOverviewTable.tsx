import { useEffect, useState, type ReactNode } from 'react';
import type { DashboardOverviewRow } from '../types/api';
import { HealthBadge } from './HealthBadge';
import { ProductThresholdEditor } from './ProductThresholdEditor';
import { ScopeAutoEditor } from './ScopeAutoEditor';
import { ProviderMetricCell } from './ProviderMetricCell';
import { effectiveRowHealth } from '../utils/dashboardHealth';
import { isProviderUnderMaintenance } from '../utils/maintenanceDisplay';
import type { ProductThreshold } from '../types/api';
import {
  activeRoutingProviders,
  initialRoutingPct,
  isRoutingProviderSupported,
  routingPctFieldInvalid,
  routingPctMapsEqual,
  routingPctValidationError,
  ROUTING_PCT_MAX,
} from './RoutingPctEditor';
import {
  defaultMaintenanceWindow,
  isSkuWideMaintenance,
  maintenanceSuggestLabel,
  maintenanceWindowError,
  maintenanceWindowISO,
} from '../utils/maintenanceWindow';
import { DateTimeLocalField } from './DateTimeLocalField';
import {
  ActiveMaintenanceCell,
  type ExtendMaintenancePayload,
  type ScopeMaintenanceActionPayload,
} from './ActiveMaintenanceCell';

const PROVIDERS = ['ESALE', 'IMEDIA', 'SHOPPAY'] as const;

type PlanAction = 'approved' | 'rejected';

interface ProductGroup {
  productCode: string;
  productLabel: string;
  rows: DashboardOverviewRow[];
}

function compareSkuCode(a: string, b: string): number {
  const aNum = /^\d+$/.test(a);
  const bNum = /^\d+$/.test(b);
  if (aNum && bNum) {
    return Number(a) - Number(b);
  }
  return a.localeCompare(b, undefined, { numeric: true });
}

function groupRowsByProduct(rows: DashboardOverviewRow[]): ProductGroup[] {
  const groups: ProductGroup[] = [];
  const index = new Map<string, number>();
  for (const row of rows) {
    const existing = index.get(row.product_code);
    if (existing === undefined) {
      index.set(row.product_code, groups.length);
      groups.push({
        productCode: row.product_code,
        productLabel: row.product_label || row.product_code,
        rows: [row],
      });
    } else {
      groups[existing].rows.push(row);
    }
  }
  for (const group of groups) {
    group.rows.sort((a, b) => compareSkuCode(a.sku_code, b.sku_code));
  }
  return groups;
}

function fmtSku(sku: string): string {
  return sku === '' ? '—' : sku;
}

function planStatusLabel(status: string, localAction?: PlanAction): string {
  if (localAction === 'approved') return 'Đã duyệt';
  if (localAction === 'rejected') return 'Đã từ chối';
  switch (status) {
    case 'pending_approve':
    case 'draft':
      return 'Chờ duyệt';
    case 'executed':
      return 'Đã áp dụng';
    case 'rejected':
      return 'Đã từ chối';
    case 'cancelled':
      return 'Đã hủy';
    default:
      return status;
  }
}

function planBadgeClass(status: string, localAction?: PlanAction): string {
  if (localAction === 'approved' || status === 'executed') return 'badge badge--executed';
  if (localAction === 'rejected' || status === 'rejected' || status === 'cancelled') {
    return 'badge badge--muted';
  }
  return 'badge badge--pending';
}

function isRoutingRejected(
  scopeKey: string,
  plan: DashboardOverviewRow['pending_plan'],
  scopeDone: Record<string, { planId: number; action: PlanAction }>,
  planActions: Record<number, PlanAction>,
): boolean {
  if (scopeDone[scopeKey]?.action === 'rejected') return true;
  const planId = plan?.id;
  return planId != null && planActions[planId] === 'rejected';
}

function isMaintenanceRejected(
  scopeKey: string,
  maint: DashboardOverviewRow['pending_maintenance'],
  maintScopeDone: Record<string, { maintId: number; action: PlanAction }>,
  maintActions: Record<number, PlanAction>,
): boolean {
  if (maintScopeDone[scopeKey]?.action === 'rejected') return true;
  const maintId = maint?.id;
  return maintId != null && maintActions[maintId] === 'rejected';
}

function countSubRows(
  row: DashboardOverviewRow,
  scopeKey: string,
  scopeDone: Record<string, { planId: number; action: PlanAction }>,
  maintScopeDone: Record<string, { maintId: number; action: PlanAction }>,
  planActions: Record<number, PlanAction>,
  maintActions: Record<number, PlanAction>,
): number {
  let n = 0;
  if (row.pending_plan && !isRoutingRejected(scopeKey, row.pending_plan, scopeDone, planActions)) {
    n += 1;
  }
  if (
    row.pending_maintenance &&
    !isMaintenanceRejected(scopeKey, row.pending_maintenance, maintScopeDone, maintActions)
  ) {
    n += 1;
  }
  return n;
}

function groupBodyRowCount(
  group: ProductGroup,
  scopeDone: Record<string, { planId: number; action: PlanAction }>,
  maintScopeDone: Record<string, { maintId: number; action: PlanAction }>,
  planActions: Record<number, PlanAction>,
  maintActions: Record<number, PlanAction>,
  restoreScopeKey: string | null,
): number {
  let count = 0;
  for (const row of group.rows) {
    const scopeKey = `${row.product_code}:${row.sku_code}`;
    count += 1 + countSubRows(row, scopeKey, scopeDone, maintScopeDone, planActions, maintActions);
    if (restoreScopeKey === scopeKey) {
      count += 1;
    }
  }
  return count;
}

export interface ApproveRoutingPayload {
  planId: number;
  routing: Record<string, number>;
}

export interface ScopeRoutingPayload {
  productCode: string;
  skuCode: string;
  routing: Record<string, number>;
  plan: NonNullable<DashboardOverviewRow['pending_plan']>['plan'];
}

export interface ScopeRoutingApplyPayload {
  productCode: string;
  skuCode: string;
  routing: Record<string, number>;
}

export interface ScopeMaintenancePayload {
  productCode: string;
  skuCode: string;
  reason?: string;
  providerCode?: string;
  startsAt?: string;
  endsAt?: string;
}

export interface MaintenanceApprovePayload {
  recommendationId: number;
  startsAt?: string;
  endsAt?: string;
}

type DraftKey = number | string;

function planDraftKey(planId: number | undefined, scopeKey: string): DraftKey {
  if (planId != null && planId > 0) return planId;
  return `scope:${scopeKey}`;
}

interface Props {
  rows: DashboardOverviewRow[];
  providers?: string[];
  thresholdsByProduct?: Record<string, ProductThreshold>;
  refreshing?: boolean;
  onApprove?: (payload: ApproveRoutingPayload) => void;
  onReject?: (planId: number) => void;
  onApproveScope?: (payload: ScopeRoutingPayload) => void;
  onRejectScope?: (payload: ScopeRoutingPayload) => void;
  onApplyScopeRouting?: (payload: ScopeRoutingApplyPayload) => void;
  onApproveMaintenance?: (payload: MaintenanceApprovePayload) => void;
  onRejectMaintenance?: (recommendationId: number) => void;
  onApproveScopeMaintenance?: (payload: ScopeMaintenancePayload) => void;
  onRejectScopeMaintenance?: (payload: ScopeMaintenancePayload) => void;
  onReopenMaintenance?: (payload: ScopeMaintenanceActionPayload) => void;
  onExtendMaintenance?: (payload: ExtendMaintenancePayload) => void;
  busyPlanId?: number | null;
  busyMaintId?: number | null;
  busyScopeKey?: string | null;
  planActions?: Record<number, PlanAction>;
  maintActions?: Record<number, PlanAction>;
  scopeDone?: Record<string, { planId: number; action: PlanAction }>;
  maintScopeDone?: Record<string, { maintId: number; action: PlanAction }>;
  updatedAt?: string;
}

export function ServiceOverviewTable({
  rows,
  providers = [...PROVIDERS],
  thresholdsByProduct = {},
  refreshing = false,
  onApprove,
  onReject,
  onApproveScope,
  onRejectScope,
  onApplyScopeRouting,
  onApproveMaintenance,
  onRejectMaintenance,
  onApproveScopeMaintenance,
  onRejectScopeMaintenance,
  onReopenMaintenance,
  onExtendMaintenance,
  busyPlanId,
  busyMaintId,
  busyScopeKey,
  planActions = {},
  maintActions = {},
  scopeDone = {},
  maintScopeDone = {},
  updatedAt,
}: Props) {
  const [draftRouting, setDraftRouting] = useState<Record<DraftKey, Record<string, number>>>({});
  const [restoreScopeKey, setRestoreScopeKey] = useState<string | null>(null);
  const [restoreDraft, setRestoreDraft] = useState<Record<string, Record<string, number>>>({});
  const [maintWindowDraft, setMaintWindowDraft] = useState<
    Record<string, { startsAt: string; endsAt: string }>
  >({});

  useEffect(() => {
    if (!busyScopeKey) {
      setRestoreScopeKey(null);
    }
  }, [busyScopeKey]);

  useEffect(() => {
    setMaintWindowDraft((prev) => {
      let changed = false;
      const next = { ...prev };
      for (const row of rows) {
        if (!row.pending_maintenance) continue;
        const scopeKey = `${row.product_code}:${row.sku_code}`;
        if (next[scopeKey]) continue;
        next[scopeKey] = defaultMaintenanceWindow();
        changed = true;
      }
      return changed ? next : prev;
    });
  }, [rows, updatedAt]);

  useEffect(() => {
    setDraftRouting((prev) => {
      let changed = false;
      const next = { ...prev };
      for (const row of rows) {
        const plan = row.pending_plan;
        if (!plan?.plan?.proposed_pct) continue;
        const scopeKey = `${row.product_code}:${row.sku_code}`;
        const dk = planDraftKey(plan.id, scopeKey);
        const isSuggested = Boolean(plan.suggested || plan.id === 0);
        if (
          !isSuggested &&
          plan.status !== 'pending_approve' &&
          plan.status !== 'draft'
        ) {
          continue;
        }
        const fresh = initialRoutingPct(plan.plan.proposed_pct, row.routing_pct, providers);
        const existing = next[dk];
        if (!existing || !routingPctMapsEqual(existing, fresh, providers)) {
          next[dk] = fresh;
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [rows, providers, updatedAt]);

  return (
    <div className="overview-table-wrap">
      {updatedAt && (
        <p className="overview-table__meta muted">
          Cập nhật: {new Date(updatedAt).toLocaleString('vi-VN')} · tự làm mới mỗi 60 giây
          {refreshing ? ' · đang cập nhật…' : ''}
        </p>
      )}
      <div className="table-scroll">
        <table className="data-table overview-table">
          <colgroup>
            <col className="overview-table__col-product" />
            <col className="overview-table__col-sku" />
            <col className="overview-table__col-tt" />
            {providers.map((p) => (
              <col key={p} className="overview-table__col-provider" />
            ))}
            <col className="overview-table__col-auto" />
            <col className="overview-table__col-maint" />
          </colgroup>
          {groupRowsByProduct(rows).map((group) => {
              const planLabelColSpan = 2;
              const tableColCount = planLabelColSpan + providers.length + 2;
              const bodyRowCount = groupBodyRowCount(
                group,
                scopeDone,
                maintScopeDone,
                planActions,
                maintActions,
                restoreScopeKey,
              );
              const productThreshold = thresholdsByProduct[group.productCode];
              return (
                <tbody key={group.productCode} className="overview-table__product-group">
                  <tr className="overview-table__threshold-row">
                    <ProductThresholdEditor
                      productCode={group.productCode}
                      providers={providers}
                      threshold={thresholdsByProduct[group.productCode]}
                    />
                  </tr>
                  {group.rows.flatMap((row, rowIdx) => {
                const scopeKey = `${row.product_code}:${row.sku_code}`;
                const done = scopeDone[scopeKey];
                const maintDone = maintScopeDone[scopeKey];
                const plan = row.pending_plan;
                const planId = plan?.id;
                const localAction = planId != null ? planActions[planId] : undefined;
                const routingRejected = isRoutingRejected(scopeKey, plan, scopeDone, planActions);
                const isSuggested = Boolean(plan?.suggested || planId === 0);
                const draftKey = planDraftKey(planId, scopeKey);
                const isPending =
                  plan &&
                  !routingRejected &&
                  !localAction &&
                  !done &&
                  (isSuggested ||
                    plan.status === 'pending_approve' ||
                    plan.status === 'draft');
                const maintRejected = isMaintenanceRejected(
                  scopeKey,
                  row.pending_maintenance,
                  maintScopeDone,
                  maintActions,
                );
                const isPendingMaint = Boolean(row.pending_maintenance && !maintDone && !maintRejected);
                const canActReal =
                  isPending && !isSuggested && onApprove && onReject && planId != null && planId > 0;
                const canActSuggested =
                  isPending &&
                  isSuggested &&
                  onApproveScope &&
                  onRejectScope &&
                  plan?.plan != null;
                const canAct = canActReal || canActSuggested;
                const scopeBusy = busyScopeKey === scopeKey;
                const reason = plan?.plan?.reason_vi ?? '';
                const planJson = plan?.plan;
                const isProviderSupported = (p: string) =>
                  isRoutingProviderSupported(p, row.routing_pct, planJson);
                const activeProviders = activeRoutingProviders(providers, row.routing_pct, planJson);
                const draft =
                  plan
                    ? draftRouting[draftKey] ??
                      initialRoutingPct(plan.plan?.proposed_pct, row.routing_pct, providers)
                    : {};
                const routingError = routingPctValidationError(draft, activeProviders);
                const routingValid = routingError === null;
                const displayPct =
                  plan?.plan?.proposed_pct ?? plan?.plan?.current_pct ?? row.routing_pct ?? {};
                const isFirstInGroup = rowIdx === 0;
                const showPlanRow =
                  countSubRows(row, scopeKey, scopeDone, maintScopeDone, planActions, maintActions) > 0;
                const isRestoring = restoreScopeKey === scopeKey;
                const restoreValues =
                  restoreDraft[scopeKey] ?? initialRoutingPct(undefined, row.routing_pct, providers);
                const restoreError = isRestoring
                  ? routingPctValidationError(restoreValues, activeProviders)
                  : null;
                const restoreValid = restoreError === null;

                const buildRestoreRow = () => {
                  if (!isRestoring || !onApplyScopeRouting) return null;
                  return (
                    <tr
                      key={`${row.product_code}-${row.sku_code}-restore`}
                      className="overview-table__plan-row overview-table__restore-row"
                    >
                      <td colSpan={planLabelColSpan} className="overview-table__plan-label">
                        Mở lại provider
                      </td>
                      {providers.map((p) => {
                        const supported = isProviderSupported(p);
                        const invalid = supported && routingPctFieldInvalid(p, restoreValues, providers);
                        return (
                          <td key={p} className="mono nowrap overview-table__plan-pct">
                            {supported ? (
                              <span className="overview-table__plan-pct-wrap">
                                <input
                                  type="number"
                                  className={`overview-table__plan-input overview-table__plan-input--stepper${invalid ? ' overview-table__plan-input--bad' : ''}`}
                                  min={0}
                                  max={ROUTING_PCT_MAX}
                                  step={1}
                                  disabled={scopeBusy}
                                  value={restoreValues[p] ?? ''}
                                  onChange={(e) => {
                                    const raw = e.target.value;
                                    const n = raw === '' ? 0 : Number.parseFloat(raw);
                                    setRestoreDraft((prev) => ({
                                      ...prev,
                                      [scopeKey]: {
                                        ...restoreValues,
                                        [p]: Number.isFinite(n) ? Math.round(n) : 0,
                                      },
                                    }));
                                  }}
                                />
                                <span className="muted">%</span>
                              </span>
                            ) : (
                              <span className="muted">—</span>
                            )}
                          </td>
                        );
                      })}
                      <td className="muted">—</td>
                      <td className="overview-table__plan-actions">
                        <div className="overview-table__plan-actions-inner">
                          {restoreError && (
                            <span className="overview-table__plan-error">{restoreError}</span>
                          )}
                          <button
                            type="button"
                            className="btn btn--primary btn--xs"
                            disabled={scopeBusy || !restoreValid}
                            title={restoreValid ? 'Áp dụng routing đã nhập' : (restoreError ?? 'Chưa hợp lệ')}
                            onClick={() => {
                              if (!restoreValid) return;
                              onApplyScopeRouting({
                                productCode: row.product_code,
                                skuCode: row.sku_code,
                                routing: restoreValues,
                              });
                            }}
                          >
                            {scopeBusy ? '…' : 'Lưu'}
                          </button>
                          <button
                            type="button"
                            className="btn btn--ghost btn--xs"
                            disabled={scopeBusy}
                            onClick={() => setRestoreScopeKey(null)}
                          >
                            Hủy
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                };

                const skuHealth = effectiveRowHealth(row, productThreshold);

                const skuRow = (
                  <tr key={`${row.product_code}-${row.sku_code}`}>
                    {isFirstInGroup && (
                      <td
                        rowSpan={bodyRowCount}
                        className="overview-table__product-cell overview-table__product-cell--group"
                      >
                        <span className="overview-table__product-name">{group.productLabel}</span>
                        <span className="muted overview-table__product-code">{group.productCode}</span>
                        {group.rows.length > 1 && (
                          <span className="overview-table__sku-count muted">
                            {group.rows.length} SKU
                          </span>
                        )}
                      </td>
                    )}
                    <td className="nowrap overview-table__sku-cell">{fmtSku(row.sku_code)}</td>
                    <td>
                      <HealthBadge
                        status={skuHealth}
                        label={
                          isPending
                            ? 'Kế hoạch routing chờ duyệt'
                            : isPendingMaint
                              ? 'Đề xuất bảo trì chờ duyệt'
                              : skuHealth === 'red'
                                ? 'Vượt ngưỡng — đủ chu kỳ liên tiếp'
                                : skuHealth === 'yellow'
                                  ? 'Đang theo dõi — vượt ngưỡng'
                                  : undefined
                        }
                        compact
                      />
                    </td>
                    {providers.map((p) => {
                      const supported = isProviderSupported(p);
                      const routingPct = row.routing_pct?.[p];
                      const maintained = isProviderUnderMaintenance(row, p);
                      const inactive = supported && !maintained && (routingPct ?? 0) <= 0;
                      const reopenBlocked =
                        isPending || isPendingMaint || Boolean(row.maintenance) || !onApplyScopeRouting;
                      return (
                        <ProviderMetricCell
                          key={p}
                          provider={p}
                          metrics={row.provider_metrics}
                          routingPct={routingPct}
                          supported={supported}
                          threshold={productThreshold}
                          inactive={inactive}
                          maintained={maintained}
                          reopenDisabled={reopenBlocked || (restoreScopeKey != null && restoreScopeKey !== scopeKey)}
                          reopenBusy={scopeBusy}
                          scopeRestoring={isRestoring}
                          onReopen={
                            inactive && !reopenBlocked && !isRestoring
                              ? () => {
                                  setRestoreScopeKey(scopeKey);
                                  setRestoreDraft((prev) => ({
                                    ...prev,
                                    [scopeKey]: initialRoutingPct(undefined, row.routing_pct, providers),
                                  }));
                                }
                              : undefined
                          }
                        />
                      );
                    })}
                    <td className="overview-table__auto-cell">
                      <ScopeAutoEditor
                        productCode={row.product_code}
                        skuCode={row.sku_code}
                        initial={{
                          auto_action: row.auto_action ?? 'recommend_only',
                          window_start: row.window_start,
                          window_end: row.window_end,
                        }}
                      />
                    </td>
                    <td className="overview-table__maint-cell">
                      <ActiveMaintenanceCell
                        row={row}
                        busy={scopeBusy}
                        onReopen={onReopenMaintenance}
                        onExtend={onExtendMaintenance}
                      />
                    </td>
                  </tr>
                );

                if (!showPlanRow) {
                  const restoreRow = buildRestoreRow();
                  return restoreRow ? [skuRow, restoreRow] : [skuRow];
                }

                const subRows: ReactNode[] = [];

                if (row.pending_maintenance && !maintRejected) {
                  const pm = row.pending_maintenance;
                  const maintId = pm.id;
                  const isSuggestedMaint = Boolean(pm.suggested || maintId == null);
                  const maintLocalAction = maintId != null ? maintActions[maintId] : undefined;
                  const canActMaintReal =
                    maintId != null &&
                    !maintLocalAction &&
                    !maintDone &&
                    onApproveMaintenance &&
                    onRejectMaintenance;
                  const canActMaintSuggested =
                    isSuggestedMaint &&
                    !maintLocalAction &&
                    !maintDone &&
                    onApproveScopeMaintenance &&
                    onRejectScopeMaintenance;
                  const canActMaint = canActMaintReal || canActMaintSuggested;
                  const maintBusy = scopeBusy || (maintId != null && busyMaintId === maintId);
                  const maintWindow = maintWindowDraft[scopeKey] ?? defaultMaintenanceWindow();
                  const maintWindowErr = maintenanceWindowError(maintWindow.startsAt, maintWindow.endsAt);
                  const maintISO = maintenanceWindowISO(maintWindow.startsAt, maintWindow.endsAt);
                  const skuWideMaint =
                    Boolean(pm.scope_level) || isSkuWideMaintenance(pm.reason);
                  subRows.push(
                    <tr
                      key={`${row.product_code}-${row.sku_code}-maint-suggest`}
                      className="overview-table__plan-row overview-table__maint-suggest-row"
                      title={pm.reason}
                    >
                      <td colSpan={tableColCount} className="overview-table__maint-suggest-cell">
                        <div className="maint-suggest-bar">
                          <span className="maint-suggest-bar__label">Đề xuất bảo trì</span>
                          <span className="maint-suggest-bar__message" title={pm.reason}>
                            {maintenanceSuggestLabel(pm)}
                          </span>
                          {canActMaint && (
                            <div className="maint-suggest-bar__window">
                              <DateTimeLocalField
                                label="Bắt đầu"
                                className="maint-suggest-bar__field"
                                value={maintWindow.startsAt}
                                disabled={maintBusy}
                                onChange={(startsAt) =>
                                  setMaintWindowDraft((prev) => ({
                                    ...prev,
                                    [scopeKey]: { ...maintWindow, startsAt },
                                  }))
                                }
                              />
                              <DateTimeLocalField
                                label="Kết thúc"
                                className="maint-suggest-bar__field"
                                value={maintWindow.endsAt}
                                disabled={maintBusy}
                                onChange={(endsAt) =>
                                  setMaintWindowDraft((prev) => ({
                                    ...prev,
                                    [scopeKey]: { ...maintWindow, endsAt },
                                  }))
                                }
                              />
                            </div>
                          )}
                          {canActMaint && (
                            <div className="maint-suggest-bar__actions">
                              <button
                                type="button"
                                className="btn btn--primary btn--xs"
                                disabled={maintBusy || Boolean(maintWindowErr)}
                                title="Lên lịch bảo trì theo khung giờ đã chọn"
                                onClick={() =>
                                  canActMaintReal && maintId != null
                                    ? onApproveMaintenance!({
                                        recommendationId: maintId,
                                        startsAt: maintISO.starts_at,
                                        endsAt: maintISO.ends_at,
                                      })
                                    : onApproveScopeMaintenance!({
                                        productCode: row.product_code,
                                        skuCode: row.sku_code,
                                        reason: pm.reason,
                                        providerCode: skuWideMaint ? undefined : pm.provider_code,
                                        startsAt: maintISO.starts_at,
                                        endsAt: maintISO.ends_at,
                                      })
                                }
                              >
                                {maintBusy ? '…' : 'Duyệt'}
                              </button>
                              <button
                                type="button"
                                className="btn btn--ghost btn--xs"
                                disabled={maintBusy}
                                onClick={() =>
                                  canActMaintReal && maintId != null
                                    ? onRejectMaintenance!(maintId)
                                    : onRejectScopeMaintenance!({
                                        productCode: row.product_code,
                                        skuCode: row.sku_code,
                                        reason: pm.reason,
                                        providerCode: pm.provider_code,
                                      })
                                }
                              >
                                Từ chối
                              </button>
                            </div>
                          )}
                          {(maintLocalAction || maintDone) && maintId != null && (
                            <span
                              className={`plan-row__done ${planBadgeClass('', maintLocalAction ?? maintDone?.action)}`}
                            >
                              {planStatusLabel('', maintLocalAction ?? maintDone?.action)} #{maintId}
                            </span>
                          )}
                          {canActMaint && maintWindowErr && (
                            <span className="maint-suggest-bar__error">{maintWindowErr}</span>
                          )}
                        </div>
                      </td>
                    </tr>,
                  );
                }

                if ((!plan || routingRejected) && subRows.length > 0) {
                  const restoreRow = buildRestoreRow();
                  return restoreRow ? [skuRow, ...subRows, restoreRow] : [skuRow, ...subRows];
                }

                if (!plan || routingRejected) {
                  const restoreRow = buildRestoreRow();
                  return restoreRow ? [skuRow, restoreRow] : [skuRow];
                }

                const planValues = canAct ? draft : displayPct;

                const planRow = (
                  <tr
                    key={`${row.product_code}-${row.sku_code}-plan`}
                    className="overview-table__plan-row"
                    title={reason || undefined}
                  >
                    <td colSpan={planLabelColSpan} className="overview-table__plan-label">
                      {isSuggested ? 'Đề xuất routing' : 'Kế hoạch routing'}
                    </td>
                    {providers.map((p) => {
                      const supported = isProviderSupported(p);
                      const invalid =
                        canAct && supported && routingPctFieldInvalid(p, draft, providers);
                      return (
                        <td key={p} className="mono nowrap overview-table__plan-pct">
                          {supported ? (
                            canAct ? (
                              <span className="overview-table__plan-pct-wrap">
                                <input
                                  type="number"
                                  className={`overview-table__plan-input${invalid ? ' overview-table__plan-input--bad' : ''}`}
                                  min={0}
                                  max={ROUTING_PCT_MAX}
                                  step={1}
                                  disabled={canActSuggested ? scopeBusy : busyPlanId === planId}
                                  value={planValues[p] ?? ''}
                                  onChange={(e) => {
                                    const raw = e.target.value;
                                    const n = raw === '' ? 0 : Number.parseFloat(raw);
                                    setDraftRouting((prev) => ({
                                      ...prev,
                                      [draftKey]: {
                                        ...draft,
                                        [p]: Number.isFinite(n) ? Math.round(n) : 0,
                                      },
                                    }));
                                  }}
                                />
                                <span className="muted">%</span>
                              </span>
                            ) : (
                              <span className="provider-metric-line provider-metric-line--route">
                                {planValues[p] != null ? `${Math.round(planValues[p] * 10) / 10}%` : '—'}
                              </span>
                            )
                          ) : (
                            <span className="muted">—</span>
                          )}
                        </td>
                      );
                    })}
                    <td className="muted">—</td>
                    <td className="overview-table__plan-actions">
                      <div className="overview-table__plan-actions-inner">
                        {plan ? (
                          <>
                            {canAct && (
                              <>
                                {routingError && (
                                  <span className="overview-table__plan-error">{routingError}</span>
                                )}
                                <button
                                  type="button"
                                  className="btn btn--primary btn--xs"
                                  disabled={
                                    (canActSuggested ? scopeBusy : busyPlanId === planId) ||
                                    !routingValid
                                  }
                                  title={
                                    routingValid
                                      ? 'Áp dụng routing đã nhập'
                                      : (routingError ?? 'Chưa hợp lệ')
                                  }
                                  onClick={() =>
                                    canActSuggested
                                      ? onApproveScope!({
                                          productCode: row.product_code,
                                          skuCode: row.sku_code,
                                          routing: draft,
                                          plan: plan!.plan!,
                                        })
                                      : onApprove!({ planId: planId!, routing: draft })
                                  }
                                >
                                  {(canActSuggested ? scopeBusy : busyPlanId === planId) ? '…' : 'Duyệt'}
                                </button>
                                <button
                                  type="button"
                                  className="btn btn--ghost btn--xs"
                                  disabled={canActSuggested ? scopeBusy : busyPlanId === planId}
                                  onClick={() =>
                                    canActSuggested
                                      ? onRejectScope!({
                                          productCode: row.product_code,
                                          skuCode: row.sku_code,
                                          routing: draft,
                                          plan: plan!.plan!,
                                        })
                                      : onReject!(planId!)
                                  }
                                >
                                  Từ chối
                                </button>
                              </>
                            )}
                            {localAction && planId != null && (
                              <span className={`plan-row__done ${planBadgeClass(plan.status, localAction)}`}>
                                {planStatusLabel(plan.status, localAction)} #{planId}
                              </span>
                            )}
                          </>
                        ) : done ? (
                          <span className={`plan-row__done ${planBadgeClass('', done.action)}`}>
                            {planStatusLabel('', done.action)} #{done.planId}
                          </span>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                );

                const restoreRow = buildRestoreRow();
                return restoreRow
                  ? [skuRow, ...subRows, planRow, restoreRow]
                  : [skuRow, ...subRows, planRow];
                  })}
                </tbody>
              );
            })}
        </table>
      </div>
      {rows.length === 0 && <p className="muted">Chưa có dữ liệu routing.</p>}
    </div>
  );
}
