// hooks/useServerTime.ts

import { useState, useEffect, useRef, useCallback } from 'react';

interface ServerTimeOptions {
  syncInterval?: number; // 同步间隔（毫秒），默认10秒
  onTimeSync?: (serverTime: number) => void;
}

export const useServerTime = (
  serverEndTime: number,
  options: ServerTimeOptions = {}
) => {
  const { syncInterval = 10000, onTimeSync } = options;

  const [countdown, setCountdown] = useState(0);
  const [serverTimeOffset, setServerTimeOffset] = useState(0);
  const frameIdRef = useRef<number>(0);
  const lastSyncTimeRef = useRef<number>(0);

  // 计算倒计时
  const updateCountdown = useCallback(() => {
    const now = Date.now() + serverTimeOffset;
    const remaining = Math.max(0, serverEndTime - now);
    setCountdown(remaining);

    if (remaining > 0) {
      frameIdRef.current = requestAnimationFrame(updateCountdown);
    }
  }, [serverEndTime, serverTimeOffset]);

  // 启动倒计时
  useEffect(() => {
    frameIdRef.current = requestAnimationFrame(updateCountdown);

    return () => {
      if (frameIdRef.current) {
        cancelAnimationFrame(frameIdRef.current);
      }
    };
  }, [updateCountdown]);

  // 同步服务器时间
  const syncServerTime = useCallback((serverTime: number) => {
    const localTime = Date.now();
    const offset = serverTime - localTime;
    setServerTimeOffset(offset);
    lastSyncTimeRef.current = Date.now();

    if (onTimeSync) {
      onTimeSync(serverTime);
    }
  }, [onTimeSync]);

  // 定期同步
  useEffect(() => {
    const interval = setInterval(() => {
      // 触发同步请求（通过WebSocket或其他方式）
      // 这里只是占位，实际需要通过WebSocket发送sync_request
    }, syncInterval);

    return () => clearInterval(interval);
  }, [syncInterval]);

  // 格式化倒计时
  const formatCountdown = useCallback((ms: number): string => {
    if (ms <= 0) return '00:00:00.000';

    const hours = Math.floor(ms / 3600000);
    const minutes = Math.floor((ms % 3600000) / 60000);
    const seconds = Math.floor((ms % 60000) / 1000);
    const milliseconds = ms % 1000;

    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}.${milliseconds.toString().padStart(3, '0')}`;
  }, []);

  return {
    countdown,
    serverTimeOffset,
    syncServerTime,
    formatCountdown,
    formattedCountdown: formatCountdown(countdown),
  };
};

export default useServerTime;
