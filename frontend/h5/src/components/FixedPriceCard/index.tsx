import type { FixedPriceItem, ProductBrief } from '../../api/fixedPrice';
import styles from './index.module.css';

interface FixedPriceCardProps {
  item: FixedPriceItem;
  purchased?: boolean;
  onPurchase: (itemId: number) => void;
}

function getProductBrief(item: FixedPriceItem): ProductBrief {
  return item.product_brief ?? item.product ?? {
    id: item.product_id ?? item.id,
    title: item.product_title ?? '一口价商品',
  };
}

function getButtonState(item: FixedPriceItem, purchased: boolean): { disabled: boolean; label: string } {
  if (purchased) {
    return { disabled: true, label: '已购买' };
  }

  if (item.status === 'offline') {
    return { disabled: true, label: '已下架' };
  }

  if (item.status === 'sold_out' || item.remaining_stock <= 0) {
    return { disabled: true, label: '已售罄' };
  }

  return { disabled: false, label: '立即抢' };
}

export default function FixedPriceCard({ item, purchased = false, onPurchase }: FixedPriceCardProps) {
  const product = getProductBrief(item);
  const button = getButtonState(item, purchased);
  const stockText = `剩 ${item.remaining_stock} / ${item.total_stock}`;

  return (
    <article className={styles.card} aria-label={`${product.title} 一口价商品`}>
      <div className={styles.badge}>限时一口价</div>

      {product.cover_image ? (
        <img className={styles.cover} src={product.cover_image} alt={product.title} />
      ) : (
        <div className={styles.coverFallback} role="img" aria-label={product.title}>
          无图
        </div>
      )}

      <div className={styles.info}>
        <h3 className={styles.title}>{product.title}</h3>
        <div className={styles.meta}>
          <span className={styles.price}>¥{item.price}</span>
          <span className={styles.stock}>{stockText}</span>
        </div>
      </div>

      <button
        className={styles.purchaseButton}
        disabled={button.disabled}
        type="button"
        onClick={() => onPurchase(item.id)}
      >
        {button.label}
      </button>
    </article>
  );
}
