import { useEffect, useRef, useCallback } from 'react';
import { getReport, type TestResult } from '@/api/test';

interface UsePollReportOptions {
  maxAttempts?: number;
  intervalMs?: number;
}

export function usePollReport<T = unknown>(
  options: UsePollReportOptions = {},
) {
  const { maxAttempts = 120, intervalMs = 1000 } = options;
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = useRef(0);

  const cancel = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    attemptRef.current = 0;
  }, []);

  useEffect(() => cancel, [cancel]);

  const start = useCallback(
    (testID: string, onResult: (r: T) => void, onError?: (msg: string) => void) => {
      cancel();
      attemptRef.current = 0;

      const tick = async () => {
        attemptRef.current += 1;
        try {
          const t = await getReport(testID);
          if (t.Status === 'completed' || t.Status === 'failed' || t.Status === 'cancelled') {
            try {
              onResult(JSON.parse(t.ResultJSON || '{}') as T);
            } catch {
              const msg = t.ErrorMsg || 'parse error';
              if (onError) onError(msg);
              else onResult({ error: msg } as T);
            }
            return;
          }
        } catch {
          // ignore transient errors
        }
        if (attemptRef.current < maxAttempts) {
          timerRef.current = setTimeout(tick, intervalMs);
        }
      };

      timerRef.current = setTimeout(tick, intervalMs);
    },
    [maxAttempts, intervalMs, cancel],
  );

  return { start, cancel };
}

export type { TestResult };
