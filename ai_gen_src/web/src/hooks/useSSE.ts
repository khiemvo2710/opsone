import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { eventsUrl } from '../api/client';

export function useSSE(enabled = true) {
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!enabled) return;

    let es: EventSource | null = null;
    let pollTimer: ReturnType<typeof setInterval> | null = null;
    let closed = false;

    const invalidate = () => {
      void queryClient.invalidateQueries({ queryKey: ['health-status'] });
      void queryClient.invalidateQueries({ queryKey: ['dashboard-overview'] });
      void queryClient.invalidateQueries({ queryKey: ['incidents'] });
      void queryClient.invalidateQueries({ queryKey: ['routing-plans'] });
      void queryClient.invalidateQueries({ queryKey: ['maintenance'] });
    };

    const startPoll = () => {
      if (pollTimer) return;
      pollTimer = setInterval(invalidate, 30_000);
    };

    try {
      es = new EventSource(eventsUrl());
      es.addEventListener('cycle_finished', invalidate);
      es.addEventListener('health_status', invalidate);
      es.onerror = () => {
        es?.close();
        es = null;
        startPoll();
      };
    } catch {
      startPoll();
    }

    return () => {
      closed = true;
      es?.close();
      if (pollTimer) clearInterval(pollTimer);
      if (closed) {
        /* noop — satisfy lint */
      }
    };
  }, [enabled, queryClient]);
}
