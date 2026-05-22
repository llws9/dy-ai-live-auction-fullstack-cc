// hooks/useReconnect.ts

import { useState, useCallback, useRef, useEffect } from 'react';

interface ReconnectOptions {
  maxAttempts?: number;
  baseDelay?: number;
  maxDelay?: number;
  onReconnect?: () => void;
  onMaxAttemptsReached?: () => void;
}

export const useReconnect = (options: ReconnectOptions = {}) => {
  const {
    maxAttempts = 10,
    baseDelay = 1000, // 1秒
    maxDelay = 30000, // 30秒
    onReconnect,
    onMaxAttemptsReached,
  } = options;

  const [reconnectCount, setReconnectCount] = useState(0);
  const [isReconnecting, setIsReconnecting] = useState(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const attemptRef = useRef(0);

  // 指数退避策略：1s, 2s, 4s, 8s, 16s, 30s, 30s, ...
  const getDelay = useCallback((attempt: number): number => {
    const delay = baseDelay * Math.pow(2, attempt);
    return Math.min(delay, maxDelay);
  }, [baseDelay, maxDelay]);

  // 开始重连
  const startReconnect = useCallback(() => {
    if (attemptRef.current >= maxAttempts) {
      console.error('Max reconnect attempts reached');
      onMaxAttemptsReached?.();
      return false;
    }

    setIsReconnecting(true);
    const delay = getDelay(attemptRef.current);
    attemptRef.current++;
    setReconnectCount(attemptRef.current);

    console.log(`Reconnecting in ${delay}ms (attempt ${attemptRef.current}/${maxAttempts})`);

    timeoutRef.current = setTimeout(() => {
      onReconnect?.();
    }, delay);

    return true;
  }, [maxAttempts, getDelay, onReconnect, onMaxAttemptsReached]);

  // 重置重连计数
  const resetReconnect = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    attemptRef.current = 0;
    setReconnectCount(0);
    setIsReconnecting(false);
  }, []);

  // 停止重连
  const stopReconnect = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    setIsReconnecting(false);
  }, []);

  // 清理定时器
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return {
    reconnectCount,
    isReconnecting,
    startReconnect,
    resetReconnect,
    stopReconnect,
    maxAttempts,
  };
};

export default useReconnect;
