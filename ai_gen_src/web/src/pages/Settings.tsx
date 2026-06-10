import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { AgentConfig } from '../types/api';

const MOCK_SCENARIOS = [
  { value: 'normal', label: 'Bình thường — nhiễu nhẹ, không sự cố lớn' },
  { value: 'esale_degrading', label: 'ESALE suy giảm — TOPUP_VINA + ZING SKU 20k' },
  { value: 'sku_local_fault', label: 'Lỗi cục bộ SKU — DATA_VINA VNP50 / ESALE' },
  { value: 'random_spike', label: 'Đột biến lỗi — spike ngẫu nhiên, hồi sau 5–10 phút' },
] as const;

export function Settings() {
  const qc = useQueryClient();
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: () => api<AgentConfig>('/config'),
  });

  const [form, setForm] = useState<AgentConfig | null>(null);
  const [saved, setSaved] = useState('');

  useEffect(() => {
    if (config) setForm(config);
  }, [config]);

  const saveConfig = useMutation({
    mutationFn: (body: Partial<AgentConfig>) =>
      api<AgentConfig>('/config', { method: 'PUT', body: JSON.stringify(body) }),
    onSuccess: (data) => {
      setForm(data);
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
    saveConfig.mutate({
      scheduler_enabled: form.scheduler_enabled,
      scheduler_interval_min: form.scheduler_interval_min,
      mock_enabled: form.mock_enabled,
      mock_scenario: form.mock_scenario,
    });
  };

  return (
    <div className="page settings-page">
      <h1>Cấu hình</h1>
      {saved && <p className="success">{saved}</p>}
      <p className="muted">
        Ngưỡng cảnh báo và chế độ <strong>Auto</strong> (Chỉ đề xuất / Tự động / Tự động theo khung giờ) cấu hình
        tại <strong>Dashboard → bảng routing</strong> — mỗi SKU một dòng.
      </p>

      <section className="settings-section">
        <h2>Scheduler</h2>
        <label className="field">
          <input
            type="checkbox"
            checked={form.scheduler_enabled}
            onChange={(e) => update('scheduler_enabled', e.target.checked)}
          />
          Bật scheduler
        </label>
        <label className="field">
          Chu kỳ phân tích (phút)
          <input
            type="number"
            min={1}
            max={60}
            value={form.scheduler_interval_min}
            onChange={(e) => update('scheduler_interval_min', Number(e.target.value))}
          />
        </label>
      </section>

      <section className="settings-section">
        <h2>Mock data</h2>
        <label className="field">
          <input
            type="checkbox"
            checked={form.mock_enabled}
            onChange={(e) => update('mock_enabled', e.target.checked)}
          />
          Bật mock data
        </label>
        <label className="field">
          Kịch bản mock
          <select value={form.mock_scenario} onChange={(e) => update('mock_scenario', e.target.value)}>
            {MOCK_SCENARIOS.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
        </label>
        <p className="muted">Nguồn dữ liệu: {form.data_source}</p>
      </section>

      <button type="button" className="btn btn--primary" disabled={saveConfig.isPending} onClick={save}>
        Lưu cấu hình
      </button>
    </div>
  );
}
