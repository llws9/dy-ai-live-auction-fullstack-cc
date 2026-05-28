// hooks/useWebSocket.ts

import { useEffect, useRef, useState, useCallback } from 'react';
import WebSocketService from '../services/websocket';

interface UseWebSocketOptions {
  auctionId: number;
  onBidPlaced?: (data: any) => void;
  onRankUpdate?: (data: any) => void;
  onDelayTriggered?: (data: any) => void;
  onAuctionEnded?: (data: any) => void;
  onTimeSync?: (data: any) => void;
  onError?: (data: any) => void;
  onNotification?: (notification: NotificationData) => void;
}

export interface NotificationData {
  id: number;
  type: string;
  title: string;
  content: string;
  data?: Record<string, unknown>;
  created_at: string;
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
    onNotification,
  } = options;

  const wsRef = useRef<WebSocketService | null>(null);
  const [connected, setConnected] = useState(false);

  // 显示浏览器通知
  const showBrowserNotification = useCallback((notification: NotificationData) => {
    if (!('Notification' in window)) {
      return;
    }

    if (Notification.permission === 'granted') {
      new Notification(notification.title, {
        body: notification.content,
        icon: '/favicon.ico',
        data: notification.data,
      });
    } else if (Notification.permission !== 'denied') {
      Notification.requestPermission().then((permission) => {
        if (permission === 'granted') {
          new Notification(notification.title, {
            body: notification.content,
            icon: '/favicon.ico',
          });
        }
      });
    }
  }, []);

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

    // 注册通知处理器
    if (onNotification) {
      wsRef.current.onNotification((notification) => {
        onNotification(notification);
        showBrowserNotification(notification);
      });
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
  }, [auctionId, onBidPlaced, onRankUpdate, onDelayTriggered, onAuctionEnded, onTimeSync, onError, onNotification, showBrowserNotification]);

  return {
    connected,
    send: (message: any) => wsRef.current?.send(message),
  };
}
