import { useMemo } from 'react';
import { useQueries } from '@tanstack/react-query';
import { api } from '../api/client';
import type { ProductThreshold } from '../types/api';

export function useProductThresholds(
  productCodes: string[],
  embedded?: Record<string, ProductThreshold>,
): Record<string, ProductThreshold> {
  const codes = useMemo(
    () => [...new Set(productCodes)].sort(),
    [productCodes.join('|')],
  );

  const missingCodes = useMemo(
    () => codes.filter((code) => !embedded?.[code]),
    [codes, embedded],
  );

  const queries = useQueries({
    queries: missingCodes.map((code) => ({
      queryKey: ['threshold', code],
      queryFn: () => api<ProductThreshold>(`/products/${code}/thresholds`),
      staleTime: 120_000,
      enabled: missingCodes.length > 0,
    })),
  });

  return useMemo(() => {
    const map: Record<string, ProductThreshold> = { ...embedded };
    missingCodes.forEach((code, i) => {
      const data = queries[i]?.data;
      if (data) map[code] = data;
    });
    return map;
  }, [embedded, missingCodes, queries]);
}
