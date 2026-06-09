import { act, renderHook } from '@testing-library/react';
import { useBidHeat } from '../useBidHeat';

describe('useBidHeat', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-06-10T00:00:00.000Z'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('starts in calm state', () => {
    const { result } = renderHook(() => useBidHeat());

    expect(result.current.level).toBe('calm');
  });

  it('warms up after two bids in the sliding window', () => {
    const { result } = renderHook(() => useBidHeat());

    act(() => {
      result.current.markBid();
      result.current.markBid();
    });

    expect(result.current.level).toBe('warming');
  });

  it('becomes blazing after five bids in the sliding window', () => {
    const { result } = renderHook(() => useBidHeat());

    act(() => {
      for (let i = 0; i < 5; i += 1) {
        result.current.markBid();
      }
    });

    expect(result.current.level).toBe('blazing');
  });

  it('decays back to calm after the 10 second window expires', () => {
    const { result } = renderHook(() => useBidHeat());

    act(() => {
      for (let i = 0; i < 5; i += 1) {
        result.current.markBid();
      }
    });
    expect(result.current.level).toBe('blazing');

    act(() => {
      jest.advanceTimersByTime(10000);
    });

    expect(result.current.level).toBe('calm');
  });

  it('resets heat immediately', () => {
    const { result } = renderHook(() => useBidHeat());

    act(() => {
      for (let i = 0; i < 5; i += 1) {
        result.current.markBid();
      }
    });
    expect(result.current.level).toBe('blazing');

    act(() => {
      result.current.reset();
    });

    expect(result.current.level).toBe('calm');
  });
});
