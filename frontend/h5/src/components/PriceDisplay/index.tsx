// components/PriceDisplay/index.tsx

import React, { useState, useEffect, useRef } from 'react';

interface PriceDisplayProps {
  currentPrice: number;
  endTime: string;
}

const PriceDisplay: React.FC<PriceDisplayProps> = ({ currentPrice, endTime }) => {
  const [countdown, setCountdown] = useState<number>(0);
  const frameRef = useRef<number>();

  useEffect(() => {
    const endTimestamp = new Date(endTime).getTime();

    const updateCountdown = () => {
      const now = Date.now();
      const remaining = Math.max(0, endTimestamp - now);
      setCountdown(remaining);

      if (remaining > 0) {
        frameRef.current = requestAnimationFrame(updateCountdown);
      }
    };

    frameRef.current = requestAnimationFrame(updateCountdown);

    return () => {
      if (frameRef.current) {
        cancelAnimationFrame(frameRef.current);
      }
    };
  }, [endTime]);

  // 格式化倒计时（毫秒级精度）
  const formatCountdown = (ms: number): string => {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    const milliseconds = Math.floor((ms % 1000) / 10);

    return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}.${String(milliseconds).padStart(2, '0')}`;
  };

  // 判断是否在延时窗口内
  const isInDelayWindow = countdown > 0 && countdown <= 30000;

  return (
    <div style={{
      padding: '20px',
      backgroundColor: '#fff',
      borderRadius: '8px',
      marginBottom: '20px',
      boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
    }}>
      {/* 当前价格 */}
      <div style={{ textAlign: 'center', marginBottom: '15px' }}>
        <div style={{ fontSize: '14px', color: '#666', marginBottom: '5px' }}>
          当前价格
        </div>
        <div style={{
          fontSize: '36px',
          fontWeight: 'bold',
          color: '#ff4d4f',
        }}>
          ¥{currentPrice.toFixed(2)}
        </div>
      </div>

      {/* 倒计时 */}
      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: '14px', color: '#666', marginBottom: '5px' }}>
          剩余时间
        </div>
        <div style={{
          fontSize: '28px',
          fontWeight: 'bold',
          color: countdown <= 30000 ? '#ff4d4f' : '#1890ff',
          fontFamily: 'monospace',
        }}>
          {formatCountdown(countdown)}
        </div>
        {isInDelayWindow && (
          <div style={{
            marginTop: '5px',
            fontSize: '12px',
            color: '#ff4d4f',
          }}>
            ⚠️ 即将结束，出价将触发延时
          </div>
        )}
      </div>

      {/* 进度条 */}
      {countdown > 0 && (
        <div style={{
          marginTop: '15px',
          height: '4px',
          backgroundColor: '#f0f0f0',
          borderRadius: '2px',
          overflow: 'hidden',
        }}>
          <div style={{
            height: '100%',
            width: `${Math.min(100, (countdown / 300000) * 100)}%`,
            backgroundColor: countdown <= 30000 ? '#ff4d4f' : '#1890ff',
            transition: 'width 0.1s linear',
          }} />
        </div>
      )}
    </div>
  );
};

export default PriceDisplay;
