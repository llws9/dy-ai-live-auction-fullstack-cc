import { useCallback, useEffect, useRef, useState } from 'react';

export type BidHeatLevel = 'calm' | 'warming' | 'blazing';

const BID_HEAT_WINDOW_MS = 10_000;
const BID_HEAT_TICK_MS = 1_000;

const levelFromBidCount = (count: number): BidHeatLevel => {
  if (count >= 5) return 'blazing';
  if (count >= 2) return 'warming';
  return 'calm';
};

const pruneExpiredBids = (timestamps: number[], now: number) =>
  timestamps.filter((timestamp) => now - timestamp < BID_HEAT_WINDOW_MS);

export const useBidHeat = () => {
  const timestampsRef = useRef<number[]>([]);
  const [level, setLevel] = useState<BidHeatLevel>('calm');

  const syncLevel = useCallback((now = Date.now()) => {
    const activeTimestamps = pruneExpiredBids(timestampsRef.current, now);
    timestampsRef.current = activeTimestamps;
    setLevel(levelFromBidCount(activeTimestamps.length));
  }, []);

  const markBid = useCallback(() => {
    const now = Date.now();
    const activeTimestamps = [...pruneExpiredBids(timestampsRef.current, now), now];
    timestampsRef.current = activeTimestamps;
    setLevel(levelFromBidCount(activeTimestamps.length));
  }, []);

  const reset = useCallback(() => {
    timestampsRef.current = [];
    setLevel('calm');
  }, []);

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      syncLevel();
    }, BID_HEAT_TICK_MS);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [syncLevel]);

  return {
    level,
    markBid,
    reset,
  };
};
