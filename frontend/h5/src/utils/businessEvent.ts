import { IS_DEV } from '@/utils/env';

export type BusinessEventType =
  | 'reminder_subscribe'
  | 'reminder_click'
  | 'live_room_enter'
  | 'bid_button_click'
  | 'fixed_price_click'
  | 'purchase_success'
  | 'auction_win'
  | 'notification_expose'
  | 'notification_click';

export type BusinessEventSource =
  | 'home'
  | 'live_room'
  | 'live_reminder'
  | 'notification_center'
  | 'product_detail'
  | 'auction_card'
  | 'fixed_price_card'
  | 'unknown';

export interface BusinessEventParams {
  source: BusinessEventSource;
  liveStreamId?: number;
  auctionId?: number;
  productId?: number;
  metadata?: Record<string, unknown>;
}

const BUSINESS_EVENT_ENDPOINT = '/api/v1/events';

function getBusinessEventToken(): string | null {
  if (typeof localStorage === 'undefined') {
    return null;
  }
  return localStorage.getItem('auth_token') || localStorage.getItem('token');
}

function reportFailure(error: unknown) {
  if (IS_DEV) {
    console.warn('[businessEvent] failed to report business event', error);
  }
}

export function trackBusinessEvent(eventType: BusinessEventType, params: BusinessEventParams): void {
  const token = getBusinessEventToken();
  if (!token || typeof fetch !== 'function') {
    return;
  }

  const body = JSON.stringify({
    event_type: eventType,
    source: params.source,
    ...(params.liveStreamId ? { live_stream_id: params.liveStreamId } : {}),
    ...(params.auctionId ? { auction_id: params.auctionId } : {}),
    ...(params.productId ? { product_id: params.productId } : {}),
    ...(params.metadata ? { metadata: params.metadata } : {}),
  });

  try {
    void fetch(BUSINESS_EVENT_ENDPOINT, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body,
      keepalive: true,
    }).catch(reportFailure);
  } catch (error) {
    reportFailure(error);
  }
}
