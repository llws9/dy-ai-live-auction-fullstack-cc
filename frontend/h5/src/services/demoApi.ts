import { ApiError } from './api';

const DEMO_BASE_URL = '/api/test/demo';

type MoneyInput = string | number;

export type TriggerFollowBidInput = {
  auctionId: number;
  amount?: MoneyInput;
  increment?: MoneyInput;
};

export type RechargeDemoUserInput = {
  userId: number;
  amount: MoneyInput;
};

export type ShortenDemoAuctionInput = {
  auctionId: number;
  remainingSeconds: number;
};

export type TriggerOtherSkyLampInput = {
  auctionId: number;
};

export type DemoMerchantAuctionMode = 'upcoming' | 'ongoing';

export type CreateDemoFixedPriceItemInput = {
  auctionId: number;
  liveStreamId: number;
};

export type CreateDemoFixedPriceItemResponse = {
  ok: boolean;
  item_id: number;
  product_id: number;
  live_stream_id: number;
  price: string;
  stock: number;
};

function toMoneyString(value: MoneyInput): string {
  return String(value);
}

function getStoredToken(): string | null {
  return localStorage.getItem('auth_token') || localStorage.getItem('token');
}

function buildHeaders(): Record<string, string> {
  const token = getStoredToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  return headers;
}

async function readJson(response: Response): Promise<any> {
  try {
    return await response.json();
  } catch (_error) {
    return {};
  }
}

function getErrorMessage(data: any, status: number): string {
  return data?.error || data?.message || data?.msg || `请求失败 (${status})`;
}

async function postDemo<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${DEMO_BASE_URL}${path}`, {
    method: 'POST',
    headers: buildHeaders(),
    body: JSON.stringify(body),
  });

  const data = await readJson(response);
  if (!response.ok) {
    throw new ApiError(getErrorMessage(data, response.status), response.status, data?.code, data);
  }

  return data as T;
}

export function triggerFollowBid(input: TriggerFollowBidInput) {
  const body: Record<string, unknown> = {
    auction_id: input.auctionId,
  };

  if (input.amount !== undefined) {
    body.amount = toMoneyString(input.amount);
  }
  if (input.increment !== undefined) {
    body.increment = toMoneyString(input.increment);
  }

  return postDemo('/follow-bid', body);
}

export function rechargeDemoUser(input: RechargeDemoUserInput) {
  return postDemo('/recharge', {
    user_id: input.userId,
    amount: toMoneyString(input.amount),
  });
}

export function shortenDemoAuction(input: ShortenDemoAuctionInput) {
  return postDemo('/auctions/shorten', {
    auction_id: input.auctionId,
    remaining_seconds: input.remainingSeconds,
  });
}

export function triggerOtherSkyLamp(input: TriggerOtherSkyLampInput) {
  return postDemo('/sky-lamp', {
    auction_id: input.auctionId,
  });
}

export function createDemoMerchantAuction(mode: DemoMerchantAuctionMode) {
  return postDemo('/merchant/auctions', { mode });
}

export function createDemoFixedPriceItem(input: CreateDemoFixedPriceItemInput) {
  return postDemo('/merchant/fixed-price-items', {
    auction_id: input.auctionId,
    live_stream_id: input.liveStreamId,
  }) as Promise<CreateDemoFixedPriceItemResponse>;
}
