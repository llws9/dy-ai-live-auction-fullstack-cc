import { useEffect, useRef, useState, type CSSProperties } from 'react';
import styles from './index.module.css';

const FLAIR_MESSAGE_TYPE = 'fixed_price_flair';
const FLAIR_DURATION_MS = 4000;
const MAX_VISIBLE_FLAIRS = 3;

type FixedPriceFlairPayload = {
  item_id?: number;
  buyer_id?: number;
  buyer_nickname?: string;
  product_title?: string;
  price: string;
};

type FixedPriceFlairMessage = {
  payload?: FixedPriceFlairPayload;
  data?: FixedPriceFlairPayload;
} & Partial<FixedPriceFlairPayload>;

type FlairHandler = (message: FixedPriceFlairMessage | FixedPriceFlairPayload) => void;

type SubscribeSocket = {
  subscribe: (type: string, handler: FlairHandler) => void | (() => void);
};

type OnOffSocket = {
  on: (type: string, handler: FlairHandler) => void;
  off?: (type: string, handler: FlairHandler) => void;
};

type FixedPriceFlairSocket = SubscribeSocket | OnOffSocket;

type VisibleFlair = FixedPriceFlairPayload & {
  id: string;
};

interface FixedPriceFlairProps {
  socket: FixedPriceFlairSocket | null | undefined;
}

function isSubscribeSocket(socket: FixedPriceFlairSocket): socket is SubscribeSocket {
  return 'subscribe' in socket && typeof socket.subscribe === 'function';
}

function isOnOffSocket(socket: FixedPriceFlairSocket): socket is OnOffSocket {
  return 'on' in socket && typeof socket.on === 'function';
}

function normalizePayload(message: FixedPriceFlairMessage | FixedPriceFlairPayload): FixedPriceFlairPayload | null {
  const wrapped = message as FixedPriceFlairMessage;
  const candidate = wrapped.payload ?? wrapped.data ?? message;

  if (!candidate?.price) {
    return null;
  }

  const buyerNickname = candidate.buyer_nickname || (candidate.buyer_id ? `用户 #${candidate.buyer_id}` : '');
  const productTitle = candidate.product_title || (candidate.item_id ? `商品 #${candidate.item_id}` : '');

  if (!buyerNickname || !productTitle) {
    return null;
  }

  return {
    item_id: candidate.item_id,
    buyer_id: candidate.buyer_id,
    buyer_nickname: buyerNickname,
    product_title: productTitle,
    price: candidate.price,
  };
}

export default function FixedPriceFlair({ socket }: FixedPriceFlairProps) {
  const [flairs, setFlairs] = useState<VisibleFlair[]>([]);
  const sequenceRef = useRef(0);
  const timersRef = useRef<number[]>([]);

  useEffect(() => {
    if (!socket) {
      return undefined;
    }

    const handleFlair: FlairHandler = (message) => {
      const payload = normalizePayload(message);
      if (!payload) {
        return;
      }

      sequenceRef.current += 1;
      const flair: VisibleFlair = {
        ...payload,
        id: `${payload.item_id ?? 'item'}-${Date.now()}-${sequenceRef.current}`,
      };

      setFlairs((current) => [...current, flair].slice(-MAX_VISIBLE_FLAIRS));

      const timer = window.setTimeout(() => {
        setFlairs((current) => current.filter((item) => item.id !== flair.id));
        timersRef.current = timersRef.current.filter((current) => current !== timer);
      }, FLAIR_DURATION_MS);
      timersRef.current.push(timer);
    };

    const unsubscribe = isSubscribeSocket(socket)
      ? socket.subscribe(FLAIR_MESSAGE_TYPE, handleFlair)
      : undefined;

    if (!isSubscribeSocket(socket) && isOnOffSocket(socket)) {
      socket.on(FLAIR_MESSAGE_TYPE, handleFlair);
    }

    return () => {
      if (typeof unsubscribe === 'function') {
        unsubscribe();
      } else if (!isSubscribeSocket(socket) && isOnOffSocket(socket)) {
        socket.off?.(FLAIR_MESSAGE_TYPE, handleFlair);
      }
    };
  }, [socket]);

  useEffect(() => {
    return () => {
      timersRef.current.forEach((timer) => window.clearTimeout(timer));
      timersRef.current = [];
    };
  }, []);

  if (flairs.length === 0) {
    return null;
  }

  return (
    <div className={styles.viewport} aria-live="polite" aria-label="一口价购买飘屏">
      {flairs.map((flair, index) => (
        <div
          className={styles.flair}
          data-flair
          key={flair.id}
          style={{ '--flair-row': index } as CSSProperties}
        >
          <span className={styles.spark}>抢</span>
          <span className={styles.copy}>
            <strong>{flair.buyer_nickname}</strong>
            <span> 刚刚抢到 {flair.product_title}</span>
          </span>
          <span className={styles.price}>¥{flair.price}</span>
        </div>
      ))}
    </div>
  );
}
