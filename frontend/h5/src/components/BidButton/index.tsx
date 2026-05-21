// components/BidButton/index.tsx

import React, { useState, useCallback } from 'react';

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
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

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
    setLoading(true);
    setMessage(null);

    try {
      const response = await fetch(`/api/v1/auctions/${auctionId}/bids`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': '1', // 实际应从认证状态获取
        },
        body: JSON.stringify({ amount }),
      });

      const result = await response.json();

      if (result.success) {
        setMessage('出价成功！');
        if (onBidSuccess) {
          onBidSuccess(amount);
        }
      } else {
        setMessage(result.message || '出价失败');
      }
    } catch (error) {
      setMessage('网络错误，请重试');
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

  return (
    <div style={{ padding: '20px 0' }}>
      {/* 消息提示 */}
      {message && (
        <div style={{
          padding: '10px',
          marginBottom: '15px',
          borderRadius: '4px',
          backgroundColor: message.includes('成功') ? '#f6ffed' : '#fff2f0',
          color: message.includes('成功') ? '#52c41a' : '#ff4d4f',
          textAlign: 'center',
        }}>
          {message}
        </div>
      )}

      {/* 出价按钮组 */}
      <div style={{ display: 'flex', gap: '10px', justifyContent: 'center' }}>
        {/* 基础出价 */}
        <button
          onClick={() => handleBid(1)}
          disabled={loading}
          style={{
            ...buttonStyle,
            padding: '15px 30px',
            fontSize: '16px',
            opacity: loading ? 0.5 : 1,
          }}
        >
          出价 +{increment}元
        </button>

        {/* 加倍出价 */}
        <button
          onClick={() => handleBid(2)}
          disabled={loading}
          style={{
            ...buttonStyle,
            padding: '15px 25px',
            fontSize: '14px',
            backgroundColor: '#ff4d4f',
            opacity: loading ? 0.5 : 1,
          }}
        >
          加倍 +{increment * 2}元
        </button>
      </div>

      {/* 自定义出价 */}
      <div style={{ marginTop: '15px', textAlign: 'center' }}>
        <input
          type="number"
          placeholder={`最低 ${currentPrice + increment} 元`}
          style={{
            padding: '10px',
            width: '150px',
            border: '1px solid #ddd',
            borderRadius: '4px',
            marginRight: '10px',
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              const value = parseFloat((e.target as HTMLInputElement).value);
              if (value >= currentPrice + increment) {
                debouncedPlaceBid(value);
              }
            }
          }}
        />
        <button
          style={{
            ...buttonStyle,
            padding: '10px 20px',
            backgroundColor: '#52c41a',
          }}
        >
          确认出价
        </button>
      </div>
    </div>
  );
};

const buttonStyle: React.CSSProperties = {
  backgroundColor: '#1890ff',
  color: 'white',
  border: 'none',
  borderRadius: '8px',
  cursor: 'pointer',
  fontWeight: 'bold',
};

export default BidButton;
