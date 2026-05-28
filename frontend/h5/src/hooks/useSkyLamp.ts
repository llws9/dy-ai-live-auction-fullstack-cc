import { useCallback, useState } from 'react';
import {
  getSkyLampSubscriptions,
  startSkyLampSubscription,
  stopSkyLampSubscription,
} from '../services/skyLamp';

interface SkyLampSubscription {
  id: number;
  auction_id: number;
  status: number;
}

export function useSkyLamp(token: string | null, auctionId: number) {
  const [loading, setLoading] = useState(false);
  const [subscriptionId, setSubscriptionId] = useState<number | null>(null);
  const [active, setActive] = useState(false);

  const refreshStatus = useCallback(async () => {
    if (!token) return;
    try {
      const resp = await getSkyLampSubscriptions(token, 1);
      const list = (resp.subscriptions || resp.data?.subscriptions || []) as SkyLampSubscription[];
      const matched = list.find((x) => x.auction_id === auctionId && x.status === 1);
      if (matched) {
        setSubscriptionId(matched.id);
        setActive(true);
      } else {
        setSubscriptionId(null);
        setActive(false);
      }
    } catch {
      // ignore refresh errors
    }
  }, [token, auctionId]);

  const start = useCallback(async () => {
    if (!token) throw new Error('未登录');
    setLoading(true);
    try {
      const resp = await startSkyLampSubscription(auctionId, token);
      const sub = resp.subscription || resp.data?.subscription;
      if (sub?.id) {
        setSubscriptionId(sub.id);
      }
      setActive(true);
      return resp;
    } finally {
      setLoading(false);
    }
  }, [auctionId, token]);

  const stop = useCallback(async () => {
    if (!token) throw new Error('未登录');

    let targetSubscriptionId = subscriptionId;
    if (!targetSubscriptionId) {
      const resp = await getSkyLampSubscriptions(token, 1);
      const list = (resp.subscriptions || resp.data?.subscriptions || []) as SkyLampSubscription[];
      const matched = list.find((x) => x.auction_id === auctionId && x.status === 1);
      targetSubscriptionId = matched?.id ?? null;
      if (targetSubscriptionId) {
        setSubscriptionId(targetSubscriptionId);
      }
    }

    if (!targetSubscriptionId) {
      throw new Error('未找到可停止的天灯订阅');
    }

    setLoading(true);
    try {
      const resp = await stopSkyLampSubscription(targetSubscriptionId, token);
      setActive(false);
      setSubscriptionId(null);
      return resp;
    } finally {
      setLoading(false);
    }
  }, [token, subscriptionId, auctionId]);

  return {
    loading,
    active,
    subscriptionId,
    refreshStatus,
    start,
    stop,
  };
}
