import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, jest, beforeEach, afterEach } from '@jest/globals';
import { useReconnect } from '../useReconnect';

describe('useReconnect Hook', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should initialize with default options', () => {
    const { result } = renderHook(() => useReconnect());

    expect(result.current.reconnectCount).toBe(0);
    expect(result.current.isReconnecting).toBe(false);
    expect(result.current.maxAttempts).toBe(10);
  });

  it('should start reconnecting', () => {
    const { result } = renderHook(() => useReconnect());

    act(() => {
      result.current.startReconnect();
    });

    expect(result.current.isReconnecting).toBe(true);
    expect(result.current.reconnectCount).toBe(1);
  });

  it('should stop reconnecting after max attempts', () => {
    const onMaxAttemptsReached = jest.fn();
    const { result } = renderHook(() =>
      useReconnect({
        maxAttempts: 3,
        onMaxAttemptsReached,
      })
    );

    // Start reconnecting 3 times
    act(() => {
      result.current.startReconnect();
      result.current.startReconnect();
      result.current.startReconnect();
    });

    // Fourth attempt should fail and call onMaxAttemptsReached
    act(() => {
      const success = result.current.startReconnect();
      expect(success).toBe(false);
    });

    expect(onMaxAttemptsReached).toHaveBeenCalled();
  });

  it('should reset reconnect count', () => {
    const { result } = renderHook(() => useReconnect());

    act(() => {
      result.current.startReconnect();
      result.current.startReconnect();
    });

    expect(result.current.reconnectCount).toBe(2);

    act(() => {
      result.current.resetReconnect();
    });

    expect(result.current.reconnectCount).toBe(0);
    expect(result.current.isReconnecting).toBe(false);
  });

  it('should stop reconnecting', () => {
    const { result } = renderHook(() => useReconnect());

    act(() => {
      result.current.startReconnect();
    });

    expect(result.current.isReconnecting).toBe(true);

    act(() => {
      result.current.stopReconnect();
    });

    expect(result.current.isReconnecting).toBe(false);
  });

  it('should call onReconnect callback after delay', () => {
    const onReconnect = jest.fn();
    const { result } = renderHook(() =>
      useReconnect({ onReconnect })
    );

    act(() => {
      result.current.startReconnect();
    });

    // Fast-forward time
    act(() => {
      jest.advanceTimersByTime(1000);
    });

    expect(onReconnect).toHaveBeenCalled();
  });

  it('should use custom max attempts', () => {
    const { result } = renderHook(() =>
      useReconnect({ maxAttempts: 5 })
    );

    expect(result.current.maxAttempts).toBe(5);
  });
});
