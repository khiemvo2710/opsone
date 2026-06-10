const DEFAULT_PROVIDERS = ['ESALE', 'IMEDIA', 'SHOPPAY'] as const;

/** Largest-remainder: whole % summing exactly to 100 (matches Go roundPctToInt100). */
export function roundPctToInt100(
  values: Record<string, number>,
  providers: readonly string[],
): Record<string, number> {
  type Row = { p: string; base: number; rem: number };
  const rows: Row[] = providers.map((p) => {
    const v = Math.max(0, values[p] ?? 0);
    const base = Math.floor(v);
    return { p, base, rem: v - base };
  });
  let sum = rows.reduce((s, r) => s + r.base, 0);
  let diff = 100 - sum;
  rows.sort((a, b) => (diff > 0 ? b.rem - a.rem : a.rem - b.rem));
  const n = rows.length;
  for (let i = 0; i < Math.abs(diff) && n > 0; i++) {
    const idx = i % n;
    if (diff > 0) {
      rows[idx].base++;
    } else if (rows[idx].base > 0) {
      rows[idx].base--;
    }
  }
  const out: Record<string, number> = {};
  for (const r of rows) {
    out[r.p] = r.base;
  }
  return out;
}

/** Baseline biz → ô nhập % (tổng = 100). */
export function baselineRoutingPct(
  baseline: Record<string, number> | undefined,
  providers: readonly string[] = DEFAULT_PROVIDERS,
): Record<string, number> | null {
  if (!baseline || Object.keys(baseline).length === 0) return null;
  const raw: Record<string, number> = {};
  for (const p of providers) {
    raw[p] = baseline[p] ?? 0;
  }
  return roundPctToInt100(raw, providers);
}

export function initialRoutingPct(
  proposed: Record<string, number> | undefined,
  current: Record<string, number> | undefined,
  providers: readonly string[] = DEFAULT_PROVIDERS,
): Record<string, number> {
  const raw: Record<string, number> = {};
  for (const p of providers) {
    const v = proposed?.[p] ?? current?.[p];
    raw[p] = v != null ? v : 0;
  }
  return roundPctToInt100(raw, providers);
}

export function routingPctSum(values: Record<string, number>, providers: readonly string[]): number {
  return providers.reduce((s, p) => s + (values[p] ?? 0), 0);
}

/** Validate routing §8.6.3 — mỗi provider 0–100%, tổng = 100%. */
export const ROUTING_PCT_MAX = 100;

export function routingPctValidationError(
  values: Record<string, number>,
  providers: readonly string[],
): string | null {
  const sum = Math.round(routingPctSum(values, providers));
  if (sum !== 100) {
    return `Tổng phải = 100% (hiện ${sum}%)`;
  }
  for (const p of providers) {
    const v = values[p] ?? 0;
    if (v < 0 || v > ROUTING_PCT_MAX) {
      return `${p} phải trong 0–100% (hiện ${v}%)`;
    }
  }
  return null;
}

export function isRoutingPctValid(values: Record<string, number>, providers: readonly string[]): boolean {
  return routingPctValidationError(values, providers) === null;
}

export function isRoutingProviderSupported(
  provider: string,
  routingPct: Record<string, number> | undefined,
  plan?: { proposed_pct?: Record<string, number>; current_pct?: Record<string, number> },
): boolean {
  if (routingPct != null && provider in routingPct) return true;
  if (plan?.proposed_pct != null && provider in plan.proposed_pct) return true;
  if (plan?.current_pct != null && provider in plan.current_pct) return true;
  return false;
}

export function activeRoutingProviders(
  providers: readonly string[],
  routingPct: Record<string, number> | undefined,
  plan?: { proposed_pct?: Record<string, number>; current_pct?: Record<string, number> },
): string[] {
  return providers.filter((p) => isRoutingProviderSupported(p, routingPct, plan));
}

export function routingPctFieldInvalid(
  provider: string,
  values: Record<string, number>,
  _providers: readonly string[] = DEFAULT_PROVIDERS,
): boolean {
  const v = values[provider] ?? 0;
  return v < 0 || v > ROUTING_PCT_MAX;
}

export function routingPctMapsEqual(
  a: Record<string, number>,
  b: Record<string, number>,
  providers: readonly string[],
): boolean {
  for (const p of providers) {
    if (Math.round(a[p] ?? 0) !== Math.round(b[p] ?? 0)) return false;
  }
  return true;
}

interface Props {
  providers?: readonly string[];
  values: Record<string, number>;
  onChange: (values: Record<string, number>) => void;
  disabled?: boolean;
  validationError?: string | null;
  compact?: boolean;
  isSupported?: (provider: string) => boolean;
  readOnly?: boolean;
}

export function RoutingPctEditor({
  providers = DEFAULT_PROVIDERS,
  values,
  onChange,
  disabled,
  validationError,
  compact = false,
  isSupported,
  readOnly = false,
}: Props) {
  const activeProviders = providers.filter((p) => (isSupported ? isSupported(p) : true));
  const sum = Math.round(routingPctSum(values, activeProviders));
  const sumOk = sum === 100;

  const fieldInvalid = (provider: string) => routingPctFieldInvalid(provider, values, providers);

  const setProvider = (provider: string, raw: string) => {
    const n = raw === '' ? 0 : Number.parseFloat(raw);
    onChange({
      ...values,
      [provider]: Number.isFinite(n) ? Math.round(n) : 0,
    });
  };

  return (
    <div className={`plan-routing-edit${compact ? ' plan-routing-edit--compact' : ''}`}>
      {providers.map((p) => {
        const supported = isSupported ? isSupported(p) : true;
        return (
          <label key={p} className="plan-routing-edit__field">
            <span>{p}</span>
            {supported ? (
              readOnly ? (
                <span className="plan-routing-edit__value">{values[p] ?? 0}</span>
              ) : (
                <input
                  type="number"
                  className={`plan-routing-edit__input${fieldInvalid(p) ? ' plan-routing-edit__input--bad' : ''}`}
                  min={0}
                  max={ROUTING_PCT_MAX}
                  step={1}
                  disabled={disabled}
                  value={values[p] ?? ''}
                  onChange={(e) => setProvider(p, e.target.value)}
                />
              )
            ) : (
              <span className="plan-routing-edit__empty">—</span>
            )}
            <span className="muted">%</span>
          </label>
        );
      })}
      {!compact && (
        <span className={`plan-routing-edit__sum${sumOk ? '' : ' plan-routing-edit__sum--bad'}`}>
          Σ {sum}%
        </span>
      )}
      {validationError && (
        <span className="plan-routing-edit__error">{validationError}</span>
      )}
    </div>
  );
}
