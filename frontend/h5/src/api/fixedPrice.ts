import { get, post } from '../services/api';

export interface ProductBrief {
  id: number;
  title: string;
  cover_image?: string;
}

export type FixedPriceItemStatus = 'on_sale' | 'sold_out' | 'offline' | 'live';

export interface FixedPriceItem {
  id: number;
  product_id?: number;
  product?: ProductBrief;
  price: string;
  total_stock: number;
  remaining_stock: number;
  status: FixedPriceItemStatus;
  product_brief?: ProductBrief;
}

export interface FixedPriceItemsResponse {
  items: FixedPriceItem[];
}

export interface PurchaseResult {
  order_id: number;
  item_id: number;
  price: string;
  remaining_stock: number;
  status: 'success';
}

export function generateIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }

  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (char) => {
    const r = Math.floor(Math.random() * 16);
    const v = char === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export function fetchItems(liveStreamId: number): Promise<FixedPriceItemsResponse> {
  return get<FixedPriceItemsResponse>(`/live-streams/${liveStreamId}/fixed-price/items`);
}

export function purchase(params: {
  itemId: number;
  idempotencyKey: string;
}): Promise<PurchaseResult> {
  return post<PurchaseResult>(`/fixed-price/items/${params.itemId}/purchase`, undefined, {
    headers: {
      'X-Idempotency-Key': params.idempotencyKey,
    },
  });
}
