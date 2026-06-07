import React from 'react';
import { useNavigate } from 'react-router-dom';
import styles from './LiveRoomCard.module.css';

export interface NextAuction {
  auction_id: number;
  product_id: number;
  product_name: string;
  start_price: string;
  start_time: string;
}

export interface RecentDeal {
  product_name: string;
  final_price: string;
}

export interface LiveRoomItem {
  id: number;
  name: string;
  status: number;
  cover_image?: string;
  host_name?: string;
  host_avatar?: string;
  viewer_count?: number;
  current_auction_id?: number | null;
  current_product_id?: number | null;
  current_price?: string | null;
  next_auction?: NextAuction | null;
  recent_deals?: RecentDeal[];
}

interface Props {
  room: LiveRoomItem;
  onSubscribe?: (productId?: number, auctionId?: number) => void;
  onEnter?: (liveStreamId: number, auctionId?: number) => void;
  subscribedProductIds?: Set<number>;
}

const hasCurrent = (room: LiveRoomItem) =>
  room.current_auction_id != null && Number(room.current_auction_id) > 0;

const LiveRoomCard: React.FC<Props> = ({ room, onSubscribe, onEnter }) => {
  const navigate = useNavigate();
  const live = hasCurrent(room);
  const next = room.next_auction;
  const deals = room.recent_deals ?? [];

  const enter = () => {
    const auctionId = Number(room.current_auction_id) || undefined;
    if (onEnter) {
      onEnter(room.id, auctionId);
      return;
    }
    navigate(`/live?id=${room.id}&auction_id=${auctionId ?? ''}`);
  };

  return (
    <article className={styles.card}>
      <div className={styles.imageWrapper}>
        {room.cover_image && <img className={styles.cover} src={room.cover_image} alt={room.name} />}
        <span className={live ? styles.statusLive : styles.statusUpcoming}>
          {live ? '直播中' : '即将开始'}
        </span>
        {typeof room.viewer_count === 'number' && (
          <span className={styles.viewers}>{room.viewer_count} 在线</span>
        )}
      </div>
      <div className={styles.body}>
        <h3 className={styles.name}>{room.name}</h3>
        {live ? (
          <p className={styles.price}>当前 ¥{room.current_price ?? '—'}</p>
        ) : next ? (
          <p className={styles.nextLine}>
            即将开拍：<span className={styles.nextProduct}>{next.product_name}</span>（起拍 ¥{next.start_price}）
          </p>
        ) : null}

        {deals.length > 0 && (
          <ul className={styles.deals} aria-label="最近成交">
            {deals.map((d, i) => (
              <li key={i} className={styles.dealItem}>
                {d.product_name} 已成交 ¥{d.final_price}
              </li>
            ))}
          </ul>
        )}

        <div className={styles.actions}>
          {live ? (
            <button type="button" className={styles.primaryButton} onClick={enter}>
              进入直播间
            </button>
          ) : (
            <button
              type="button"
              className={styles.secondaryButton}
              onClick={() => onSubscribe?.(next?.product_id, next?.auction_id)}
            >
              预约开拍提醒
            </button>
          )}
        </div>
      </div>
    </article>
  );
};

export default LiveRoomCard;
