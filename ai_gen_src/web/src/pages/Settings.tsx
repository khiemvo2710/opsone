import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { AgentConfig } from '../types/api';

const DEFAULT_MAINTENANCE_MIN = 60;

const MOCK_SCENARIOS = [
  { value: 'normal', label: 'Bình thường — nhiễu nhẹ, không sự cố lớn' },
  { value: 'esale_degrading', label: 'ESALE suy giảm — TOPUP_VINA + ZING SKU 20k' },
  { value: 'imedia_garena_pending', label: 'IMEDIA treo Garena 10k — nhiều GD pending' },
  { value: 'sku_local_fault', label: 'Lỗi cục bộ SKU — DATA_VINA VNP50 / ESALE' },
  { value: 'random_spike', label: 'Đột biến lỗi — spike ngẫu nhiên, hồi sau 5–10 phút' },
] as const;

function normalizeConfig(config: AgentConfig): AgentConfig {
  const duration = config.maintenance_default_duration_min;
  return {
    ...config,
    maintenance_default_duration_min:
      duration != null && duration > 0 ? duration : DEFAULT_MAINTENANCE_MIN,
  };
}

export function Settings() {
  const qc = useQueryClient();
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: () => api<AgentConfig>('/config'),
  });

  const [form, setForm] = useState<AgentConfig | null>(null);
  const [saved, setSaved] = useState('');

  useEffect(() => {
    if (config) setForm(normalizeConfig(config));
  }, [config]);

  const saveConfig = useMutation({
    mutationFn: (body: Partial<AgentConfig>) =>
      api<AgentConfig>('/config', { method: 'PUT', body: JSON.stringify(body) }),
    onSuccess: (data) => {
      setForm(normalizeConfig(data));
      void qc.invalidateQueries({ queryKey: ['config'] });
      setSaved('Đã lưu cấu hình.');
    },
  });

  if (isLoading || !form) {
    return <p className="loading">Đang tải cấu hình...</p>;
  }

  const update = <K extends keyof AgentConfig>(key: K, value: AgentConfig[K]) => {
    setForm((f) => (f ? { ...f, [key]: value } : f));
  };

  const save = () => {
    const duration = form.maintenance_default_duration_min;
    saveConfig.mutate({
      scheduler_enabled: form.scheduler_enabled,
      scheduler_interval_min: form.scheduler_interval_min,
      mock_enabled: form.mock_enabled,
      mock_scenario: form.mock_scenario,
      maintenance_default_duration_min:
        duration > 0 ? duration : DEFAULT_MAINTENANCE_MIN,
    });
  };

  return (
    <div className="page settings-page">
      <header className="settings-page__header">
        <h1>Cấu hình</h1>
        {saved ? <p className="success settings-page__saved">{saved}</p> : null}
        <p className="muted settings-page__intro">
          Ngưỡng &amp; chế độ Auto → <strong>Dashboard</strong>.
        </p>
      </header>

      <div className="settings-card">
        <div className="settings-group">
          <h2 className="settings-group__title">Scheduler</h2>
          <label className="settings-row settings-row--checkbox">
            <input
              type="checkbox"
              checked={form.scheduler_enabled}
              onChange={(e) => update('scheduler_enabled', e.target.checked)}
            />
            <span className="settings-row__label">Bật scheduler</span>
          </label>
          <label className="settings-row">
            <span className="settings-row__label">Chu kỳ phân tích (phút)</span>
            <input
              className="settings-input--num"
              type="number"
              min={1}
              max={60}
              value={form.scheduler_interval_min}
              onChange={(e) => update('scheduler_interval_min', Number(e.target.value))}
            />
          </label>
        </div>

        <div className="settings-group">
          <h2 className="settings-group__title">Bảo trì</h2>
          <label className="settings-row">
            <span className="settings-row__label">Thời lượng mặc định (phút)</span>
            <input
              className="settings-input--num"
              type="number"
              min={1}
              max={255}
              value={form.maintenance_default_duration_min}
              onChange={(e) =>
                update('maintenance_default_duration_min', Number(e.target.value))
              }
            />
          </label>
        </div>

        <div className="settings-group">
          <h2 className="settings-group__title">Mock data</h2>
          <label className="settings-row settings-row--checkbox">
            <input
              type="checkbox"
              checked={form.mock_enabled}
              onChange={(e) => update('mock_enabled', e.target.checked)}
            />
            <span className="settings-row__label">Bật mock data</span>
          </label>
          <label className="settings-row settings-row--select">
            <span className="settings-row__label">Kịch bản</span>
            <select
              className="settings-select"
              value={form.mock_scenario}
              onChange={(e) => update('mock_scenario', e.target.value)}
            >
              {MOCK_SCENARIOS.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </select>
          </label>
          <p className="muted settings-page__meta">Nguồn: {form.data_source}</p>
        </div>
      </div>

      <button
        type="button"
        className="btn btn--primary settings-page__save"
        disabled={saveConfig.isPending}
        onClick={save}
      >
        Lưu cấu hình
      </button>
    </div>
  );
}