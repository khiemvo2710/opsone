import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';

export interface AuthSession {
  name: string;       // tên user nhập
  micAllowed: boolean;
}

interface AuthContextValue {
  session: AuthSession | null;
  login: (name: string, micAllowed: boolean) => void;
  logout: () => void;
}

const STORAGE_KEY = 'opsone_mock_session';

export function chatHistoryStorageKey(name: string): string {
  return `opsone_chat_${name.trim().toLowerCase().replace(/\s+/g, '_')}`;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<AuthSession | null>(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      return raw ? (JSON.parse(raw) as AuthSession) : null;
    } catch {
      return null;
    }
  });

  useEffect(() => {
    if (session) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  }, [session]);

  function login(name: string, micAllowed: boolean) {
    setSession({ name: name.trim(), micAllowed });
  }

  function logout() {
    if (session) {
      localStorage.removeItem(chatHistoryStorageKey(session.name));
    }
    setSession(null);
  }

  return (
    <AuthContext.Provider value={{ session, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider');
  return ctx;
}
