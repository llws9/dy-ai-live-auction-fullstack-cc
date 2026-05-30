// hooks/useWebSocket.ts

import { useEffect, useRef, useState, useCallback } from 'react';
import WebSocketService from '../services/websocket';
import { useAuth } from '../store/authContext';

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
  const { auctionId } = options;
  const { token } = useAuth();

  const wsRef = useRef<WebSocketService | null>(null);
  const [connected, setConnected] = useState(false);

  // 使用 ref 持有最新回调，避免父组件重渲染（未 memoize 回调）导致断连重连
  const handlersRef = useRef(options);
  handlersRef.current = options;

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
    const ws = new WebSocketService(auctionId, token || undefined);
    wsRef.current = ws;

    // 注册事件处理器：通过 handlersRef 间接调用，回调更新无需重连
    ws.on('bid_placed', (data) => handlersRef.current.onBidPlaced?.(data));
    ws.on('rank_update', (data) => handlersRef.current.onRankUpdate?.(data));
    ws.on('delay_triggered', (data) => handlersRef.current.onDelayTriggered?.(data));
    ws.on('auction_ended', (data) => handlersRef.current.onAuctionEnded?.(data));
    ws.on('time_sync', (data) => handlersRef.current.onTimeSync?.(data));
    ws.on('error', (data) => handlersRef.current.onError?.(data));

    // 注册通知处理器
    ws.onNotification((notification) => {
      handlersRef.current.onNotification?.(notification);
      showBrowserNotification(notification);
    });

    // 连接状态处理
    ws.on('pong', () => setConnected(true));
    ws.on('time_sync', () => setConnected(true));

    // 建立连接
    ws.connect()
      .then(() => setConnected(true))
      .catch((error) => {
        console.error('WebSocket connection failed:', error);
        setConnected(false);
      });

    return () => {
      ws.disconnect();
      if (wsRef.current === ws) {
        wsRef.current = null;
      }
    };
  }, [auctionId, token, showBrowserNotification]);

  return {
    connected,
    send: (message: any) => wsRef.current?.send(message),
  };
}
