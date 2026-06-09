import type { FixedPriceItem } from '@/api/fixedPrice';
import { type CSSProperties, useLayoutEffect, useRef, useState } from 'react';
import styles from './FixedPriceIntroAnimation.module.css';

interface FixedPriceIntroAnimationProps {
  item: FixedPriceItem;
  onComplete: (itemId: number) => void;
}

export function FixedPriceIntroAnimation({ item, onComplete }: FixedPriceIntroAnimationProps) {
  const product = item.product_brief ?? item.product ?? { title: item.product_title ?? '一口价商品', cover_image: '' };
  const containerRef = useRef<HTMLDivElement>(null);
  const [flyOffset, setFlyOffset] = useState({ x: 0, y: 0 });

  useLayoutEffect(() => {
    const measureFlyTarget = () => {
      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect) return;

      // Mirror the previous lower-right target, but relative to the live-room
      // container instead of the browser viewport / desktop phone-shell page.
      setFlyOffset({
        x: Math.max(0, rect.width / 2 - 80),
        y: Math.max(0, rect.height * 0.6 - 120),
      });
    };

    measureFlyTarget();
    window.addEventListener('resize', measureFlyTarget);
    return () => window.removeEventListener('resize', measureFlyTarget);
  }, []);

  return (
    <div ref={containerRef} className={styles.container} data-testid="fixed-price-intro-container">
      <div 
        className={styles.card}
        data-testid="fixed-price-intro-card"
        style={{
          '--fly-to-x': `${flyOffset.x}px`,
          '--fly-to-y': `${flyOffset.y}px`,
        } as CSSProperties}
        onAnimationEnd={(e) => {
          if (e.animationName.includes('flyToBottomRight')) {
            onComplete(item.id);
          }
        }}
      >
        <div className={styles.badge}>新上架 一口价</div>
        {product.cover_image ? (
          <img className={styles.cover} src={product.cover_image} alt={product.title} />
        ) : (
          <div className={styles.coverFallback}>无图</div>
        )}
        <div className={styles.info}>
          <h3 className={styles.title}>{product.title}</h3>
          <span className={styles.price}>¥{item.price}</span>
        </div>
      </div>
    </div>
  );
}
