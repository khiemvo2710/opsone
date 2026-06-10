import type { HealthStatus } from '../types/api';

const ICON: Record<string, string> = {
  green: '🟢',
  yellow: '🟡',
  red: '🔴',
};

const LABEL: Record<string, string> = {
  green: 'Hệ thống OK',
  yellow: 'Đang theo dõi / xử lý',
  red: 'Đang có vấn đề',
};

interface Props {
  status: HealthStatus;
  label?: string;
  size?: 'sm' | 'md' | 'lg';
  /** Chỉ icon — dùng sidebar / bảng tổng quan */
  compact?: boolean;
}

export function HealthBadge({ status, label, size = 'md', compact = false }: Props) {
  const key = status?.toLowerCase() ?? 'green';
  const icon = ICON[key] ?? '🟢';
  const text = label ?? LABEL[key] ?? status;

  return (
    <span
      className={`health-badge health-badge--${size} health-badge--${key}${compact ? ' health-badge--compact' : ''}`}
      title={compact ? text : undefined}
    >
      <span className="health-badge__icon" aria-hidden>
        {icon}
      </span>
      {!compact && <span className="health-badge__text">{text}</span>}
    </span>
  );
}
