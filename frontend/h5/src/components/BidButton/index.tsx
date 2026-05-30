// components/BidButton/index.tsx

import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react';
import { useAuth } from '../../store/authContext';
import { useSkyLamp } from '../../hooks/useSkyLamp';
import { bidApi } from '../../services/api';

interface BidButtonProps {
  auctionId: number;
  currentPrice: number;
  increment: number;
  onBidSuccess?: (newPrice: number) => void;
}

const BidButton: React.FC<BidButtonProps> = ({
  auctionId,
  currentPrice,
  increment,
  onBidSuccess,
}) => {
  const { token, isAuthenticated } = useAuth();
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null);
  const [customAmount, setCustomAmount] = useState<string>('');

  const {
    loading: skyLampLoading,
    active: skyLampActive,
    refreshStatus,
    start: startSkyLamp,
    stop: stopSkyLamp,
  } = useSkyLamp(token, auctionId);

  useEffect(() => {
    if (isAuthenticated) {
      refreshStatus();
    }
  }, [isAuthenticated, refreshStatus]);

  // placeBid 通过 ref 暴露最新引用，使 debounce 实例可保持稳定
  const placeBidRef = useRef<(amount: number) => Promise<void>>(async () => {});

  placeBidRef.current = async (amount: number) => {
    if (!isAuthenticated) {
      setMessage({ text: '请先登录', type: 'error' });
      return;
    }

    setLoading(true);
    setMessage(null);

    try {
      // 使用统一的 bidApi（已封装鉴权 / 业务码 / 错误提示）
      await bidApi.placeBid(auctionId, amount);
      setMessage({ text: '🎉 出价成功！', type: 'success' });
      setCustomAmount('');
      onBidSuccess?.(amount);
    } catch (error: any) {
      // api.ts 已经统一弹出 toast，这里只做 UI 文案
      setMessage({ text: error?.message || '出价失败', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  // 防抖出价：实例只创建一次，依赖通过 ref 始终读取最新 props
  const debouncedPlaceBid = useMemo(() => {
    let timer: number | null = null;
    return (amount: number) => {
      if (timer) window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        timer = null;
        void placeBidRef.current(amount);
      }, 500);
    };
  }, []);

  const handleStartSkyLamp = useCallback(async () => {
    try {
      await startSkyLamp();
      setMessage({ text: '✨ 点天灯已开启', type: 'success' });
    } catch (error: any) {
      setMessage({ text: error?.message || '开启点天灯失败', type: 'error' });
    }
  }, [startSkyLamp]);

  const handleStopSkyLamp = useCallback(async () => {
    try {
      await stopSkyLamp();
      setMessage({ text: '🛑 点天灯已停止', type: 'success' });
    } catch (error: any) {
      setMessage({ text: error?.message || '停止点天灯失败', type: 'error' });
    }
  }, [stopSkyLamp]);

  const handleBid = (multiplier: number = 1) => {
    const bidAmount = currentPrice + increment * multiplier;
    debouncedPlaceBid(bidAmount);
  };

  const handleCustomBid = () => {
    const value = parseFloat(customAmount);
    const minBid = currentPrice + increment;
    if (value >= minBid) {
      debouncedPlaceBid(value);
    } else {
      setMessage({ text: `最低出价 ¥${minBid}`, type: 'error' });
    }
  };

  const minBid = currentPrice + increment;
  const doubleBid = currentPrice + increment * 2;

  return (
    <div style={styles.container}>
      {/* 消息提示 */}
      {message && (
        <div style={{
          ...styles.message,
          ...(message.type === 'success' ? styles.messageSuccess : styles.messageError),
        }}>
          {message.text}
        </div>
      )}

      {/* 快捷出价按钮 */}
      <div style={styles.quickBidSection}>
        <button
          onClick={() => handleBid(1)}
          disabled={loading || skyLampLoading}
          style={{
            ...styles.bidButton,
            ...styles.bidButtonPrimary,
            opacity: loading || skyLampLoading ? 0.6 : 1,
          }}
        >
          <span style={styles.buttonIcon}>💰</span>
          <span style={styles.buttonText}>
            出价 <span style={styles.buttonAmount}>¥{minBid}</span>
          </span>
          <span style={styles.buttonHint}>+{increment}元</span>
        </button>

        <button
          onClick={() => handleBid(2)}
          disabled={loading || skyLampLoading}
          style={{
            ...styles.bidButton,
            ...styles.bidButtonHot,
            opacity: loading || skyLampLoading ? 0.6 : 1,
          }}
        >
          <span style={styles.buttonIcon}>🔥</span>
          <span style={styles.buttonText}>
            加倍 <span style={styles.buttonAmount}>¥{doubleBid}</span>
          </span>
          <span style={styles.buttonHint}>+{increment * 2}元</span>
        </button>
      </div>

      {/* 自定义出价 */}
      <div style={styles.customBidSection}>
        <div style={styles.inputWrapper}>
          <span style={styles.currencySymbol}>¥</span>
          <input
            type="number"
            value={customAmount}
            onChange={(e) => setCustomAmount(e.target.value)}
            placeholder={`最低 ${minBid}`}
            style={styles.input}
            min={minBid}
          />
        </div>
        <button
          onClick={handleCustomBid}
          disabled={loading || skyLampLoading}
          style={{
            ...styles.confirmButton,
            opacity: loading || skyLampLoading ? 0.6 : 1,
          }}
        >
          确认出价
        </button>
      </div>

      {/* 点天灯 */}
      <div style={styles.skyLampSection}>
        {skyLampActive ? (
          <button
            onClick={handleStopSkyLamp}
            disabled={loading || skyLampLoading}
            style={{
              ...styles.skyLampButton,
              ...styles.skyLampStopButton,
              opacity: loading || skyLampLoading ? 0.6 : 1,
            }}
          >
            🛑 停止点天灯
          </button>
        ) : (
          <button
            onClick={handleStartSkyLamp}
            disabled={loading || skyLampLoading}
            style={{
              ...styles.skyLampButton,
              ...styles.skyLampStartButton,
              opacity: loading || skyLampLoading ? 0.6 : 1,
            }}
          >
            ✨ 开启点天灯
          </button>
        )}
        <p style={styles.skyLampHint}>
          {skyLampActive ? '点天灯进行中：系统将自动跟价' : '开启后将按规则自动跟价'}
        </p>
      </div>

      {/* 出价提示 */}
      <p style={styles.hint}>
        💡 每次出价需增加 ¥{increment} 或以上
      </p>
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    padding: '16px 0',
  },
  message: {
    padding: '12px 16px',
    marginBottom: '16px',
    borderRadius: '8px',
    textAlign: 'center',
    fontSize: '14px',
    animation: 'fadeIn 0.3s ease',
  },
  messageSuccess: {
    backgroundColor: '#f6ffed',
    color: '#52c41a',
    border: '1px solid #b7eb8f',
  },
  messageError: {
    backgroundColor: '#fff2f0',
    color: '#ff4d4f',
    border: '1px solid #ffccc7',
  },
  quickBidSection: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '12px',
    marginBottom: '16px',
  },
  bidButton: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: '16px 12px',
    border: 'none',
    borderRadius: '12px',
    cursor: 'pointer',
    transition: 'transform 0.2s, box-shadow 0.2s',
  },
  bidButtonPrimary: {
    background: 'linear-gradient(135deg, #1890ff 0%, #096dd9 100%)',
    color: 'white',
  },
  bidButtonHot: {
    background: 'linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%)',
    color: 'white',
  },
  buttonIcon: {
    fontSize: '20px',
    marginBottom: '4px',
  },
  buttonText: {
    fontSize: '14px',
    marginBottom: '2px',
  },
  buttonAmount: {
    fontSize: '18px',
    fontWeight: 'bold',
  },
  buttonHint: {
    fontSize: '11px',
    opacity: 0.8,
  },
  customBidSection: {
    display: 'flex',
    gap: '12px',
    marginBottom: '12px',
  },
  inputWrapper: {
    flex: 1,
    display: 'flex',
    alignItems: 'center',
    backgroundColor: '#f5f5f5',
    borderRadius: '8px',
    padding: '0 12px',
  },
  currencySymbol: {
    color: '#999',
    fontSize: '16px',
    marginRight: '4px',
  },
  input: {
    flex: 1,
    padding: '12px 0',
    border: 'none',
    backgroundColor: 'transparent',
    fontSize: '16px',
    outline: 'none',
  },
  confirmButton: {
    padding: '12px 24px',
    backgroundColor: '#52c41a',
    color: 'white',
    border: 'none',
    borderRadius: '8px',
    fontSize: '14px',
    fontWeight: 'bold',
    cursor: 'pointer',
  },
  skyLampSection: {
    marginBottom: '12px',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  skyLampButton: {
    width: '100%',
    padding: '12px 16px',
    border: 'none',
    borderRadius: '10px',
    fontSize: '14px',
    fontWeight: 'bold',
    cursor: 'pointer',
  },
  skyLampStartButton: {
    background: 'linear-gradient(135deg, #722ed1 0%, #531dab 100%)',
    color: '#fff',
  },
  skyLampStopButton: {
    background: '#fff1f0',
    color: '#cf1322',
    border: '1px solid #ffa39e',
  },
  skyLampHint: {
    margin: 0,
    color: '#722ed1',
    fontSize: '12px',
    textAlign: 'center',
  },
  hint: {
    textAlign: 'center',
    color: '#999',
    fontSize: '12px',
    margin: 0,
  },
};

export default BidButton;
