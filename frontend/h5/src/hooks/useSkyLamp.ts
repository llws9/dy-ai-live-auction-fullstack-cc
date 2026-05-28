// hooks/useSkyLamp.ts

import { useState, useEffect, useCallback } from 'react';
import WebSocketService from '../services/websocket';
import { activateSkyLamp, cancelSkyLamp, getSkyLampStatus } from '../services/skyLamp';

interface SkyLampStatus {
  active: boolean;
  subscription_id: number;
  max_price_limit: number;
  auto_bid_count: number;
  total_bid_amount: number;
  remaining_budget: number;
}

interface UseSkyLampOptions {
  auctionId: number;
  userId: number;
  token: string;
  ws: WebSocketService | null;
}

export function useSkyLamp(options: UseSkyLampOptions) {
  const { auctionId, userId, token, ws } = options;
  const [status, setStatus] = useState<SkyLampStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 初始化时获取天灯状态
  useEffect(() => {
    if (auctionId && token) {
      getSkyLampStatus(auctionId, token)
        .then((data) => {
          if (data.active) {
            setStatus(data);
          }
        })
        .catch((err) => {
          console.error('Failed to get sky lamp status:', err);
        });
    }
  }, [auctionId, token]);

  // WebSocket消息监听
  useEffect(() => {
    if (!ws) return;

    const handleSkyLampActivated = (data: any) => {
      if (data.user_id === userId) {
        setStatus({
          active: true,
          subscription_id: data.subscription_id,
          max_price_limit: data.max_price_limit,
          auto_bid_count: 0,
          total_bid_amount: data.initial_bid_amount,
          remaining_budget: data.max_price_limit - data.initial_bid_amount,
        });
      }
    };

    const handleSkyLampAutoBid = (data: any) => {
      if (data.user_id === userId) {
        setStatus(prev => prev ? {
          ...prev,
          auto_bid_count: data.auto_bid_count,
          total_bid_amount: data.amount,
          remaining_budget: data.remaining_budget,
        } : null);
      }
    };

    const handleSkyLampStopped = (data: any) => {
      if (data.user_id === userId) {
        setStatus(null);
      }
    };

    ws.on('sky_lamp_activated', handleSkyLampActivated);
    ws.on('sky_lamp_auto_bid', handleSkyLampAutoBid);
    ws.on('sky_lamp_stopped', handleSkyLampStopped);

    return () => {
      ws.off('sky_lamp_activated', handleSkyLampActivated);
      ws.off('sky_lamp_auto_bid', handleSkyLampAutoBid);
      ws.off('sky_lamp_stopped', handleSkyLampStopped);
    };
  }, [ws, userId]);

  // 开启天灯
  const activate = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await activateSkyLamp(auctionId, token);
      if (result.error) {
        setError(result.error);
        return false;
      }
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to activate sky lamp');
      return false;
    } finally {
      setLoading(false);
    }
  }, [auctionId, token]);

  // 取消天灯
  const cancel = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await cancelSkyLamp(auctionId, token);
      if (result.error) {
        setError(result.error);
        return false;
      }
      setStatus(null);
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to cancel sky lamp');
      return false;
    } finally {
      setLoading(false);
    }
  }, [auctionId, token]);

  return {
    status,
    loading,
    error,
    activate,
    cancel,
  };
}