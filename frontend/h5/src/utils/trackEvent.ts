import { IS_DEV } from '@/utils/env';

export type TouchpointEventName =
  | 'summary_exposed'
  | 'entry_clicked'
  | 'notification_list_exposed'
  | 'notification_item_clicked'
  | 'mark_read'
  | 'hot_pull_triggered'
  | 'live_reminder_exposed'
  | 'live_reminder_clicked'
  | 'live_reminder_dismissed';

export type TouchpointSource =
  | 'home'
  | 'bottom_nav'
  | 'profile'
  | 'notification_center'
  | 'mobile_shell'
  | 'notification_hook';

export type TouchpointEntry =
  | 'notification_bell'
  | 'profile_tab'
  | 'auction_history'
  | 'notification_center'
  | 'notification_item'
  | 'mark_all_read'
  | 'hot_pull'
  | 'live_reminder_modal';

export type TouchpointType =
  | 'all'
  | 'pending_payment'
  | 'outbid'
  | 'ending_soon'
  | 'live_start'
  | 'notification';

export type TouchpointResult = 'success' | 'failed' | 'clicked' | 'dismissed' | 'debounced';
export type CountBucket = '0' | '1' | '2_5' | '6_10' | '10_plus';

export interface TouchpointEventParams {
  source: TouchpointSource;
  entry: TouchpointEntry;
  type: TouchpointType;
  result: TouchpointResult;
  countBucket?: CountBucket;
}

interface TrackEventPayload {
  event_type: 'touchpoint_event';
  event_name: TouchpointEventName;
  params: {
    source: TouchpointSource;
    entry: TouchpointEntry;
    type: TouchpointType;
    result: TouchpointResult;
    count_bucket?: CountBucket;
  };
  timestamp: number;
}

const TRACK_ENDPOINT = '/api/v1/track';

export function getCountBucket(count: number): CountBucket {
  if (count <= 0) return '0';
  if (count === 1) return '1';
  if (count <= 5) return '2_5';
  if (count <= 10) return '6_10';
  return '10_plus';
}

function buildPayload(eventName: TouchpointEventName, params: TouchpointEventParams): TrackEventPayload {
  return {
    event_type: 'touchpoint_event',
    event_name: eventName,
    params: {
      source: params.source,
      entry: params.entry,
      type: params.type,
      result: params.result,
      ...(params.countBucket ? { count_bucket: params.countBucket } : {}),
    },
    timestamp: Date.now(),
  };
}

function reportFailure(error: unknown) {
  if (IS_DEV) {
    console.warn('[trackEvent] failed to report touchpoint event', error);
  }
}

export function trackEvent(eventName: TouchpointEventName, params: TouchpointEventParams): void {
  const body = JSON.stringify(buildPayload(eventName, params));
  const blob = new Blob([body], { type: 'application/json' });

  try {
    if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {
      const sent = navigator.sendBeacon(TRACK_ENDPOINT, blob);
      if (sent) return;
    }

    if (typeof fetch === 'function') {
      void fetch(TRACK_ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: blob,
        keepalive: true,
      }).catch(reportFailure);
    }
  } catch (error) {
    reportFailure(error);
  }
}
