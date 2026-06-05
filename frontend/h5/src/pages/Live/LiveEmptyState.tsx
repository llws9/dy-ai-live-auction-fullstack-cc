import React from 'react';
import { Link } from 'react-router-dom';
import styles from './Live.module.css';

export interface UpcomingAuctionItem {
  id: number;
  product_id?: number;
  productId?: number;
  product_name?: string;
  productName?: string;
  name?: string;
  title?: string;
  current_price?: number | string | null;
  start_price?: number | string | null;
  start_time?: string;
  startTime?: string;
  product?: {
    id?: number;
    name?: string;
  };
}

interface LiveEmptyStateProps {
  upcomingAuctions: UpcomingAuctionItem[];
  subscribedProductIds: Set<number>;
  pendingProductId: number | null;
  onAuctionClick: (auctionId: number) => void;
  onSubscribe: (productId?: number, auctionId?: number) => void;
}

const formatStartTime = (value?: string) => {
  if (!value) return '即将';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '即将';
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
};

const formatPriceHint = (auction: UpcomingAuctionItem) => {
  const value = auction.start_price ?? auction.current_price;
  const numeric = Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) return '开拍详情待公布';
  return `起拍 ¥${numeric.toLocaleString('zh-CN')}`;
};

const getProductId = (auction: UpcomingAuctionItem) => auction.product_id ?? auction.productId ?? auction.product?.id;

const getProductName = (auction: UpcomingAuctionItem) =>
  auction.product_name ||
  auction.productName ||
  auction.product?.name ||
  auction.name ||
  auction.title ||
  `竞拍场次 #${auction.id}`;

const LiveEmptyState: React.FC<LiveEmptyStateProps> = ({
  upcomingAuctions,
  subscribedProductIds,
  pendingProductId,
  onAuctionClick,
  onSubscribe,
}) => {
  const visibleAuctions = upcomingAuctions.slice(0, 2);

  if (visibleAuctions.length === 0) {
    return (
      <section className={styles.liveEmptyPage} aria-live="polite">
        <div className={styles.liveEmptyPanel}>
          <h1 className={styles.liveEmptyTitle}>当前没有竞拍直播</h1>
          <p className={styles.liveEmptyText}>可以先看看正在预热的拍品，开拍提醒会第一时间通知你。</p>
          <Link className={styles.liveEmptyPrimaryLink} to="/">
            去首页看拍品
          </Link>
        </div>
      </section>
    );
  }

  return (
    <section className={styles.liveEmptyPage} aria-live="polite">
      <div className={styles.liveEmptyPanel}>
        <h1 className={styles.liveEmptyTitle}>下一场竞拍正在准备</h1>
        <p className={styles.liveEmptyText}>当前没有正在竞拍的直播间。先订阅感兴趣的预告场次，开拍前会提醒你回来。</p>
        <div className={styles.upcomingHeader}>即将开播</div>
        <div className={styles.upcomingList}>
          {visibleAuctions.map((auction) => {
            const productId = getProductId(auction);
            const subscribed = typeof productId === 'number' && subscribedProductIds.has(productId);
            const pending = typeof productId === 'number' && pendingProductId === productId;
            const buttonLabel = pending ? '订阅中...' : subscribed ? '已订阅' : '订阅';

            return (
              <article
                key={auction.id}
                className={styles.upcomingCard}
                role="button"
                tabIndex={0}
                onClick={() => onAuctionClick(auction.id)}
                onKeyDown={(event) => {
                  if (event.key !== 'Enter' && event.key !== ' ') return;
                  event.preventDefault();
                  onAuctionClick(auction.id);
                }}
              >
                <div className={styles.upcomingTime}>{formatStartTime(auction.start_time ?? auction.startTime)}</div>
                <div className={styles.upcomingInfo}>
                  <strong>{getProductName(auction)}</strong>
                  <span>{formatPriceHint(auction)} · 点击查看详情</span>
                </div>
                <button
                  type="button"
                  className={styles.upcomingSubscribe}
                  disabled={!productId || subscribed || pending}
                  onClick={(event) => {
                    event.stopPropagation();
                    onSubscribe(productId, auction.id);
                  }}
                >
                  {buttonLabel}
                </button>
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
};

export default LiveEmptyState;
