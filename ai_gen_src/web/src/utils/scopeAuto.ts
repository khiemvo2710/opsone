import type { DashboardOverviewRow } from '../types/api';

type ScopeAutoRow = Pick<DashboardOverviewRow, 'auto_action' | 'window_start' | 'window_end'>;

function parseScopeDateTime(s: string): Date | null {
  const raw = s.trim();
  if (!raw) return null;
  const d = new Date(raw);
  if (Number.isFinite(d.getTime())) return d;
  const m = raw.match(/^(\d{1,2}):(\d{2})(?::(\d{2}))?$/);
  if (!m) return null;
  const now = new Date();
  return new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate(),
    Number(m[1]),
    Number(m[2]),
    m[3] ? Number(m[3]) : 0,
  );
}

export function inAutoTimeWindow(windowStart?: string, windowEnd?: string): boolean {
  const st = parseScopeDateTime(windowStart ?? '');
  const en = parseScopeDateTime(windowEnd ?? '');
  if (!st || !en) return false;
  const now = new Date();
  return now >= st && now < en;
}

export function shouldAutoApplyScope(row: ScopeAutoRow): boolean {
  const action =
    row.auto_action === 'auto' || row.auto_action === 'time_window'
      ? row.auto_action
      : 'recommend_only';
  if (action === 'auto') return true;
  if (action === 'time_window') return inAutoTimeWindow(row.window_start, row.window_end);
  return false;
}

/** Chỉ hiện đề xuất routing/bảo trì khi scope không ở chế độ tự động (hoặc ngoài khung giờ). */
export function shouldShowManualApproval(row: ScopeAutoRow): boolean {
  return !shouldAutoApplyScope(row);
}
