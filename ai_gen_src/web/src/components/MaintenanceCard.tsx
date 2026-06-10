import { useEffect, useState } from 'react';
import type { MaintenanceWindow } from '../types/api';

function remainingMinutes(endsAt: string): number {
  const diff = new Date(endsAt).getTime() - Date.now();
  return Math.max(0, Math.ceil(diff / 60_000));
}

interface Props {
  window: MaintenanceWindow;
}

export function MaintenanceCard({ window: mw }: Props) {
  const [mins, setMins] = useState(remainingMinutes(mw.ends_at));

  useEffect(() => {
    const t = setInterval(() => setMins(remainingMinutes(mw.ends_at)), 30_000);
    return () => clearInterval(t);
  }, [mw.ends_at]);

  const active = mw.status === 'active' || mw.status === 'approved';

  return (
    <article className="card maintenance-card">
      <header className="card__header">
        <h3>Bảo trì {mw.product_code}</h3>
        <span className={`badge badge--${mw.status}`}>{mw.status}</span>
      </header>
      <p className="card__meta">
        {mw.provider_code}
        {mw.sku_code ? ` · SKU ${mw.sku_code}` : ''}
      </p>
      <p className="card__body">
        {new Date(mw.starts_at).toLocaleString('vi-VN')} → {new Date(mw.ends_at).toLocaleString('vi-VN')}
      </p>
      {mw.reason && <p className="card__body">{mw.reason}</p>}
      {active && <p className="countdown">Còn {mins} phút</p>}
    </article>
  );
}
