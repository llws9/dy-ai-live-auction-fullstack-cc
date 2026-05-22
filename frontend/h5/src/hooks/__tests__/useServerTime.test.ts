import { describe, it, expect, jest, beforeEach, afterEach } from '@jest/globals';
import { renderHook, act } from '@testing-library/react';
import { useServerTime } from '../useServerTime';

describe('useServerTime Hook', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should format countdown correctly for zero', () => {
    const { result } = renderHook(() => useServerTime(0));

    // Zero or negative should show 00:00:00.000
    expect(result.current.formatCountdown(0)).toBe('00:00:00.000');
    expect(result.current.formatCountdown(-100)).toBe('00:00:00.000');
  });

  it('should format milliseconds correctly', () => {
    const { result } = renderHook(() => useServerTime(0));

    // 1 hour, 23 minutes, 45 seconds, 678 milliseconds
    const ms = 1 * 3600000 + 23 * 60000 + 45 * 1000 + 678;
    expect(result.current.formatCountdown(ms)).toBe('01:23:45.678');
  });

  it('should format seconds correctly', () => {
    const { result } = renderHook(() => useServerTime(0));

    // 5 seconds
    expect(result.current.formatCountdown(5000)).toBe('00:00:05.000');

    // 1 minute
    expect(result.current.formatCountdown(60000)).toBe('00:01:00.000');

    // 1 hour
    expect(result.current.formatCountdown(3600000)).toBe('01:00:00.000');
  });

  it('should call onTimeSync callback', () => {
    const onTimeSync = jest.fn();
    const { result } = renderHook(() =>
      useServerTime(Date.now() + 60000, { onTimeSync })
    );

    const serverTime = Date.now() + 5000;
    act(() => {
      result.current.syncServerTime(serverTime);
    });

    expect(onTimeSync).toHaveBeenCalledWith(serverTime);
  });

  it('should apply server time offset', () => {
    const { result } = renderHook(() => useServerTime(Date.now() + 60000));

    // Simulate server time being 5 seconds ahead
    act(() => {
      result.current.syncServerTime(Date.now() + 5000);
    });

    // Offset should be approximately 5000ms
    expect(result.current.serverTimeOffset).toBeGreaterThanOrEqual(4990);
    expect(result.current.serverTimeOffset).toBeLessThanOrEqual(5010);
  });

  it('should handle sync interval configuration', () => {
    const syncInterval = 5000;
    renderHook(() =>
      useServerTime(Date.now() + 60000, { syncInterval })
    );

    // Interval should be set (this tests the hook doesn't crash)
    expect(true).toBe(true);
  });
});
