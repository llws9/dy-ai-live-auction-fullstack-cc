import React, { useState, useEffect } from 'react';
import { IntroAnimation } from './IntroAnimation';
import '../../styles/bid-success-animation.css';

interface BidSuccessAnimationProps {
  productName: string;
  price: number;
  imageUrl?: string;
  onAnimationEnd?: () => void;
}

export const BidSuccessAnimation: React.FC<BidSuccessAnimationProps> = ({
  productName,
  price,
  imageUrl,
  onAnimationEnd
}) => {
  const [show, setShow] = useState(true);

  // 动画总时长大概在 2.5s 左右，3 秒后可触发结束回调
  useEffect(() => {
    const timer = setTimeout(() => {
      if (onAnimationEnd) onAnimationEnd();
    }, 3000);
    return () => clearTimeout(timer);
  }, [onAnimationEnd]);

  if (!show) return null;

  return (
    <div className="shake-trigger" style={{ position: 'fixed', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9999 }}>
      <IntroAnimation />
      
      <div className="card-container">
        <div className="auction-card v1-anim">
          <div className="v1-stamp">成交</div>
          
          <div style={{
            width: 120, height: 120, borderRadius: 16, margin: '0 auto 24px',
            background: '#F3F4F6', display: 'flex', alignItems: 'center', justifyContent: 'center',
            overflow: 'hidden'
          }}>
            {imageUrl ? (
              <img src={imageUrl} alt={productName} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
            ) : (
              <span style={{ color: '#6B7280', fontSize: 14 }}>[ 拍品图 ]</span>
            )}
          </div>
          
          <div style={{ fontSize: 18, fontWeight: 600, marginBottom: 4, color: '#111827' }}>
            {productName}
          </div>
          <div style={{ fontSize: 14, color: '#6B7280' }}>
            最终成交价
          </div>
          <div style={{ fontSize: 32, fontWeight: 800, color: '#F59E0B', marginBottom: 8 }}>
            ¥ {price.toLocaleString()}
          </div>
        </div>
      </div>
    </div>
  );
};
