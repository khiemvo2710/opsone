import { createContext, useContext, useState, type ReactNode } from 'react';
import type { DashboardOverviewRow, ProductThreshold } from '../types/api';

interface ScrollNavData {
  rows: DashboardOverviewRow[];
  thresholdsByProduct: Record<string, ProductThreshold>;
}

interface ScrollNavContextValue {
  data: ScrollNavData | null;
  setData: (rows: DashboardOverviewRow[], thresholds: Record<string, ProductThreshold>) => void;
}

const ScrollNavContext = createContext<ScrollNavContextValue | null>(null);

export function ScrollNavProvider({ children }: { children: ReactNode }) {
  const [data, setDataState] = useState<ScrollNavData | null>(null);

  const setData = (
    rows: DashboardOverviewRow[],
    thresholdsByProduct: Record<string, ProductThreshold>,
  ) => {
    setDataState({ rows, thresholdsByProduct });
  };

  return (
    <ScrollNavContext.Provider value={{ data, setData }}>
      {children}
    </ScrollNavContext.Provider>
  );
}

export function useScrollNav() {
  const ctx = useContext(ScrollNavContext);
  if (!ctx) throw new Error('useScrollNav must be used within ScrollNavProvider');
  return ctx;
}
