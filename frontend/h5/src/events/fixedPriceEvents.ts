import type { FixedPriceItem } from '@/api/fixedPrice';

export const DEMO_FIXED_PRICE_LISTED_EVENT = 'demo:fixed-price-listed';

export interface DemoFixedPriceListedDetail {
  auctionId: number;
  liveStreamId: number;
  item: FixedPriceItem;
}

export function dispatchDemoFixedPriceListed(detail: DemoFixedPriceListedDetail) {
  window.dispatchEvent(new CustomEvent<DemoFixedPriceListedDetail>(DEMO_FIXED_PRICE_LISTED_EVENT, { detail }));
}
