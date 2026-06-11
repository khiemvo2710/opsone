import { useMemo, useState } from 'react';
import { useToast } from '../context/ToastContext';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import { routingPctMapsEqual } from '../components/RoutingPctEditor';
import { scopeDisplayLabel } from '../utils/dashboardRowOrder';
import type {
  DashboardOverview,
  DashboardOverviewRow,
  HealthStatus,
  ProductThreshold,
  ServiceType,
} from '../types/api';
import { useProductThresholds } from '../hooks/useProductThresholds';
import { HealthBadge } from '../components/HealthBadge';
import {
  ServiceOverviewTable,
  type ApproveRoutingPayload,
  type MaintenanceApprovePayload,
  type ScopeMaintenancePayload,
  type ProductMaintenancePayload,
  type ProductReopenMaintenancePayload,
  type ProductExtendMaintenancePayload,
  type ScopeRoutingPayload,
  type ScopeRoutingApplyPayload,
} from '../components/ServiceOverviewTable';
import type {
  ExtendMaintenancePayload,
  ScopeMaintenanceActionPayload,
} from '../components/ActiveMaintenanceCell';
import { useOverallHealth } from '../hooks/useOverallHealth';
import { RedSkuScrollNav } from '../components/RedSkuScrollNav';
import {
  effectiveRowHealth,
  rowPendingApprove,
  worstHealth,
} from '../utils/dashboardHealth';

type PlanAction = 'approved' | 'rejected';

const SERVICE_TABS: { id: ServiceType; label: string }[] = [
  { id: 'card', label: 'Thẻ' },
  { id: 'topup', label: 'Topup' },
  { id: 'topup_data', label: 'Data' },
];

function rowServiceType(row: DashboardOverviewRow): ServiceType {
  if (row.service_type) return row.service_type;
  if (row.product_code.startsWith('DATA_')) return 'topup_data';
  if (row.product_code.startsWith('TOPUP_')) return 'topup';
  return 'card';
}

interface ServiceTabMeta {
  status: HealthStatus;
  pendingApprove: boolean;
}

function tabMetaMap(
  grouped: Record<string, DashboardOverviewRow[]>,
  thresholdsByProduct: Record<string, ProductThreshold>,
): Record<ServiceType, ServiceTabMeta> {
  const out: Record<ServiceType, ServiceTabMeta> = {
    card: { status: 'green', pendingApprove: false },
    topup: { status: 'green', pendingApprove: false },
    topup_data: { status: 'green', pendingApprove: false },
  };
  for (const tab of SERVICE_TABS) {
    let pendingApprove = false;
    const rowStatuses: HealthStatus[] = [];
    for (const row of grouped[tab.id] ?? []) {
      if (rowPendingApprove(row)) {
        pendingApprove = true;
      }
      rowStatuses.push(effectiveRowHealth(row, thresholdsByProduct[row.product_code]));
    }
    out[tab.id] = {
      status: worstHealth(rowStatuses.length ? rowStatuses : ['green']),
      pendingApprove,
    };
  }
  return out;
}

const ROUTING_PROVIDERS = ['ESALE', 'IMEDIA', 'SHOPPAY'] as const;

function removePendingPlan(data: DashboardOverview | undefined, planId: number): DashboardOverview | undefined {
  if (!data?.rows) return data;
  return {
    ...data,
    rows: data.rows.map((row) =>
      row.pending_plan?.id === planId ? { ...row, pending_plan: undefined } : row,
    ),
  };
}

function removePendingMaintenance(
  data: DashboardOverview | undefined,
  maintId: number,
): DashboardOverview | undefined {
  if (!data?.rows) return data;
  return {
    ...data,
    rows: data.rows.map((row) =>
      row.pending_maintenance?.id === maintId ? { ...row, pending_maintenance: undefined } : row,
    ),
  };
}

