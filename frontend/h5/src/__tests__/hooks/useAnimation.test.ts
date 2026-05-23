import { renderHook, act } from '@testing-library/react';
import { useAnimation, useMountAnimation, useStaggerAnimation } from '@/hooks/useAnimation';

describe('useAnimation Hook', () => {
  it('returns initial state correctly', () => {
    const { result } = renderHook(() => useAnimation(false));
    expect(result.current.shouldRender).toBe(false);
    expect(result.current.animationClass).toBe('');
  });

  it('sets shouldRender to true when active', () => {
    const { result, rerender } = renderHook(
      ({ isActive }) => useAnimation(isActive),
      { initialProps: { isActive: false } }
    );

    expect(result.current.shouldRender).toBe(false);

    rerender({ isActive: true });
    expect(result.current.shouldRender).toBe(true);
    expect(result.current.animationClass).toBe('fadeIn');
  });

  it('uses custom enter and exit animations', () => {
    const { result, rerender } = renderHook(
      ({ isActive }) => useAnimation(isActive, 'slideUp', 'slideDown', 500),
      { initialProps: { isActive: false } }
    );

    rerender({ isActive: true });
    expect(result.current.animationClass).toBe('slideUp');
  });

  it('sets exit animation when deactivated', () => {
    jest.useFakeTimers();
    const { result, rerender } = renderHook(
      ({ isActive }) => useAnimation(isActive, 'fadeIn', 'fadeOut', 300),
      { initialProps: { isActive: true } }
    );

    expect(result.current.shouldRender).toBe(true);

    rerender({ isActive: false });
    expect(result.current.animationClass).toBe('fadeOut');

    act(() => {
      jest.advanceTimersByTime(300);
    });

    expect(result.current.shouldRender).toBe(false);
    jest.useRealTimers();
  });
});

describe('useMountAnimation Hook', () => {
  it('returns empty string initially', () => {
    const { result } = renderHook(() => useMountAnimation('slideUp'));
    expect(result.current).toBe('');
  });

  it('returns animation class after delay', () => {
    jest.useFakeTimers();
    const { result } = renderHook(() => useMountAnimation('slideUp', 100));

    act(() => {
      jest.advanceTimersByTime(100);
    });

    expect(result.current).toBe('slideUp');
    jest.useRealTimers();
  });

  it('uses default values', () => {
    jest.useFakeTimers();
    const { result } = renderHook(() => useMountAnimation());

    act(() => {
      jest.runAllTimers();
    });

    expect(result.current).toBe('slideUp');
    jest.useRealTimers();
  });
});

describe('useStaggerAnimation Hook', () => {
  it('returns function that returns false initially for all items', () => {
    const { result } = renderHook(() => useStaggerAnimation(3));
    expect(result.current(0)).toBe(false);
    expect(result.current(1)).toBe(false);
    expect(result.current(2)).toBe(false);
  });

  it('reveals items in sequence', () => {
    jest.useFakeTimers();
    const { result } = renderHook(() => useStaggerAnimation(3, 0, 50));

    act(() => {
      jest.advanceTimersByTime(0);
    });
    expect(result.current(0)).toBe(true);

    act(() => {
      jest.advanceTimersByTime(50);
    });
    expect(result.current(1)).toBe(true);

    act(() => {
      jest.advanceTimersByTime(50);
    });
    expect(result.current(2)).toBe(true);

    jest.useRealTimers();
  });

  it('resets when itemCount changes', () => {
    jest.useFakeTimers();
    const { result, rerender } = renderHook(
      ({ count }) => useStaggerAnimation(count, 0, 50),
      { initialProps: { count: 2 } }
    );

    act(() => {
      jest.runAllTimers();
    });
    expect(result.current(0)).toBe(true);

    rerender({ count: 3 });
    expect(result.current(0)).toBe(false);
    expect(result.current(1)).toBe(false);
    expect(result.current(2)).toBe(false);

    jest.useRealTimers();
  });
});
