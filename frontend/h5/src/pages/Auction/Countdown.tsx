// frontend/h5/src/pages/Auction/Countdown.tsx

import React from 'react';
import { useCountdown } from '../../hooks/useCountdown';

interface CountdownProps {
  endTime: number; // 毫秒时间戳
  onEnd?: () => void;
}

const Countdown: React.FC<CountdownProps> = ({ endTime, onEnd }) => {
  const {
    countdown,
    isInDelayWindow,
    isEnding,
    formatTime,
  } = useCountdown({ endTime, onEnd });

  return (
    <div style={{
      textAlign: 'center',
      padding: '20px',
      backgroundColor: '#fff',
      borderRadius: '8px',
    }}>
      <div style={{ fontSize: '14px', color: '#666', marginBottom: '10px' }}>
        剩余时间
      </div>

      <div style={{
        fontSize: '36px',
        fontWeight: 'bold',
        color: isInDelayWindow ? '#ff4d4f' : isEnding ? '#faad14' : '#1890ff',
        fontFamily: 'monospace',
        letterSpacing: '2px',
      }}>
        {formatTime(countdown)}
      </div>

      {isInDelayWindow && (
        <div style={{
          marginTop: '10px',
          padding: '8px',
          backgroundColor: '#fff2f0',
          borderRadius: '4px',
          color: '#ff4d4f',
          fontSize: '12px',
        }}>
          ⚠️ 即将结束，出价将触发延时
        </div>
      )}

      {/* 进度条 */}
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
          backgroundColor: isInDelayWindow ? '#ff4d4f' : isEnding ? '#faad14' : '#1890ff',
          transition: 'width 0.1s linear',
        }} />
      </div>
    </div>
  );
};

export default Countdown;
