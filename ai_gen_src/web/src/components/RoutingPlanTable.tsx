import type { RoutingPlan } from '../types/api';

interface Props {
  plan: RoutingPlan;
  onApprove?: (id: number) => void;
  onReject?: (id: number) => void;
  busy?: boolean;
}

export function RoutingPlanTable({ plan, onApprove, onReject, busy }: Props) {
  const proposed = plan.plan?.proposed_pct ?? {};
  const current = plan.plan?.current_pct ?? {};
  const providers = Array.from(new Set([...Object.keys(current), ...Object.keys(proposed)]));

  const canAct = plan.status === 'pending_approve' || plan.status === 'draft';

  return (
    <article className="card routing-plan">
      <header className="card__header">
        <h3>
          Routing — {plan.product_code}
          {plan.sku_code ? ` / ${plan.sku_code}` : ''}
        </h3>
        <span className={`badge badge--${plan.status}`}>{plan.status}</span>
      </header>

      <div className="table-scroll">
        <table className="data-table">
          <thead>
            <tr>
              <th>Provider</th>
              <th>Hiện tại</th>
              <th>Đề xuất</th>
            </tr>
          </thead>
          <tbody>
            {providers.map((p) => (
              <tr key={p}>
                <td>{p}</td>
                <td>{current[p] != null ? `${current[p]}%` : '—'}</td>
                <td>{proposed[p] != null ? `${proposed[p]}%` : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {plan.plan?.reason_vi && <p className="card__body">{plan.plan.reason_vi}</p>}

      {canAct && onApprove && onReject && (
        <footer className="card__actions">
          <button type="button" className="btn btn--primary" disabled={busy} onClick={() => onApprove(plan.id)}>
            Duyệt
          </button>
          <button type="button" className="btn btn--ghost" disabled={busy} onClick={() => onReject(plan.id)}>
            Từ chối
          </button>
        </footer>
      )}
    </article>
  );
}