function removeScopePending(
  data: DashboardOverview | undefined,
  productCode: string,
  skuCode: string,
  kind: 'routing' | 'maintenance',
): DashboardOverview | undefined {
  if (!data?.rows) return data;
  return {
    ...data,
    rows: data.rows.map((row) => {
      if (row.product_code !== productCode || row.sku_code !== skuCode) return row;
      if (kind === 'routing') {
        return { ...row, pending_plan: undefined };
      }
      return { ...row, pending_maintenance: undefined };
    }),
  };
}

export function Dashboard() {
  const qc = useQueryClient();
  const { showToast } = useToast();
  const [busyPlanId, setBusyPlanId] = useState<number | null>(null);
  const [busyMaintId, setBusyMaintId] = useState<number | null>(null);
  const [busyScopeKey, setBusyScopeKey] = useState<string | null>(null);
  const [busyProductKey, setBusyProductKey] = useState<string | null>(null);
  const [planActions, setPlanActions] = useState<Record<number, PlanAction>>({});
  const [maintActions, setMaintActions] = useState<Record<number, PlanAction>>({});
  const [scopeDone, setScopeDone] = useState<
    Record<string, { planId: number; action: PlanAction; rejectedRouting?: Record<string, number> }>
  >({});
  const [maintScopeDone, setMaintScopeDone] = useState<
    Record<string, { maintId: number; action: PlanAction }>
  >({});
  const [serviceTab, setServiceTab] = useState<ServiceType>('card');
  const overallHealth = useOverallHealth();

  const { data: overview, isLoading: overviewLoading, isFetching: overviewFetching } = useQuery({
    queryKey: ['dashboard-overview'],
    queryFn: () => api<DashboardOverview>('/dashboard/overview'),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const rowsByService = useMemo(() => {
    const all = overview?.rows ?? [];
    const grouped: Record<string, DashboardOverviewRow[]> = {
      card: [],
      topup: [],
      topup_data: [],
    };
    for (const row of all) {
      const svc = rowServiceType(row);
      if (!grouped[svc]) grouped[svc] = [];
      grouped[svc].push(row);
    }
    return grouped;
  }, [overview?.rows]);

  const productCodes = useMemo(
    () => (overview?.rows ?? []).map((r) => r.product_code),
    [overview?.rows],
  );
  const thresholdsByProduct = useProductThresholds(productCodes, overview?.thresholds);

  const filteredRows = rowsByService[serviceTab] ?? [];
  const overviewRows = () =>
    qc.getQueryData<DashboardOverview>(['dashboard-overview'])?.rows ?? overview?.rows;
  const scopeDisplayName = (productCode: string, skuCode: string) =>
    scopeDisplayLabel(overviewRows(), productCode, skuCode);
  const scopeNameByMaintId = (maintId: number) => {
    const row = overviewRows()?.find((r) => r.pending_maintenance?.id === maintId);
    return row
      ? scopeDisplayLabel(overviewRows(), row.product_code, row.sku_code)
      : `#${maintId}`;
  };
  const scopeNameByPlanId = (planId: number) => {
    const row = overviewRows()?.find((r) => r.pending_plan?.id === planId);
    return row
      ? scopeDisplayLabel(overviewRows(), row.product_code, row.sku_code)
      : `#${planId}`;
  };
  const serviceTabMeta = useMemo(
    () => tabMetaMap(rowsByService, thresholdsByProduct),
    [rowsByService, thresholdsByProduct],
  );

  const refreshOverview = async (planId: number, action: PlanAction) => {
    const before = qc.getQueryData<DashboardOverview>(['dashboard-overview']);
    const row = before?.rows?.find((r) => r.pending_plan?.id === planId);
    if (row) {
      const scopeKey = `${row.product_code}:${row.sku_code}`;
      setScopeDone((prev) => ({ ...prev, [scopeKey]: { planId, action } }));
    }

    setPlanActions((prev) => ({ ...prev, [planId]: action }));
    qc.setQueryData<DashboardOverview>(['dashboard-overview'], (old) => removePendingPlan(old, planId));
    await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
    void qc.invalidateQueries({ queryKey: ['health-status'] });
    void qc.invalidateQueries({ queryKey: ['routing-plans'] });
    void qc.invalidateQueries({ queryKey: ['incidents'] });

    const fresh = qc.getQueryData<DashboardOverview>(['dashboard-overview']);
    if (row) {
      const scopeKey = `${row.product_code}:${row.sku_code}`;
      const newPlan = fresh?.rows?.find((r) => `${r.product_code}:${r.sku_code}` === scopeKey)?.pending_plan;
      if (!newPlan || action === 'rejected') {
        setScopeDone((prev) => {
          const next = { ...prev };
          delete next[scopeKey];
          return next;
        });
        if (action === 'rejected') {
          setPlanActions((prev) => {
            const next = { ...prev };
            delete next[planId];
            return next;
          });
        }
      } else if (newPlan.id !== planId) {
        setScopeDone((prev) => {
          const next = { ...prev };
          delete next[scopeKey];
          return next;
        });
        setPlanActions((prev) => {
          const next = { ...prev };
          delete next[planId];
          return next;
        });
      }
    }
  };

  const approve = useMutation({
    mutationFn: ({ planId, routing }: ApproveRoutingPayload) =>
      api(`/routing-plans/${planId}/approve`, {
        method: 'POST',
        body: JSON.stringify({ proposed_pct: routing }),
      }),
    onMutate: ({ planId }) => {
      setBusyPlanId(planId);
    },
    onSuccess: async (_, { planId }) => {
      showToast('ok', `Đã duyệt kế hoạch routing ${scopeNameByPlanId(planId)} — routing đã áp dụng.`);
      await refreshOverview(planId, 'approved');
      setBusyPlanId(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Duyệt thất bại');
      setBusyPlanId(null);
    },
  });

  const refreshMaintenanceOverview = async (maintId: number, action: PlanAction) => {
    const before = qc.getQueryData<DashboardOverview>(['dashboard-overview']);
    const row = before?.rows?.find((r) => r.pending_maintenance?.id === maintId);
    if (row) {
      const scopeKey = `${row.product_code}:${row.sku_code}`;
      setMaintScopeDone((prev) => ({ ...prev, [scopeKey]: { maintId, action } }));
    }
    setMaintActions((prev) => ({ ...prev, [maintId]: action }));
    qc.setQueryData<DashboardOverview>(['dashboard-overview'], (old) =>
      removePendingMaintenance(old, maintId),
    );
    await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
    void qc.invalidateQueries({ queryKey: ['health-status'] });
    void qc.invalidateQueries({ queryKey: ['maintenance'] });
    void qc.invalidateQueries({ queryKey: ['incidents'] });

    const fresh = qc.getQueryData<DashboardOverview>(['dashboard-overview']);
    const scopeKey = row ? `${row.product_code}:${row.sku_code}` : '';
    if (scopeKey) {
      const freshRow = fresh?.rows?.find((r) => `${r.product_code}:${r.sku_code}` === scopeKey);
      if (action === 'approved') {
        const resolved =
          Boolean(freshRow?.maintenance) || !freshRow?.pending_maintenance;
        if (resolved) {
          setMaintScopeDone((prev) => {
            const next = { ...prev };
            delete next[scopeKey];
            return next;
          });
        }
      }
    }
  };

  const approveMaintenance = useMutation({
    mutationFn: (payload: MaintenanceApprovePayload) =>
      api(`/recommendations/${payload.recommendationId}/approve`, {
        method: 'POST',
        body: JSON.stringify({
          starts_at: payload.startsAt,
          ends_at: payload.endsAt,
        }),
      }),
    onMutate: (payload) => {
      setBusyMaintId(payload.recommendationId);
    },
    onSuccess: async (_, payload) => {
      showToast('ok', `Đã duyệt bảo trì ${scopeNameByMaintId(payload.recommendationId)} — cửa sổ bảo trì đã kích hoạt.`);
      await refreshMaintenanceOverview(payload.recommendationId, 'approved');
      setBusyMaintId(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Duyệt bảo trì thất bại');
      setBusyMaintId(null);
    },
  });

  const rejectMaintenance = useMutation({
    mutationFn: (id: number) => api(`/recommendations/${id}/reject`, { method: 'POST' }),
    onMutate: (id) => {
      setBusyMaintId(id);
    },
    onSuccess: async (_, id) => {
      showToast('ok', `Đã từ chối đề xuất bảo trì ${scopeNameByMaintId(id)}.`);
      await refreshMaintenanceOverview(id, 'rejected');
      setBusyMaintId(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Từ chối bảo trì thất bại');
      setBusyMaintId(null);
    },
  });

  const refreshScopeRouting = async (
    payload: ScopeRoutingPayload,
    action: PlanAction,
    planId?: number,
  ) => {
    const scopeKey = `${payload.productCode}:${payload.skuCode}`;
    const rejectedRouting = payload.routing ?? payload.plan?.proposed_pct ?? {};
    setScopeDone((prev) => ({
      ...prev,
      [scopeKey]: {
        planId: planId ?? 0,
        action,
        ...(action === 'rejected' ? { rejectedRouting } : {}),
      },
    }));
    qc.setQueryData<DashboardOverview>(['dashboard-overview'], (old) =>
      removeScopePending(old, payload.productCode, payload.skuCode, 'routing'),
    );
    await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
    void qc.invalidateQueries({ queryKey: ['health-status'] });
    void qc.invalidateQueries({ queryKey: ['routing-plans'] });
    void qc.invalidateQueries({ queryKey: ['incidents'] });

    const fresh = qc.getQueryData<DashboardOverview>(['dashboard-overview']);
    const freshRow = fresh?.rows?.find(
      (r) => r.product_code === payload.productCode && r.sku_code === payload.skuCode,
    );
    if (action === 'rejected' && !freshRow?.pending_plan) {
      setScopeDone((prev) => {
        const next = { ...prev };
        delete next[scopeKey];
        return next;
      });
    } else if (action === 'rejected' && freshRow?.pending_plan) {
      const proposed = freshRow.pending_plan.plan?.proposed_pct;
      if (
        !proposed ||
        !routingPctMapsEqual(proposed, rejectedRouting, ROUTING_PROVIDERS)
      ) {
        setScopeDone((prev) => {
          const next = { ...prev };
          delete next[scopeKey];
          return next;
        });
      }
    } else if (action === 'approved') {
      setScopeDone((prev) => {
        const next = { ...prev };
        delete next[scopeKey];
        return next;
      });
    }
  };

  const refreshScopeMaintenance = async (payload: ScopeMaintenancePayload, action: PlanAction) => {
    const scopeKey = `${payload.productCode}:${payload.skuCode}`;
    setMaintScopeDone((prev) => ({
      ...prev,
      [scopeKey]: { maintId: 0, action },
    }));
    qc.setQueryData<DashboardOverview>(['dashboard-overview'], (old) =>
      removeScopePending(old, payload.productCode, payload.skuCode, 'maintenance'),
    );
    await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
    void qc.invalidateQueries({ queryKey: ['health-status'] });
    void qc.invalidateQueries({ queryKey: ['maintenance'] });
    void qc.invalidateQueries({ queryKey: ['incidents'] });

    const fresh = qc.getQueryData<DashboardOverview>(['dashboard-overview']);
    const freshRow = fresh?.rows?.find(
      (r) => r.product_code === payload.productCode && r.sku_code === payload.skuCode,
    );
    if (action === 'approved') {
      const resolved = Boolean(freshRow?.maintenance) || !freshRow?.pending_maintenance;
      if (resolved) {
        setMaintScopeDone((prev) => {
          const next = { ...prev };
          delete next[scopeKey];
          return next;
        });
      }
    }
  };

  const approveScope = useMutation({
    mutationFn: (payload: ScopeRoutingPayload) =>
      api(`/scopes/${payload.productCode}/${payload.skuCode}/routing/approve`, {
        method: 'POST',
        body: JSON.stringify({
          proposed_pct: payload.routing,
          plan: payload.plan,
        }),
      }),
    onMutate: (payload) => {
      setBusyScopeKey(`${payload.productCode}:${payload.skuCode}`);
    },
    onSuccess: async (res, payload) => {
      const planId = (res as { plan_id?: number })?.plan_id;
      showToast('ok', `Đã duyệt đề xuất routing ${scopeDisplayName(payload.productCode, payload.skuCode)} — routing đã áp dụng.`);
      await refreshScopeRouting(payload, 'approved', planId);
      setBusyScopeKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Duyệt đề xuất routing thất bại');
      setBusyScopeKey(null);
    },
  });

  const rejectScope = useMutation({
    mutationFn: (payload: ScopeRoutingPayload) =>
      api(`/scopes/${payload.productCode}/${payload.skuCode}/routing/reject`, {
        method: 'POST',
        body: JSON.stringify({
          plan: {
            ...payload.plan,
            proposed_pct: payload.routing,
          },
        }),
      }),
    onMutate: (payload) => {
      setBusyScopeKey(`${payload.productCode}:${payload.skuCode}`);
    },
    onSuccess: async (_, payload) => {
      showToast('ok', `Đã từ chối đề xuất routing ${scopeDisplayName(payload.productCode, payload.skuCode)}.`);
      await refreshScopeRouting(payload, 'rejected');
      setBusyScopeKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Từ chối đề xuất routing thất bại');
      setBusyScopeKey(null);
    },
  });

  const applyScopeRouting = useMutation({
    mutationFn: (payload: ScopeRoutingApplyPayload) =>
      api(`/scopes/${payload.productCode}/${payload.skuCode}/routing/apply`, {
        method: 'POST',
        body: JSON.stringify({ proposed_pct: payload.routing }),
      }),
    onMutate: (payload) => {
      setBusyScopeKey(`${payload.productCode}:${payload.skuCode}`);
    },
    onSuccess: async (_, payload) => {
      showToast('ok', `Đã cập nhật routing ${scopeDisplayName(payload.productCode, payload.skuCode)}.`);
      await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      void qc.invalidateQueries({ queryKey: ['health-status'] });
      setBusyScopeKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Cập nhật routing thất bại');
      setBusyScopeKey(null);
    },
  });

  const restoreScopeBaseline = useMutation({
    mutationFn: (payload: { productCode: string; skuCode: string }) =>
      api<{ applied: boolean }>(
        `/scopes/${payload.productCode}/${payload.skuCode}/routing/restore-baseline`,
        { method: 'POST', body: JSON.stringify({}) },
      ),
    onMutate: (payload) => {
      setBusyScopeKey(`${payload.productCode}:${payload.skuCode}`);
    },
    onSuccess: async (_, payload) => {
      showToast(
        'ok',
        `Đã mở lại provider ${scopeDisplayName(payload.productCode, payload.skuCode)} — routing về baseline; auto chờ chu kỳ Agent tiếp theo.`,
      );
      await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      void qc.invalidateQueries({ queryKey: ['health-status'] });
      setBusyScopeKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Mở lại provider thất bại');
      setBusyScopeKey(null);
    },
  });

  const approveScopeMaintenance = useMutation({
    mutationFn: (payload: ScopeMaintenancePayload) =>
      api(`/scopes/${payload.productCode}/${payload.skuCode}/maintenance/approve`, {
        method: 'POST',
        body: JSON.stringify({
          reason: payload.reason,
          provider_code: payload.providerCode,
          starts_at: payload.startsAt,
          ends_at: payload.endsAt,
        }),
      }),
    onMutate: (payload) => {
      setBusyScopeKey(`${payload.productCode}:${payload.skuCode}`);
    },
    onSuccess: async (_, payload) => {
      showToast('ok', `Đã duyệt bảo trì ${scopeDisplayName(payload.productCode, payload.skuCode)} — cửa sổ bảo trì đã kích hoạt.`);
      await refreshScopeMaintenance(payload, 'approved');
      setBusyScopeKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Duyệt bảo trì thất bại');
      setBusyScopeKey(null);
    },
  });

  const approveProductMaintenance = useMutation({
    mutationFn: async (payload: ProductMaintenancePayload) => {
      const failed: string[] = [];
      for (const skuCode of payload.skuCodes) {
        try {
          await api(`/scopes/${payload.productCode}/${skuCode}/maintenance/approve`, {
            method: 'POST',
            body: JSON.stringify({
              reason: payload.reason,
              starts_at: payload.startsAt,
              ends_at: payload.endsAt,
            }),
          });
        } catch {
          failed.push(skuCode || '—');
        }
      }
      if (failed.length > 0) {
        throw new ApiClientError({
          code: 'partial_failure',
          message_vi: `Bảo trì thất bại ${failed.length}/${payload.skuCodes.length} SKU: ${failed.join(', ')}`,
        });
      }
    },
    onMutate: (payload) => {
      setBusyProductKey(payload.productCode);
    },
    onSuccess: async (_, payload) => {
      showToast(
        'ok',
        `Đã bảo trì toàn bộ ${payload.skuCodes.length} SKU của ${payload.productCode}.`,
      );
      await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      void qc.invalidateQueries({ queryKey: ['health-status'] });
      void qc.invalidateQueries({ queryKey: ['maintenance'] });
      setBusyProductKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Bảo trì toàn sản phẩm thất bại');
      void qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      setBusyProductKey(null);
    },
  });

  const reopenProductMaintenance = useMutation({
    mutationFn: async (payload: ProductReopenMaintenancePayload) => {
      const failed: string[] = [];
      for (const scope of payload.scopes) {
        try {
          await api<{ applied: boolean }>(
            `/scopes/${payload.productCode}/${scope.skuCode}/maintenance/reopen-service`,
            {
              method: 'POST',
              body: JSON.stringify({ maintenance_ids: scope.maintenanceIds ?? [] }),
            },
          );
        } catch {
          failed.push(scope.skuCode || '—');
        }
      }
      if (failed.length > 0) {
        throw new ApiClientError({
          code: 'partial_failure',
          message_vi: `Mở lại thất bại ${failed.length}/${payload.scopes.length} SKU: ${failed.join(', ')}`,
        });
      }
    },
    onMutate: (payload) => {
      setBusyProductKey(payload.productCode);
    },
    onSuccess: async (_, payload) => {
      showToast(
        'ok',
        `Đã mở lại dịch vụ toàn bộ ${payload.scopes.length} SKU của ${payload.productCode} — routing về baseline.`,
      );
      await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      void qc.invalidateQueries({ queryKey: ['health-status'] });
      void qc.invalidateQueries({ queryKey: ['maintenance'] });
      setBusyProductKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Mở lại toàn sản phẩm thất bại');
      void qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      setBusyProductKey(null);
    },
  });

  const extendProductMaintenance = useMutation({
    mutationFn: async (payload: ProductExtendMaintenancePayload) => {
      const failed: string[] = [];
      for (const scope of payload.scopes) {
        try {
          await api(`/scopes/${payload.productCode}/${scope.skuCode}/maintenance/extend`, {
            method: 'POST',
            body: JSON.stringify({
              starts_at: scope.startsAt,
              ends_at: scope.endsAt,
            }),
          });
        } catch {
          failed.push(scope.skuCode || '—');
        }
      }
      if (failed.length > 0) {
        throw new ApiClientError({
          code: 'partial_failure',
          message_vi: `Gia hạn thất bại ${failed.length}/${payload.scopes.length} SKU: ${failed.join(', ')}`,
        });
      }
    },
    onMutate: (payload) => {
      setBusyProductKey(payload.productCode);
    },
    onSuccess: async (_, payload) => {
      showToast(
        'ok',
        `Đã gia hạn bảo trì ${payload.scopes.length} SKU của ${payload.productCode}.`,
      );
      await qc.invalidateQueries({ queryKey: ['dashboard-overview'] });
      void qc.invalidateQueries({ queryKey: ['maintenance'] });
      setBusyProductKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Gia hạn toàn sản phẩm thất bại');
      void qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      setBusyProductKey(null);
    },
  });

  const reopenMaintenance = useMutation({
    mutationFn: (payload: ScopeMaintenanceActionPayload) =>
      api<{ applied: boolean; proposed_pct?: Record<string, number> }>(
        `/scopes/${payload.productCode}/${payload.skuCode}/maintenance/reopen-service`,
        {
          method: 'POST',
          body: JSON.stringify({
            maintenance_ids: payload.maintenanceIds ?? [],
          }),
        },
      ),
    onMutate: (payload) => {
      setBusyScopeKey(`${payload.productCode}:${payload.skuCode}`);
    },
    onSuccess: async (_, payload) => {
      showToast(
        'ok',
        `Đã mở lại dịch vụ ${scopeDisplayName(payload.productCode, payload.skuCode)} — routing về baseline; auto chờ chu kỳ Agent tiếp theo.`,
      );
      await qc.refetchQueries({ queryKey: ['dashboard-overview'] });
      void qc.invalidateQueries({ queryKey: ['health-status'] });
      void qc.invalidateQueries({ queryKey: ['maintenance'] });
      setBusyScopeKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Mở lại dịch vụ thất bại');
      setBusyScopeKey(null);
    },
  });

  const extendMaintenance = useMutation({
    mutationFn: (payload: ExtendMaintenancePayload) =>
      api(`/scopes/${payload.productCode}/${payload.skuCode}/maintenance/extend`, {
        method: 'POST',
        body: JSON.stringify({
          starts_at: payload.startsAt,
          ends_at: payload.endsAt,
        }),
      }),
    onMutate: (payload) => {
      setBusyScopeKey(`${payload.productCode}:${payload.skuCode}`);
    },
    onSuccess: async (_, payload) => {
      showToast('ok', `Đã cập nhật thời gian bảo trì ${scopeDisplayName(payload.productCode, payload.skuCode)}.`);
      await qc.invalidateQueries({ queryKey: ['dashboard-overview'] });
      void qc.invalidateQueries({ queryKey: ['maintenance'] });
      setBusyScopeKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Gia hạn bảo trì thất bại');
      setBusyScopeKey(null);
    },
  });

  const rejectScopeMaintenance = useMutation({
    mutationFn: (payload: ScopeMaintenancePayload) =>
      api(`/scopes/${payload.productCode}/${payload.skuCode}/maintenance/reject`, {
        method: 'POST',
        body: JSON.stringify({ reason: payload.reason }),
      }),
    onMutate: (payload) => {
      setBusyScopeKey(`${payload.productCode}:${payload.skuCode}`);
    },
    onSuccess: async (_, payload) => {
      showToast('ok', `Đã từ chối đề xuất bảo trì ${scopeDisplayName(payload.productCode, payload.skuCode)}.`);
      await refreshScopeMaintenance(payload, 'rejected');
      setBusyScopeKey(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Từ chối bảo trì thất bại');
      setBusyScopeKey(null);
    },
  });

  const reject = useMutation({
    mutationFn: (id: number) => api(`/routing-plans/${id}/reject`, { method: 'POST' }),
    onMutate: (id) => {
      setBusyPlanId(id);
    },
    onSuccess: async (_, id) => {
      showToast('ok', `Đã từ chối kế hoạch routing ${scopeNameByPlanId(id)}.`);
      await refreshOverview(id, 'rejected');
      setBusyPlanId(null);
    },
    onError: (e) => {
      showToast('err', e instanceof ApiClientError ? e.message : 'Từ chối thất bại');
      setBusyPlanId(null);
    },
  });

  return (
    <div className="page dashboard">
      <section className="page__hero">
        <h1>Dashboard</h1>
        <div className="hero-health">
          <HealthBadge status={overallHealth.status} label={overallHealth.label} size="lg" />
          {overallHealth.summary && <p>{overallHealth.summary}</p>}
        </div>
      </section>

      <section className="page__section page__section--routing">
        <div className="section-head section-head--dashboard">
          <nav className="service-tabs" aria-label="Loai dich vu">
            {SERVICE_TABS.map((tab) => {
              const count = rowsByService[tab.id]?.length ?? 0;
              const meta = serviceTabMeta[tab.id] ?? { status: 'green', pendingApprove: false };
              const tabStatus = meta.status;
              const tabTitle = meta.pendingApprove
                ? `${tab.label} — Có đề xuất chờ duyệt`
                : `${tab.label} — ${tabStatus === 'green' ? 'Ổn định' : tabStatus === 'yellow' ? 'Đang theo dõi' : 'Có vấn đề'}`;
              return (
                <button
                  key={tab.id}
                  type="button"
                  className={`service-tabs__btn${serviceTab === tab.id ? ' service-tabs__btn--active' : ''}`}
                  onClick={() => setServiceTab(tab.id)}
                  title={tabTitle}
                >
                  <HealthBadge status={tabStatus} compact size="sm" />
                  <span>{tab.label}</span>
                  <span className="service-tabs__count">{count}</span>
                </button>
              );
            })}
          </nav>
          <div className="section-head__aside">
            {overview?.updated_at && (
              <p className="section-head__meta muted">
                Cập nhật: {new Date(overview.updated_at).toLocaleString('vi-VN')} · tự làm mới mỗi 60 giây
                {overviewFetching && !overviewLoading ? ' · đang cập nhật…' : ''}
              </p>
            )}
          </div>
        </div>
        <RedSkuScrollNav rows={filteredRows} thresholdsByProduct={thresholdsByProduct} />
        {overviewLoading ? (
          <p className="loading">Đang tải bảng routing…</p>
        ) : (
          <ServiceOverviewTable
            rows={filteredRows}
            providers={overview?.providers}
            thresholdsByProduct={thresholdsByProduct}
            updatedAt={overview?.updated_at}
            busyPlanId={busyPlanId}
            busyMaintId={busyMaintId}
            busyScopeKey={busyScopeKey}
            busyProductKey={busyProductKey}
            planActions={planActions}
            maintActions={maintActions}
            scopeDone={scopeDone}
            maintScopeDone={maintScopeDone}
            onApprove={(payload) => approve.mutate(payload)}
            onReject={(id) => reject.mutate(id)}
            onApproveScope={(payload) => approveScope.mutate(payload)}
            onRejectScope={(payload) => rejectScope.mutate(payload)}
            onRestoreProviderRouting={(payload) => restoreScopeBaseline.mutate(payload)}
            onApplyScopeRouting={(payload) => applyScopeRouting.mutate(payload)}
            onApproveMaintenance={(payload) => approveMaintenance.mutate(payload)}
            onRejectMaintenance={(id) => rejectMaintenance.mutate(id)}
            onApproveScopeMaintenance={(payload) => approveScopeMaintenance.mutate(payload)}
            onApproveProductMaintenance={(payload) => approveProductMaintenance.mutate(payload)}
            onReopenProductMaintenance={(payload) => reopenProductMaintenance.mutate(payload)}
            onExtendProductMaintenance={(payload) => extendProductMaintenance.mutate(payload)}
            onRejectScopeMaintenance={(payload) => rejectScopeMaintenance.mutate(payload)}
            onReopenMaintenance={(payload) => reopenMaintenance.mutate(payload)}
            onExtendMaintenance={(payload) => extendMaintenance.mutate(payload)}
          />
        )}
      </section>
    </div>
  );
}
