import React, { useEffect, useState } from 'react';
import styles from './BidFlairOverlay.module.css';

export interface BidEvent {
  id: string;
  userId: string;
  avatar: string;
  price: string;
  combo: number;
  isSelf: boolean;
}

export const BidFlairOverlay: React.FC<{ latestBid?: BidEvent | null }> = ({ latestBid }) => {
  const [flairs, setFlairs] = useState<BidEvent[]>([]);

  useEffect(() => {
    if (latestBid) {
      setFlairs(prev => [...prev.slice(-4), latestBid]);
      const timer = setTimeout(() => {
        setFlairs(prev => prev.filter(f => f.id !== latestBid.id));
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [latestBid]);

  return (
    <div className={styles.flairContainer} data-testid="bid-flair-overlay">
      {flairs.map(f => (
        <div
          key={f.id}
          className={`${styles.flairItem} ${f.isSelf ? styles.isSelf : ''}`}
          data-testid={f.isSelf ? 'bid-success-flair' : `bid-flair-item-${f.id}`}
        >
          {f.avatar ? (
            <img src={f.avatar} className={styles.avatar} alt="" />
          ) : (
            <div className={styles.avatar} style={{ background: '#ccc', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#333' }}>
              {f.userId.slice(0, 1)}
            </div>
          )}
          {f.combo > 1 && <span className={styles.comboText}>x{f.combo} COMBO</span>}
          <span>{f.userId} 刚刚出价 <span className={styles.priceText}>¥{f.price}</span></span>
        </div>
      ))}
    </div>
  );
};
