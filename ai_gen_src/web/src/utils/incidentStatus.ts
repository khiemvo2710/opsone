export const INCIDENT_STATUS_LABEL: Record<string, string> = {
  open: 'Đang mở',
  resolved: 'Đã xử lý',
  acknowledged: 'Đã từ chối',
};

export const INCIDENT_RESOLUTION_LABEL: Record<string, string> = {
  admin_approve: 'Duyệt routing',
  admin_reject: 'Từ chối routing',
  auto: 'Tự động',
};

export function incidentStatusLabel(status: string): string {
  return INCIDENT_STATUS_LABEL[status] ?? status;
}

export function incidentResolutionLabel(action?: string): string | null {
  if (!action) return null;
  return INCIDENT_RESOLUTION_LABEL[action] ?? action;
}

export function formatIncidentHandled(inc: {
  status: string;
  handled_by?: string;
  handled_at?: string;
  resolution_action?: string;
}): string | null {
  if (inc.status === 'open') return null;
  const parts: string[] = [];
  const action = incidentResolutionLabel(inc.resolution_action);
  if (action) parts.push(action);
  if (inc.handled_by) parts.push(inc.handled_by);
  if (inc.handled_at) parts.push(new Date(inc.handled_at).toLocaleString('vi-VN'));
  return parts.length > 0 ? parts.join(' · ') : null;
}

export function formatHandledBy(handledBy?: string, resolutionAction?: string): string {
  if (handledBy) return handledBy;
  if (resolutionAction === 'auto') return 'opsone-agent';
  return '—';
}

export function formatHandledAt(handledAt?: string, status?: string): string {
  if (handledAt) return new Date(handledAt).toLocaleString('vi-VN');
  if (status && status !== 'open') return '—';
  return '—';
}
