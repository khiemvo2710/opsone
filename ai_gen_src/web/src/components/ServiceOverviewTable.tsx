import { cloneElement, useEffect, useState, type ReactElement } from 'react';
import type { DashboardOverviewRow } from '../types/api';
import { HealthBadge } from './HealthBadge';
import { ProductThresholdEditor } from './ProductThresholdEditor';
import { ServiceMaintenanceButton } from './ServiceMaintenanceButton';
import { ProductMaintenanceActions } from './ProductMaintenanceActions';
import SkuMaintenanceTimeLabel from './SkuMaintenanceTimeLabel';
import { ScopeAutoEditor } from './ScopeAutoEditor';
import { ProviderMetricCell } from './ProviderMetricCell';
import { effectiveRowHealth } from '../utils/dashboardHealth';
import { isProviderUnderMaintenance, isSkuUnderActiveMaintenance } from '../utils/maintenanceDisplay';
import type { ProductThreshold } from '../types/api';
import {
  activeRoutingProviders,
  baselineRoutingPct,
  initialRoutingPct,
  isRoutingProviderSupported,
  routingPctFieldInvalid,
  routingPctMapsEqual,
  routingPctValidationError,
  ROUTING_PCT_MAX,
} from './RoutingPctEditor';
import { shouldShowManualApproval } from '../utils/scopeAuto';
import {
  maintenanceSuggestLabel,
  maintenanceWindowError,
  maintenanceWindowISO,
  defaultMaintenanceWindow,
  isSkuWideMaintenance,
} from '../utils/maintenanceWindow';
import { DateTimeLocalField } from './DateTimeLocalField';
import {
  ActiveMaintenanceCell,
  type ExtendMaintenancePayload,
  type ScopeMaintenanceActionPayload,
} from './ActiveMaintenanceCell';
import { groupRowsByProduct, type ProductGroup } from '../utils/dashboardRowOrder';
import { useMaintenanceDefaultDurationMin } from '../hooks/useMaintenanceDefaultDurationMin';

const PROVIDERS = ['ESALE', 'IMEDIA', 'SHOPPAY'] as const;

type PlanAction = 'approved' | 'rejected';

function fmtSku(sku: string): string {
  return sku === '' ? '—' : sku;
}

