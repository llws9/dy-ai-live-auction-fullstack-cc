import type { FixedPriceItem } from '@/api/fixedPrice';
import styles from './FixedPriceIntroAnimation.module.css';

interface FixedPriceIntroAnimationProps {
  item: FixedPriceItem;
  onComplete: (itemId: number) => void;
}

export function FixedPriceIntroAnimation({ item, onComplete }: FixedPriceIntroAnimationProps) {
  const product = item.product_brief ?? item.product ?? { title: item.product_title ?? '一口价商品', cover_image: '' };

  return (
    <div className={styles.container}>
      <div 
        className={styles.card}
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
