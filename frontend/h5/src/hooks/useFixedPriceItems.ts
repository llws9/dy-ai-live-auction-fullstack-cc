import { useEffect, useMemo, useReducer } from 'react';
import { fetchItems, type FixedPriceItem } from '../api/fixedPrice';
import WebSocketService from '../services/websocket';

const FIXED_PRICE_MESSAGE_TYPES = [
  'fixed_price_listed',
  'fixed_price_stock',
  'fixed_price_sold_out',
  'fixed_price_offline',
] as const;

type FixedPriceMessageType = typeof FIXED_PRICE_MESSAGE_TYPES[number];

type FixedPriceAction =
  | { type: 'init'; payload: { items: FixedPriceItem[] } }
  | { type: 'fixed_price_listed'; payload: { item?: FixedPriceItem } & Partial<FixedPriceItem> & { item_id?: number } }
  | { type: 'fixed_price_stock'; payload: { item_id: number; remaining_stock: number } }
  | { type: 'fixed_price_sold_out'; payload: { item_id: number } }
  | { type: 'fixed_price_offline'; payload: { item_id: number } }
  | { type: string; payload: unknown };

function itemFromListedPayload(payload: FixedPriceAction['payload']): FixedPriceItem | null {
  if (!payload || typeof payload !== 'object') {
    return null;
  }

  const listed = payload as { item?: FixedPriceItem } & Partial<FixedPriceItem> & { item_id?: number };
  if (listed.item) {
    return listed.item;
  }

  const id = listed.id ?? listed.item_id;
  if (!id || !listed.price || listed.total_stock === undefined || listed.remaining_stock === undefined) {
    return null;
  }

  return {
    id,
    product_id: listed.product_id,
    product: listed.product,
    product_brief: listed.product_brief,
    price: listed.price,
    total_stock: listed.total_stock,
    remaining_stock: listed.remaining_stock,
    status: listed.status ?? 'on_sale',
  };
}

export function reduceItems(state: FixedPriceItem[], action: FixedPriceAction): FixedPriceItem[] {
  switch (action.type) {
    case 'init':
      return (action.payload as { items: FixedPriceItem[] }).items;

    case 'fixed_price_listed': {
      const item = itemFromListedPayload(action.payload);
      if (!item) {
        return state;
      }
      return state.some((current) => current.id === item.id)
        ? state.map((current) => (current.id === item.id ? { ...current, ...item } : current))
        : [...state, item];
    }

    case 'fixed_price_stock': {
      const { item_id: itemId, remaining_stock: remainingStock } = action.payload as {
        item_id: number;
        remaining_stock: number;
      };
      return state.map((item) => (
        item.id === itemId ? { ...item, remaining_stock: remainingStock } : item
      ));
    }

    case 'fixed_price_sold_out': {
      const { item_id: itemId } = action.payload as { item_id: number };
      return state.map((item) => (
        item.id === itemId ? { ...item, remaining_stock: 0, status: 'sold_out' } : item
      ));
    }

    case 'fixed_price_offline': {
      const { item_id: itemId } = action.payload as { item_id: number };
      return state.filter((item) => item.id !== itemId);
    }

    default:
      return state;
  }
}

export function useFixedPriceItems(liveStreamId: number) {
  const [items, dispatch] = useReducer(reduceItems, [] as FixedPriceItem[]);

  useEffect(() => {
    let active = true;

    fetchItems(liveStreamId)
      .then((response) => {
        if (active) {
          dispatch({ type: 'init', payload: { items: response.items } });
        }
      })
      .catch(() => {
        if (active) {
          dispatch({ type: 'init', payload: { items: [] } });
        }
      });

    return () => {
      active = false;
    };
  }, [liveStreamId]);

  useEffect(() => {
    const token = localStorage.getItem('auth_token') ?? localStorage.getItem('token') ?? undefined;
    const socket = new WebSocketService(liveStreamId, token);
    const handlers = FIXED_PRICE_MESSAGE_TYPES.map((type: FixedPriceMessageType) => {
      const handler = (payload: unknown) => dispatch({ type, payload });
      socket.on(type, handler);
      return { type, handler };
    });

    socket.connect().catch(() => {
      // Live fixed-price cards still render from REST when the realtime channel is unavailable.
    });

    return () => {
      handlers.forEach(({ type, handler }) => socket.off(type, handler));
      socket.disconnect();
    };
  }, [liveStreamId]);

  const byId = useMemo<Record<number, FixedPriceItem>>(() => {
    return items.reduce<Record<number, FixedPriceItem>>((index, item) => {
      index[item.id] = item;
      return index;
    }, {});
  }, [items]);

  return { items, byId };
}
