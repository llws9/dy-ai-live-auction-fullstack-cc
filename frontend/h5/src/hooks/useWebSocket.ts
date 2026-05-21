// hooks/useWebSocket.ts

import { useEffect, useRef, useState } from 'react';
import WebSocketService from '../services/websocket';

interface UseWebSocketOptions {
  auctionId: number;
  onBidPlaced?: (data: any) => void;
  onRankUpdate?: (data: any) => void;
  onDelayTriggered?: (data: any) => void;
  onAuctionEnded?: (data: any) => void;
  onTimeSync?: (data: any) => void;
  onError?: (data: any) => void;
}

export function useWebSocket(options: UseWebSocketOptions) {
  const {
    auctionId,
    onBidPlaced,
    onRankUpdate,
    onDelayTriggered,
    onAuctionEnded,
    onTimeSync,
    onError,
  } = options;

  const wsRef = useRef<WebSocketService | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    wsRef.current = new WebSocketService(auctionId);

    // 注册事件处理器
    if (onBidPlaced) {
      wsRef.current.on('bid_placed', onBidPlaced);
    }
    if (onRankUpdate) {
      wsRef.current.on('rank_update', onRankUpdate);
    }
    if (onDelayTriggered) {
      wsRef.current.on('delay_triggered', onDelayTriggered);
    }
    if (onAuctionEnded) {
      wsRef.current.on('auction_ended', onAuctionEnded);
    }
    if (onTimeSync) {
      wsRef.current.on('time_sync', onTimeSync);
    }
    if (onError) {
      wsRef.current.on('error', onError);
    }

    // 连接状态处理
    wsRef.current.on('pong', () => setConnected(true));
    wsRef.current.on('time_sync', () => setConnected(true));

    // 建立连接
    wsRef.current.connect().then(() => {
      setConnected(true);
    }).catch((error) => {
      console.error('WebSocket connection failed:', error);
      setConnected(false);
    });

    return () => {
      if (wsRef.current) {
        wsRef.current.disconnect();
      }
    };
  }, [auctionId]);

  return {
    connected,
    send: (message: any) => wsRef.current?.send(message),
  };
}