/** Viền dưới khối SKU (dòng chính + hàng con plan/bảo trì) trước SKU kế tiếp. */
function markScopeBlockEnd(rows: ReactElement[]): ReactElement[] {
  if (rows.length === 0) return rows;
  const last = rows[rows.length - 1];
  const existing = last.props.className ?? '';
  return [
    ...rows.slice(0, -1),
    cloneElement(last, {
      className: [existing, 'overview-table__scope-block-end'].filter(Boolean).join(' '),
    }),
  ];
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
  scopeDone: Record<string, { planId: number; action: PlanAction; rejectedRouting?: Record<string, number> }>,
  planActions: Record<number, PlanAction>,
): boolean {
  const done = scopeDone[scopeKey];
  if (done?.action === 'rejected') {
    if (!plan) return true;
    const proposed = plan.plan?.proposed_pct;
    if (!proposed || !done.rejectedRouting) return true;
    return routingPctMapsEqual(done.rejectedRouting, proposed, PROVIDERS);
  }
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

function shouldShowPendingPlan(
  row: DashboardOverviewRow,
  scopeKey: string,
  scopeDone: Record<string, { planId: number; action: PlanAction; rejectedRouting?: Record<string, number> }>,
  planActions: Record<number, PlanAction>,
): boolean {
  if (!shouldShowManualApproval(row)) return false;
  if (!row.pending_plan) return false;
  return !isRoutingRejected(scopeKey, row.pending_plan, scopeDone, planActions);
}

function countPlanSubRows(
  row: DashboardOverviewRow,
  scopeKey: string,
  scopeDone: Record<string, { planId: number; action: PlanAction }>,
  planActions: Record<number, PlanAction>,
): number {
  return shouldShowPendingPlan(row, scopeKey, scopeDone, planActions) ? 1 : 0;
}

/** Ẩn đề xuất bảo trì sau duyệt trong session hoặc khi cửa sổ bảo trì đã active. */
function shouldShowPendingMaintenance(
  row: DashboardOverviewRow,
  scopeKey: string,
  maintScopeDone: Record<string, { maintId: number; action: PlanAction }>,
  maintActions: Record<number, PlanAction>,
): boolean {
  if (!shouldShowManualApproval(row)) return false;
  if (!row.pending_maintenance) return false;
  if (row.maintenance) return false;
  if (maintScopeDone[scopeKey]?.action === 'approved') return false;
  return !isMaintenanceRejected(scopeKey, row.pending_maintenance, maintScopeDone, maintActions);
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
  n += countPlanSubRows(row, scopeKey, scopeDone, planActions);
  if (shouldShowPendingMaintenance(row, scopeKey, maintScopeDone, maintActions)) {
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

export interface ScopeRestoreProviderPayload {
  productCode: string;
  skuCode: string;
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

export interface ProductMaintenancePayload {
  productCode: string;
  skuCodes: string[];
  reason: string;
  startsAt: string;
  endsAt: string;
}

export interface ProductReopenMaintenancePayload {
  productCode: string;
  scopes: { skuCode: string; maintenanceIds?: string[] }[];
}

export interface ProductExtendMaintenancePayload {
  productCode: string;
  scopes: { skuCode: string; startsAt: string; endsAt: string }[];
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
  onRestoreProviderRouting?: (payload: ScopeRestoreProviderPayload) => void;
  onApplyScopeRouting?: (payload: ScopeRoutingApplyPayload) => void;
  onApproveMaintenance?: (payload: MaintenanceApprovePayload) => void;
  onRejectMaintenance?: (recommendationId: number) => void;
  onApproveScopeMaintenance?: (payload: ScopeMaintenancePayload) => void;
  onApproveProductMaintenance?: (payload: ProductMaintenancePayload) => void;
  onReopenProductMaintenance?: (payload: ProductReopenMaintenancePayload) => void;
  onExtendProductMaintenance?: (payload: ProductExtendMaintenancePayload) => void;
  onRejectScopeMaintenance?: (payload: ScopeMaintenancePayload) => void;
  onReopenMaintenance?: (payload: ScopeMaintenanceActionPayload) => void;
  onExtendMaintenance?: (payload: ExtendMaintenancePayload) => void;
  busyPlanId?: number | null;
  busyMaintId?: number | null;
  busyScopeKey?: string | null;
  busyProductKey?: string | null;
  planActions?: Record<number, PlanAction>;
  maintActions?: Record<number, PlanAction>;
  scopeDone?: Record<string, { planId: number; action: PlanAction; rejectedRouting?: Record<string, number> }>;
  maintScopeDone?: Record<string, { maintId: number; action: PlanAction }>;
  updatedAt?: string;
}

export function ServiceOverviewTable({
  rows,
  providers = [...PROVIDERS],
  thresholdsByProduct = {},
  onApprove,
  onReject,
  onApproveScope,
  onRejectScope,
  onRestoreProviderRouting,
  onApplyScopeRouting,
  onApproveMaintenance,
  onRejectMaintenance,
  onApproveScopeMaintenance,
  onApproveProductMaintenance,
  onReopenProductMaintenance,
  onExtendProductMaintenance,
  onRejectScopeMaintenance,
  onReopenMaintenance,
  onExtendMaintenance,
  busyPlanId,
  busyMaintId,
  busyScopeKey,
  busyProductKey,
  planActions = {},
  maintActions = {},
  scopeDone = {},
  maintScopeDone = {},
  updatedAt,
}: Props) {
  const maintenanceDurationMin = useMaintenanceDefaultDurationMin();
  const [draftRouting, setDraftRouting] = useState<Record<DraftKey, Record<string, number>>>({});
  const [dirtyPlanDraftKeys, setDirtyPlanDraftKeys] = useState<Set<string>>(new Set());
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
        next[scopeKey] = defaultMaintenanceWindow(maintenanceDurationMin);
        changed = true;
      }
      return changed ? next : prev;
    });
  }, [rows, updatedAt, maintenanceDurationMin]);

  useEffect(() => {
    setDirtyPlanDraftKeys((prev) => {
      const active = new Set<string>();
      for (const row of rows) {
        const plan = row.pending_plan;
        if (!plan?.plan?.proposed_pct) continue;
        active.add(String(planDraftKey(plan.id, `${row.product_code}:${row.sku_code}`)));
      }
      const next = new Set([...prev].filter((k) => active.has(k)));
      return next.size === prev.size ? prev : next;
    });
  }, [rows]);

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
        if (dirtyPlanDraftKeys.has(String(dk))) {
          continue;
        }
        const planProviders = activeRoutingProviders(providers, row.routing_pct, plan.plan);
        const fresh = initialRoutingPct(plan.plan.proposed_pct, row.routing_pct, planProviders);
        const existing = next[dk];
        if (!existing || !routingPctMapsEqual(existing, fresh, planProviders)) {
          next[dk] = fresh;
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [rows, providers, updatedAt, dirtyPlanDraftKeys]);

  const markPlanDraftDirty = (dk: DraftKey) => {
    setDirtyPlanDraftKeys((prev) => {
      const key = String(dk);
      if (prev.has(key)) return prev;
      const next = new Set(prev);
      next.add(key);
      return next;
    });
  };

  return (
    <div className="overview-table-wrap">
      <div className="table-scroll">
        <table className="data-table overview-table">
          <colgroup>
            <col className="overview-table__col-product" />
            <col className="overview-table__col-sku" />
            <col className="overview-table__col-tt" />
            {providers.map((p) => (
              <col key={p} className="overview-table__col-provider" />
            ))}
            <col className="overview-table__col-maint" />
            <col className="overview-table__col-auto" />
          </colgroup>
          {groupRowsByProduct(rows).map((group) => {
              const planLabelColSpan = 2;
              const bodyRowCount = groupBodyRowCount(
                group,
                scopeDone,
                maintScopeDone,
                planActions,
                maintActions,
                restoreScopeKey,
              );
              const productThreshold = thresholdsByProduct[group.productCode];
              const productMaintSkus = group.rows
                .filter((row) => !isSkuUnderActiveMaintenance(row))
                .map((row) => row.sku_code);
              const allSkusUnderMaintenance =
                group.rows.length > 0 &&
                group.rows.every((row) => isSkuUnderActiveMaintenance(row));
              const productActiveScopes = group.rows
                .filter((row) => row.maintenance)
                .map((row) => ({
                  skuCode: row.sku_code,
                  maintenanceIds: row.maintenance?.maintenance_ids,
                  startsAt: String(row.maintenance?.starts_at ?? ''),
                  endsAt: String(row.maintenance?.ends_at ?? ''),
                }));
              const showProductMaintenanceActions =
                allSkusUnderMaintenance &&
                productActiveScopes.length === group.rows.length &&
                (onReopenProductMaintenance != null || onExtendProductMaintenance != null);
              const productMaintBusy = busyProductKey === group.productCode;
              const showProductScopeAuto = group.rows.some((r) => r.sku_code !== '');
              const productAutoRow = group.rows[0];
              const productAutoInitial = {
                auto_action: productAutoRow?.product_auto_action ?? 'recommend_only',
                window_start: productAutoRow?.product_window_start,
                window_end: productAutoRow?.product_window_end,
              };
              return (
                <tbody key={group.productCode} className="overview-table__product-group">
                  <tr className="overview-table__threshold-row">
                    <ProductThresholdEditor
                      productCode={group.productCode}
                      productLabel={group.productLabel}
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
                  done?.action !== 'approved' &&
                  (isSuggested ||
                    plan.status === 'pending_approve' ||
                    plan.status === 'draft');
                const showPendingMaint = shouldShowPendingMaintenance(
                  row,
                  scopeKey,
                  maintScopeDone,
                  maintActions,
                );
                const isPendingMaint = Boolean(showPendingMaint && !maintDone);
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
                      initialRoutingPct(plan.plan?.proposed_pct, row.routing_pct, activeProviders)
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
                  restoreDraft[scopeKey] ??
                  initialRoutingPct(undefined, row.routing_pct, activeProviders);
                const restoreError = isRestoring
                  ? routingPctValidationError(restoreValues, activeProviders)
                  : null;
                const restoreValid = restoreError === null;
                const baselineDraft = baselineRoutingPct(row.baseline_pct, activeProviders);

                const saveRoutingDraft = (values: Record<string, number>) => {
                  if (routingPctValidationError(values, activeProviders)) return;
                  if (
                    baselineDraft &&
                    routingPctMapsEqual(values, baselineDraft, activeProviders) &&
                    onRestoreProviderRouting
                  ) {
                    onRestoreProviderRouting({
                      productCode: row.product_code,
                      skuCode: row.sku_code,
                    });
                    return;
                  }
                  onApplyScopeRouting?.({
                    productCode: row.product_code,
                    skuCode: row.sku_code,
                    routing: values,
                  });
                };

                const buildRestoreRow = () => {
                  if (!isRestoring || !onRestoreProviderRouting) return null;
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
                        const invalid =
                          supported && routingPctFieldInvalid(p, restoreValues, providers);
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
                      <td colSpan={2} className="overview-table__plan-actions">
                        <div className="overview-table__plan-actions-inner">
                          {restoreError && (
                            <span className="overview-table__plan-error">{restoreError}</span>
                          )}
                          <button
                            type="button"
                            className="btn btn--ghost btn--xs"
                            disabled={scopeBusy || !baselineDraft}
                            title={
                              baselineDraft
                                ? 'Điền tỷ lệ routing theo baseline biz'
                                : 'Chưa có dữ liệu baseline'
                            }
                            onClick={() => {
                              if (!baselineDraft) return;
                              setRestoreDraft((prev) => ({
                                ...prev,
                                [scopeKey]: baselineDraft,
                              }));
                            }}
                          >
                            Trả lại
                          </button>
                          <button
                            type="button"
                            className="btn btn--primary btn--xs"
                            disabled={scopeBusy || !restoreValid}
                            title={
                              restoreValid ? 'Áp dụng routing đã nhập' : (restoreError ?? 'Chưa hợp lệ')
                            }
                            onClick={() => {
                              if (!restoreValid) return;
                              saveRoutingDraft(restoreValues);
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
                const inMaintenance = isSkuUnderActiveMaintenance(row);

                const skuRow = (
                  <tr
                    key={`${row.product_code}-${row.sku_code}`}
                    id={`overview-scope-${scopeKey}`}
                    className={[
                      'overview-table__sku-row',
                      !inMaintenance && skuHealth === 'red' ? 'overview-table__sku-row--alert' : '',
                    ]
                      .filter(Boolean)
                      .join(' ') || undefined}
                  >
                    {isFirstInGroup && (
                      <td
                        rowSpan={bodyRowCount}
                        className="overview-table__product-cell overview-table__product-cell--group"
                      >
                        <div className="overview-table__product-cell-layout">
                          <div className="overview-table__product-cell-main">
                            {onApproveProductMaintenance && onApproveScopeMaintenance && productMaintSkus.length > 0 && (
                              <ServiceMaintenanceButton
                                productCode={group.productCode}
                                skuCodes={productMaintSkus}
                                busy={productMaintBusy}
                                onStartProduct={onApproveProductMaintenance}
                              />
                            )}
                            {showProductMaintenanceActions && (
                                <ProductMaintenanceActions
                                  productCode={group.productCode}
                                  scopes={productActiveScopes}
                                  busy={productMaintBusy}
                                  onReopenProduct={onReopenProductMaintenance}
                                  onExtendProduct={onExtendProductMaintenance}
                                />
                              )}
                          </div>
                          {showProductScopeAuto && (
                            <div className="overview-table__product-cell-auto">
                              <ScopeAutoEditor
                                productCode={group.productCode}
                                skuCode=""
                                level="product"
                                initial={productAutoInitial}
                              />
                            </div>
                          )}
                        </div>
                      </td>
                    )}
                    {row.maintenance ? (
                      <td
                        colSpan={2}
                        className="overview-table__sku-cell overview-table__sku-cell--with-maint"
                      >
                        <div className="overview-table__sku-cell-inner">
                          <span className="overview-table__sku-name">{fmtSku(row.sku_code)}</span>
                          <SkuMaintenanceTimeLabel
                            startsAt={String(row.maintenance.starts_at ?? '')}
                            endsAt={String(row.maintenance.ends_at ?? '')}
                            title={row.maintenance.reason ?? row.maintenance.label_vi}
                          />
                        </div>
                      </td>
                    ) : (
                      <>
                        <td className="overview-table__sku-cell">
                          <div className="overview-table__sku-cell-inner">
                            <span className="overview-table__sku-name" style={{ flexGrow: 1 }}>{fmtSku(row.sku_code)}</span>
                          </div>
                        </td>
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
                      </>
                    )}
                    {providers.map((p) => {
                      const supported = isProviderSupported(p);
                      const routingPct = row.routing_pct?.[p];
                      const maintained = isProviderUnderMaintenance(row, p);
                      const inactive = supported && !maintained && (routingPct ?? 0) <= 0;
                      const reopenBlocked =
                        isPending ||
                        isPendingMaint ||
                        Boolean(row.maintenance) ||
                        !onRestoreProviderRouting ||
                        (restoreScopeKey != null && restoreScopeKey !== scopeKey);
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
                          reopenDisabled={reopenBlocked}
                          reopenBusy={scopeBusy}
                          scopeRestoring={isRestoring}
                          onReopen={
                            inactive && !reopenBlocked && !isRestoring
                              ? () => {
                                  setRestoreScopeKey(scopeKey);
                                  setRestoreDraft((prev) => ({
                                    ...prev,
                                    [scopeKey]: initialRoutingPct(undefined, row.routing_pct, activeProviders),
                                  }));
                                }
                              : undefined
                          }
                        />
                      );
                    })}
                    <td colSpan={2} className="overview-table__scope-controls-cell">
                      <div className="overview-table__product-cell-layout">
                        <div className="overview-table__product-cell-main">
                          {inMaintenance ? (
                            <ActiveMaintenanceCell
                              row={row}
                              busy={scopeBusy}
                              onReopen={onReopenMaintenance}
                              onExtend={onExtendMaintenance}
                            />
                          ) : (
                            onApproveScopeMaintenance && (
                              <ServiceMaintenanceButton
                                productCode={row.product_code}
                                skuCode={row.sku_code}
                                busy={scopeBusy}
                                onStart={onApproveScopeMaintenance}
                              />
                            )
                          )}
                        </div>
                        <div className="overview-table__product-cell-auto">
                          <ScopeAutoEditor
                            productCode={row.product_code}
                            skuCode={row.sku_code}
                            initial={{
                              auto_action: row.scope_auto_action ?? row.auto_action ?? 'recommend_only',
                              window_start: row.scope_window_start ?? row.window_start,
                              window_end: row.scope_window_end ?? row.window_end,
                            }}
                          />
                        </div>
                      </div>
                    </td>
                  </tr>
                );

                if (!showPlanRow) {
                  const restoreRow = buildRestoreRow();
                  return markScopeBlockEnd(
                    restoreRow ? [skuRow, restoreRow] : [skuRow],
                  );
                }

                const subRows: ReactElement[] = [];

                if (showPendingMaint && row.pending_maintenance) {
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
                  const maintWindow =
                    maintWindowDraft[scopeKey] ?? defaultMaintenanceWindow(maintenanceDurationMin);
                  const maintWindowErr = maintenanceWindowError(maintWindow.startsAt, maintWindow.endsAt);
                  const maintISO = maintWindowErr
                    ? null
                    : maintenanceWindowISO(maintWindow.startsAt, maintWindow.endsAt);
                  const skuWideMaint =
                    Boolean(pm.scope_level) || isSkuWideMaintenance(pm.reason);
                  subRows.push(
                    <tr
                      key={`${row.product_code}-${row.sku_code}-maint-suggest`}
                      className="overview-table__plan-row overview-table__maint-suggest-row"
                      title={pm.reason}
                    >
                      <td colSpan={planLabelColSpan} className="overview-table__plan-label">
                        Đề xuất bảo trì
                      </td>
                      <td colSpan={2} className="overview-table__maint-suggest-message">
                        <span className="maint-suggest-bar__message" title={pm.reason}>
                          {maintenanceSuggestLabel(pm)}
                        </span>
                      </td>
                      <td
                        colSpan={providers.length}
                        className="overview-table__plan-actions overview-table__maint-suggest-actions"
                      >
                        <div className="maint-suggest-bar maint-suggest-bar--even">
                          {canActMaint && (
                            <div className="maint-suggest-bar__window">
                              <DateTimeLocalField
                                label="Từ"
                                compact
                                className="datetime-local-field maint-suggest-bar__field"
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
                                label="Đến"
                                compact
                                className="datetime-local-field maint-suggest-bar__field"
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
                                disabled={maintBusy || Boolean(maintWindowErr) || !maintISO}
                                title="Lên lịch bảo trì theo khung giờ đã chọn"
                                onClick={() => {
                                  if (!maintISO) return;
                                  if (canActMaintReal && maintId != null) {
                                    onApproveMaintenance!({
                                      recommendationId: maintId,
                                      startsAt: maintISO.starts_at,
                                      endsAt: maintISO.ends_at,
                                    });
                                    return;
                                  }
                                  onApproveScopeMaintenance!({
                                    productCode: row.product_code,
                                    skuCode: row.sku_code,
                                    reason: pm.reason,
                                    providerCode: skuWideMaint ? undefined : pm.provider_code,
                                    startsAt: maintISO.starts_at,
                                    endsAt: maintISO.ends_at,
                                  });
                                }}
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

                if (!shouldShowPendingPlan(row, scopeKey, scopeDone, planActions)) {
                  const restoreRow = buildRestoreRow();
                  const out: ReactElement[] = [skuRow, ...subRows];
                  if (restoreRow) out.push(restoreRow);
                  return markScopeBlockEnd(out);
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
                                    markPlanDraftDirty(draftKey);
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
                                {planValues[p] != null
                                  ? `${Math.round(planValues[p] * 10) / 10}%`
                                  : '—'}
                              </span>
                            )
                          ) : (
                            <span className="muted">—</span>
                          )}
                        </td>
                      );
                    })}
                    <td colSpan={2} className="overview-table__plan-actions">
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
                                      ? 'Duyệt và áp dụng routing đã nhập'
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
                              <span
                                className={`plan-row__done ${planBadgeClass(plan.status, localAction)}`}
                              >
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
                return markScopeBlockEnd(
                  restoreRow
                    ? [skuRow, ...subRows, planRow, restoreRow]
                    : [skuRow, ...subRows, planRow],
                );
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
