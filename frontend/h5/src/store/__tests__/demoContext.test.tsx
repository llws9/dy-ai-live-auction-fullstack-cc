import { ReactNode } from 'react';
import { act, renderHook } from '@testing-library/react';
import { DemoProvider, useDemo } from '../demoContext';

function wrapper({ children }: { children: ReactNode }) {
  return <DemoProvider>{children}</DemoProvider>;
}

describe('DemoProvider', () => {
  it('shares the current auction id across demo consumers', () => {
    const { result } = renderHook(() => useDemo(), { wrapper });

    expect(result.current.currentAuctionId).toBeNull();

    act(() => {
      result.current.setCurrentAuctionId(42);
    });

    expect(result.current.currentAuctionId).toBe(42);

    act(() => {
      result.current.setCurrentAuctionId(null);
    });

    expect(result.current.currentAuctionId).toBeNull();
  });

  it('fails closed when useDemo is called outside DemoProvider', () => {
    const consoleError = jest.spyOn(console, 'error').mockImplementation(() => undefined);

    try {
      expect(() => renderHook(() => useDemo())).toThrow('useDemo must be used within a DemoProvider');
    } finally {
      consoleError.mockRestore();
    }
  });
});
