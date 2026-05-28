// hooks/useCountdown.ts

import { useState, useEffect, useRef, useCallback } from 'react';

interface UseCountdownOptions {
  endTime: number; // 毫秒时间戳
  onEnd?: () => void;
  syncInterval?: number; // 时间同步间隔（毫秒）
}

export function useCountdown(options: UseCountdownOptions) {
  const { endTime, onEnd } = options;
  const [countdown, setCountdown] = useState<number>(0);
  const [isEnded, setIsEnded] = useState(false);
  const frameRef = useRef<number>();

  const updateCountdown = useCallback(() => {
    const now = Date.now();
    const remaining = Math.max(0, endTime - now);

    setCountdown(remaining);

    if (remaining === 0 && !isEnded) {
      setIsEnded(true);
      if (onEnd) {
        onEnd();
      }
      return;
    }

    if (remaining > 0) {
      frameRef.current = requestAnimationFrame(updateCountdown);
    }
  }, [endTime, isEnded, onEnd]);

  useEffect(() => {
    frameRef.current = requestAnimationFrame(updateCountdown);

    return () => {
      if (frameRef.current) {
        cancelAnimationFrame(frameRef.current);
      }
    };
  }, [updateCountdown]);

  // 格式化函数
  const formatTime = useCallback((ms: number): string => {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    const milliseconds = Math.floor((ms % 1000) / 10);

    return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}.${String(milliseconds).padStart(2, '0')}`;
  }, []);

  // 判断是否在延时窗口内
  const isInDelayWindow = countdown > 0 && countdown <= 30000;

  // 判断是否即将结束
  const isEnding = countdown > 0 && countdown <= 60000;

  return {
    countdown,
    isEnded,
    isInDelayWindow,
    isEnding,
    formatTime,
    minutes: Math.floor(countdown / 60000),
    seconds: Math.floor((countdown % 60000) / 1000),
    milliseconds: Math.floor((countdown % 1000) / 10),
  };
}
