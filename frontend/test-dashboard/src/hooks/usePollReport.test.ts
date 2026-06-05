import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { getReport, type TestResult } from '@/api/test';
import { usePollReport } from './usePollReport';

vi.mock('@/api/test', () => ({
  getReport: vi.fn(),
}));

const getReportMock = vi.mocked(getReport);

describe('usePollReport', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('routes failed report with empty ResultJSON to onError using ErrorMsg', async () => {
    getReportMock.mockResolvedValue(
      report({
        Status: 'failed',
        ResultJSON: '',
        ErrorMsg: 'sky_lamp failed: upstream timeout',
      }),
    );
    const onResult = vi.fn();
    const onError = vi.fn();
    const { result } = renderHook(() => usePollReport<Record<string, unknown>>({ maxAttempts: 1, intervalMs: 10 }));

    act(() => {
      result.current.start('tj_fail', onResult, onError);
    });
    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });

    expect(onError).toHaveBeenCalledWith('sky_lamp failed: upstream timeout');
    expect(onResult).not.toHaveBeenCalled();
  });
});

function report(overrides: Partial<TestResult>): TestResult {
  return {
    ID: 'tj_fail',
    TestType: 'user_journey',
    Status: 'running',
    ConfigJSON: '{}',
    ResultJSON: '{}',
    ReplayToken: '',
    ScriptName: '',
    ErrorMsg: '',
    CreatedAt: new Date(0).toISOString(),
    CompletedAt: null,
    ...overrides,
  };
}
