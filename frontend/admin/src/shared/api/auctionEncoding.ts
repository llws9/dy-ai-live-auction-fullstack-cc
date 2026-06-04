import { repairUtf8Mojibake } from '../../utils/textEncoding';
import { normalizeProductText } from './productEncoding';

type AuctionLike = object & {
  title?: string | null;
  live_stream_name?: string | null;
  product?: (object & {
    name?: string | null;
    description?: string | null;
    category?: string | null;
    brand?: string | null;
  }) | null;
};

export function normalizeAuctionText<T extends AuctionLike>(auction: T): T {
  return {
    ...auction,
    ...(typeof auction.title === 'string' ? { title: repairUtf8Mojibake(auction.title) } : {}),
    ...(typeof auction.live_stream_name === 'string'
      ? { live_stream_name: repairUtf8Mojibake(auction.live_stream_name) }
      : {}),
    ...(auction.product ? { product: normalizeProductText(auction.product) } : {}),
  };
}

export function normalizeAuctionListResponse<T extends AuctionLike, R extends { list?: T[] }>(response: R): R {
  if (!Array.isArray(response.list)) {
    return response;
  }

  return {
    ...response,
    list: response.list.map(normalizeAuctionText),
  };
}
