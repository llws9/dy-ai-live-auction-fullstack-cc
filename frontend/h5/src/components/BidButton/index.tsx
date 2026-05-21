// components/BidButton/index.tsx

import React, { useState, useCallback } from 'react';
import { useAuth } from '../../store/authContext';

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

  // 防抖处理
  const debounce = <T extends (...args: any[]) => any>(
    func: T,
    wait: number
  ): ((...args: Parameters<T>) => void) => {
    let timeout: NodeJS.Timeout | null = null;
    return (...args: Parameters<T>) => {
      if (timeout) clearTimeout(timeout);
      timeout = setTimeout(() => func(...args), wait);
    };
  };

  const placeBid = async (amount: number) => {
    if (!isAuthenticated) {
      setMessage({ text: '请先登录', type: 'error' });
      return;
    }

    setLoading(true);
    setMessage(null);

    try {
      const response = await fetch(`/api/v1/auctions/${auctionId}/bids`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ amount }),
      });

      const result = await response.json();

      if (result.success) {
        setMessage({ text: '🎉 出价成功！', type: 'success' });
        setCustomAmount('');
        if (onBidSuccess) {
          onBidSuccess(amount);
        }
      } else {
        setMessage({ text: result.message || '出价失败', type: 'error' });
      }
    } catch (error) {
      setMessage({ text: '网络错误，请重试', type: 'error' });
      console.error('出价失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 防抖出价函数
  const debouncedPlaceBid = useCallback(
    debounce((amount: number) => placeBid(amount), 500),
    [auctionId, currentPrice, increment]
  );

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
          disabled={loading}
          style={{
            ...styles.bidButton,
            ...styles.bidButtonPrimary,
            opacity: loading ? 0.6 : 1,
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
          disabled={loading}
          style={{
            ...styles.bidButton,
            ...styles.bidButtonHot,
            opacity: loading ? 0.6 : 1,
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
          disabled={loading}
          style={{
            ...styles.confirmButton,
            opacity: loading ? 0.6 : 1,
          }}
        >
          确认出价
        </button>
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
  hint: {
    textAlign: 'center',
    color: '#999',
    fontSize: '12px',
    margin: 0,
  },
};

export default BidButton;
