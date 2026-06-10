import { Link } from 'react-router-dom';
import type { Incident } from '../types/api';
import { incidentStatusLabel, formatIncidentHandled } from '../utils/incidentStatus';
import { HealthBadge } from './HealthBadge';

const SEVERITY_STATUS: Record<string, string> = {
  critical: 'red',
  high: 'red',
  medium: 'yellow',
  low: 'yellow',
};

interface Props {
  incident: Incident;
}

export function IncidentCard({ incident }: Props) {
  const status = SEVERITY_STATUS[incident.severity] ?? 'yellow';

  return (
    <Link to={`/incidents/${incident.incident_id}`} className="card incident-card incident-card--link">
      <time className="incident-card__time" dateTime={incident.created_at}>
        {new Date(incident.created_at).toLocaleString('vi-VN')}
      </time>
      <header className="card__header">
        <HealthBadge status={status} label={incident.severity} size="sm" />
        <span className="card__id">#{incident.incident_id}</span>
      </header>
      <p className="card__meta">
        {incident.product_code}
        {incident.provider_code ? ` · ${incident.provider_code}` : ''}
        {incident.sku_code ? ` · SKU ${incident.sku_code}` : ''}
      </p>
      {incident.summary && <p className="card__body">{incident.summary}</p>}
      <footer className="card__footer">
        <span className={`badge badge--${incident.status}`}>{incidentStatusLabel(incident.status)}</span>
        {formatIncidentHandled(incident) && (
          <span className="muted incident-card__handled">{formatIncidentHandled(incident)}</span>
        )}
        <span className="incident-card__hint">Xem chi tiết →</span>
      </footer>
    </Link>
  );
}
