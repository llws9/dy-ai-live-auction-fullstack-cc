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
  product_title?: string;
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
  purchase_id: number;
  item_id: number;
  price: string;
  remaining_stock: number;
  status: 'success';
}

export interface MyPurchaseResult {
  i_bought: boolean;
  purchase_id?: number;
  price?: string;
  created_at?: string;
}

interface PurchaseResponse {
  order_id: number;
  item_id: number;
  price: string;
  remaining_stock: number;
  status: 'success';
}

interface MyPurchaseResponse {
  i_bought: boolean;
  order_id?: number;
  price?: string;
  created_at?: string;
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

export function fetchMyPurchase(itemId: number): Promise<MyPurchaseResult> {
  return get<MyPurchaseResponse>(`/fixed-price/items/${itemId}/my-purchase`).then((res) => ({
    i_bought: res.i_bought,
    purchase_id: res.order_id,
    price: res.price,
    created_at: res.created_at,
  }));
}

export function purchase(params: {
  itemId: number;
  idempotencyKey: string;
}): Promise<PurchaseResult> {
  return post<PurchaseResponse>(`/fixed-price/items/${params.itemId}/purchase`, undefined, {
    showError: false,
    headers: {
      'X-Idempotency-Key': params.idempotencyKey,
    },
  }).then((res) => ({
    purchase_id: res.order_id,
    item_id: res.item_id,
    price: res.price,
    remaining_stock: res.remaining_stock,
    status: res.status,
  }));
}
