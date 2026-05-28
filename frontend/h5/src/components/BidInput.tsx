import React, { useState, useEffect } from 'react';
import { useAuth } from '../store/authContext';
import { bidApi } from '../services/api';

interface BidInputProps {
  auctionId: number;
  currentPrice: number;
  minIncrement: number;
  maxPrice?: number;
  onBidSuccess?: (result: any) => void;
  onBidError?: (error: Error) => void;
}

const BidInput: React.FC<BidInputProps> = ({
  auctionId,
  currentPrice,
  minIncrement,
  maxPrice,
  onBidSuccess,
  onBidError,
}) => {
  const { isAuthenticated } = useAuth();
  const [amount, setAmount] = useState<string>('');
  const [error, setError] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [minBid, setMinBid] = useState(0);

  useEffect(() => {
    // 计算最小出价金额
    const min = currentPrice + minIncrement;
    setMinBid(min);
    setAmount(min.toFixed(2));
  }, [currentPrice, minIncrement]);

  const validateBid = (value: number): string | null => {
    if (isNaN(value)) {
      return '请输入有效的出价金额';
    }

    if (value < minBid) {
      return `出价金额不能低于${minBid.toFixed(2)}元`;
    }

    if (maxPrice && value > maxPrice) {
      return `出价金额不能超过${maxPrice.toFixed(2)}元`;
    }

    // 验证小数位数
    const decimalPart = value.toString().split('.')[1];
    if (decimalPart && decimalPart.length > 2) {
      return '出价金额只能精确到小数点后2位';
    }

    return null;
  };

  const handleAmountChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setAmount(value);

    // 实时验证
    const numValue = parseFloat(value);
    if (value && !isNaN(numValue)) {
      const validationError = validateBid(numValue);
      setError(validationError || '');
    } else {
      setError('');
    }
  };

  const handleBid = async () => {
    // 检查登录状态
    if (!isAuthenticated) {
      alert('请先登录');
      // 跳转到登录页
      window.location.href = '/login';
      return;
    }

    const numAmount = parseFloat(amount);
    const validationError = validateBid(numAmount);

    if (validationError) {
      setError(validationError);
      return;
    }

    setLoading(true);
    setError('');

    try {
      const result = await bidApi.placeBid(auctionId, numAmount);

      // 出价成功
      alert('出价成功！');
      if (onBidSuccess) {
        onBidSuccess(result);
      }

      // 更新最小出价金额
      const newMinBid = numAmount + minIncrement;
      setMinBid(newMinBid);
      setAmount(newMinBid.toFixed(2));
    } catch (err: any) {
      const errorMsg = err.message || '出价失败，请重试';
      setError(errorMsg);
      if (onBidError) {
        onBidError(err);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleQuickBid = (increment: number) => {
    const quickAmount = minBid + increment;
    setAmount(quickAmount.toFixed(2));
    setError('');
  };

  return (
    <div style={{ padding: '16px', backgroundColor: '#fff', borderRadius: '8px' }}>
      <div style={{ marginBottom: '12px' }}>
        <div style={{ fontSize: '14px', color: '#666', marginBottom: '4px' }}>
          当前价格
        </div>
        <div style={{ fontSize: '24px', fontWeight: 'bold', color: '#ff4d4f' }}>
          ¥{currentPrice.toFixed(2)}
        </div>
      </div>

      <div style={{ marginBottom: '12px' }}>
        <div style={{ fontSize: '14px', color: '#666', marginBottom: '8px' }}>
          出价金额（最小出价：¥{minBid.toFixed(2)}）
        </div>
        <input
          type="number"
          value={amount}
          onChange={handleAmountChange}
          placeholder={`最低出价 ¥${minBid.toFixed(2)}`}
          style={{
            width: '100%',
            padding: '12px',
            fontSize: '16px',
            border: error ? '1px solid #ff4d4f' : '1px solid #ddd',
            borderRadius: '4px',
            outline: 'none',
          }}
          step="0.01"
          min={minBid}
          max={maxPrice}
          disabled={loading}
        />
        {error && (
          <div style={{ fontSize: '12px', color: '#ff4d4f', marginTop: '4px' }}>
            {error}
          </div>
        )}
      </div>

      <div style={{ marginBottom: '16px', display: 'flex', gap: '8px' }}>
        <button
          onClick={() => handleQuickBid(0)}
          style={{
            flex: 1,
            padding: '8px',
            fontSize: '14px',
            backgroundColor: '#f5f5f5',
            border: '1px solid #ddd',
            borderRadius: '4px',
            cursor: 'pointer',
          }}
          disabled={loading}
        >
          最低价
        </button>
        <button
          onClick={() => handleQuickBid(minIncrement)}
          style={{
            flex: 1,
            padding: '8px',
            fontSize: '14px',
            backgroundColor: '#f5f5f5',
            border: '1px solid #ddd',
            borderRadius: '4px',
            cursor: 'pointer',
          }}
          disabled={loading}
        >
          +¥{minIncrement}
        </button>
        <button
          onClick={() => handleQuickBid(minIncrement * 5)}
          style={{
            flex: 1,
            padding: '8px',
            fontSize: '14px',
            backgroundColor: '#f5f5f5',
            border: '1px solid #ddd',
            borderRadius: '4px',
            cursor: 'pointer',
          }}
          disabled={loading}
        >
          +¥{minIncrement * 5}
        </button>
      </div>

      <button
        onClick={handleBid}
        disabled={loading || !!error}
        style={{
          width: '100%',
          padding: '12px',
          fontSize: '16px',
          fontWeight: 'bold',
          color: '#fff',
          backgroundColor: loading || error ? '#ccc' : '#ff4d4f',
          border: 'none',
          borderRadius: '4px',
          cursor: loading || error ? 'not-allowed' : 'pointer',
        }}
      >
        {loading ? '出价中...' : '立即出价'}
      </button>

      {!isAuthenticated && (
        <div style={{ marginTop: '8px', fontSize: '12px', color: '#999', textAlign: 'center' }}>
          登录后才能出价
        </div>
      )}
    </div>
  );
};

export default BidInput;
