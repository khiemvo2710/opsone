import { pca, loginRequest, devAuthBypass } from '../auth/msalConfig';
import type { ApiError } from '../types/api';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api/v1';

function mockSessionActor(): string | null {
  try {
    const raw = localStorage.getItem('opsone_mock_session');
    if (!raw) return null;
    const s = JSON.parse(raw) as { name?: string };
    return s.name?.trim() || null;
  } catch {
    return null;
  }
}

async function authHeaders(): Promise<Record<string, string>> {
  if (devAuthBypass) {
    const actor = mockSessionActor() ?? 'khiemvt';
    return {
      'X-OpsOne-Role': 'admin',
      'X-OpsOne-Actor': actor,
    };
  }
  const account = pca.getActiveAccount() ?? pca.getAllAccounts()[0];
  if (!account) {
    throw new Error('Chưa đăng nhập');
  }
  const res = await pca.acquireTokenSilent({ ...loginRequest, account });
  return { Authorization: `Bearer ${res.accessToken}` };
}

export class ApiClientError extends Error {
  code: string;
  constructor(err: ApiError) {
    super(err.message_vi);
    this.code = err.code;
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(await authHeaders()),
    ...(init.headers as Record<string, string> | undefined),
  };
  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (!res.ok) {
    const err = (await res.json().catch(() => ({
      code: 'http_error',
      message_vi: res.statusText,
    }))) as ApiError;
    throw new ApiClientError(err);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

export function eventsUrl(): string {
  return `${API_BASE}/events`;
}
