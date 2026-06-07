import { normalizeUserName } from './userEncoding';

type BidLike = object & {
  user_id?: number | null;
  user_name?: string | null;
  amount?: number | string | null;
  price?: number | string | null;
};

function toFiniteNumber(value: number | string | null | undefined): number {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : 0;
  }
  if (typeof value === 'string') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

export function normalizeBid<T extends BidLike>(bid: T): T & { amount: number; price: number; user_name?: string } {
  const price = toFiniteNumber(bid.price ?? bid.amount);
  const userName = normalizeUserName(bid.user_id, bid.user_name);

  return {
    ...bid,
    amount: price,
    price,
    ...(userName ? { user_name: userName } : {}),
  };
}

export function normalizeBids<T extends BidLike>(bids: T[]): Array<T & { amount: number; price: number; user_name?: string }> {
  return bids.map(normalizeBid);
}
