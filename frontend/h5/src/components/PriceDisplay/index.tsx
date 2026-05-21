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
  const formatCountdown = (ms: number): { minutes: string; seconds: string; ms: string } => {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    const milliseconds = Math.floor((ms % 1000) / 10);

    return {
      minutes: String(minutes).padStart(2, '0'),
      seconds: String(seconds).padStart(2, '0'),
      ms: String(milliseconds).padStart(2, '0'),
    };
  };

  // 判断是否在延时窗口内
  const isInDelayWindow = countdown > 0 && countdown <= 30000;
  const isEnded = countdown === 0;
  const time = formatCountdown(countdown);

  return (
    <div style={styles.container}>
      {/* 价格区域 */}
      <div style={styles.priceSection}>
        <div style={styles.priceLabel}>
          <span style={styles.priceIcon}>💎</span>
          当前价格
        </div>
        <div style={styles.priceValue}>
          <span style={styles.currency}>¥</span>
          <span style={styles.amount}>{currentPrice.toLocaleString()}</span>
        </div>
      </div>

      {/* 倒计时区域 */}
      <div style={{
        ...styles.countdownSection,
        ...(isInDelayWindow ? styles.countdownWarning : {}),
        ...(isEnded ? styles.countdownEnded : {}),
      }}>
        <div style={styles.countdownLabel}>
          {isEnded ? '竞拍已结束' : isInDelayWindow ? '⚠️ 延时窗口' : '⏱️ 剩余时间'}
        </div>

        {!isEnded && (
          <div style={styles.countdownValue}>
            <div style={styles.timeBlock}>
              <span style={styles.timeNumber}>{time.minutes}</span>
              <span style={styles.timeUnit}>分</span>
            </div>
            <span style={styles.timeSeparator}>:</span>
            <div style={styles.timeBlock}>
              <span style={styles.timeNumber}>{time.seconds}</span>
              <span style={styles.timeUnit}>秒</span>
            </div>
            <span style={styles.timeSeparator}>.</span>
            <div style={styles.timeBlockSmall}>
              <span style={styles.timeNumberSmall}>{time.ms}</span>
            </div>
          </div>
        )}

        {isInDelayWindow && (
          <div style={styles.delayHint}>
            即将结束，出价将触发延时！
          </div>
        )}
      </div>

      {/* 进度条 */}
      {countdown > 0 && (
        <div style={styles.progressBar}>
          <div style={{
            ...styles.progressFill,
            width: `${Math.min(100, (countdown / 300000) * 100)}%`,
            backgroundColor: isInDelayWindow ? '#ff4d4f' : '#1890ff',
          }} />
        </div>
      )}
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    backgroundColor: 'white',
    borderRadius: '16px',
    overflow: 'hidden',
    boxShadow: '0 4px 12px rgba(0,0,0,0.08)',
  },
  priceSection: {
    padding: '20px',
    textAlign: 'center',
    background: 'linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%)',
    color: 'white',
  },
  priceLabel: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '6px',
    fontSize: '14px',
    opacity: 0.9,
    marginBottom: '8px',
  },
  priceIcon: {
    fontSize: '16px',
  },
  priceValue: {
    display: 'flex',
    alignItems: 'baseline',
    justifyContent: 'center',
  },
  currency: {
    fontSize: '24px',
    marginRight: '4px',
  },
  amount: {
    fontSize: '48px',
    fontWeight: 'bold',
    letterSpacing: '-1px',
  },
  countdownSection: {
    padding: '20px',
    textAlign: 'center',
    backgroundColor: '#fafafa',
    transition: 'background-color 0.3s',
  },
  countdownWarning: {
    backgroundColor: '#fff1f0',
  },
  countdownEnded: {
    backgroundColor: '#f5f5f5',
  },
  countdownLabel: {
    fontSize: '14px',
    color: '#666',
    marginBottom: '12px',
  },
  countdownValue: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '4px',
  },
  timeBlock: {
    display: 'flex',
    alignItems: 'baseline',
    gap: '2px',
  },
  timeNumber: {
    fontSize: '32px',
    fontWeight: 'bold',
    fontFamily: 'monospace',
    color: '#333',
    minWidth: '48px',
    textAlign: 'center',
  },
  timeUnit: {
    fontSize: '12px',
    color: '#999',
  },
  timeSeparator: {
    fontSize: '24px',
    color: '#999',
    margin: '0 2px',
  },
  timeBlockSmall: {
    display: 'flex',
    alignItems: 'baseline',
  },
  timeNumberSmall: {
    fontSize: '20px',
    fontWeight: 'bold',
    fontFamily: 'monospace',
    color: '#666',
    minWidth: '32px',
    textAlign: 'center',
  },
  delayHint: {
    marginTop: '12px',
    padding: '8px 16px',
    backgroundColor: '#ff4d4f',
    color: 'white',
    borderRadius: '20px',
    fontSize: '12px',
    display: 'inline-block',
    animation: 'pulse 1.5s infinite',
  },
  progressBar: {
    height: '4px',
    backgroundColor: '#f0f0f0',
  },
  progressFill: {
    height: '100%',
    transition: 'width 0.1s linear',
  },
};

export default PriceDisplay;
