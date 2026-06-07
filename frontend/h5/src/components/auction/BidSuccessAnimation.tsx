import React, { useEffect, useRef } from 'react';
import { IntroAnimation } from './IntroAnimation';
import './bid-success-animation.css';

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
  onAnimationEnd,
}) => {
  const onAnimationEndRef = useRef(onAnimationEnd);

  useEffect(() => {
    onAnimationEndRef.current = onAnimationEnd;
  }, [onAnimationEnd]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      onAnimationEndRef.current?.();
    }, 3000);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <div className="shake-trigger" data-testid="bid-success-animation">
      <IntroAnimation />

      <div className="card-container">
        <div className="auction-card v1-anim">
          <div className="v1-stamp">成交</div>

          <div className="auction-card-image">
            {imageUrl ? (
              <img src={imageUrl} alt={productName} />
            ) : (
              <span>[ 拍品图 ]</span>
            )}
          </div>

          <div className="auction-card-title">{productName}</div>
          <div className="auction-card-subtitle">最终成交价</div>
          <div className="auction-card-price">¥{price.toLocaleString('zh-CN')}</div>
        </div>
      </div>
    </div>
  );
};
