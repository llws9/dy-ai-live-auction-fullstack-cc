// services/websocket.ts

import { MessageTypeThrottlers } from '../utils/throttle';
import { buildLoginRedirectPath } from './api';

// 服务端鉴权失败的 WebSocket 关闭码（与后端约定）
const WS_AUTH_FAILED_CLOSE_CODE = 4401;

type MessageHandler = (data: any) => void;

interface Message {
  type: string;
  timestamp: number;
  data?: any;
}

// 通知消息类型
interface NotificationMessage {
  id: number;
  type: string;
  title: string;
  content: string;
  data?: Record<string, unknown>;
  created_at: string;
}

class WebSocketService {
  private ws: WebSocket | null = null;
  private auctionId: number;
  private token: string | null;
  private liveStreamId: number | null;
  private handlers: Map<string, Set<MessageHandler>> = new Map();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private pingInterval: NodeJS.Timeout | null = null;
  private isConnecting = false;
  private connectingPromise: Promise<void> | null = null;
  private isManualClose = false;
  private lastMessage: Message | null = null;
  private notificationCallbacks: Set<(notification: NotificationMessage) => void> = new Set();
  private messageThrottlers: MessageTypeThrottlers;

  // 指数退避延迟序列（秒）
  private reconnectDelays = [1, 2, 4, 8, 16, 30, 30, 30, 30, 30];

  constructor(auctionId: number, token?: string, liveStreamId?: number) {
    this.auctionId = auctionId;
    this.token = token || null;
    this.liveStreamId = liveStreamId ?? null;
    this.messageThrottlers = new MessageTypeThrottlers(200);
    this.setupThrottlers();
  }

  /**
   * 设置消息节流器
   */
  private setupThrottlers(): void {
    // 排名更新节流: 200ms内只处理最新一条
    this.messageThrottlers.createThrottler('rank_update', (data) => {
      const handlers = this.handlers.get('rank_update');
      if (handlers) {
        handlers.forEach((handler) => handler(data));
      }
    }, 200);

    // 价格更新节流: 100ms内只处理最新一条
    this.messageThrottlers.createThrottler('bid_placed', (data) => {
      const handlers = this.handlers.get('bid_placed');
      if (handlers) {
        handlers.forEach((handler) => handler(data));
      }
    }, 100);
  }

  connect(): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return Promise.resolve();
    }

    // 复用进行中的连接 Promise，避免调用方挂起
    if (this.isConnecting && this.connectingPromise) {
      return this.connectingPromise;
    }

    this.isConnecting = true;
    this.isManualClose = false;

    this.connectingPromise = new Promise<void>((resolve, reject) => {
      // 根据当前页面协议自动选择 ws / wss，避免 HTTPS 站点 mixed-content 拦截
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      let wsUrl = `${proto}//${window.location.host}/api/v1/ws?auction_id=${this.auctionId}`;
      if (this.token) {
        wsUrl += `&token=${encodeURIComponent(this.token)}`;
      }
      if (this.liveStreamId) {
        wsUrl += `&live_stream_id=${this.liveStreamId}`;
      }

      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.isConnecting = false;
        this.connectingPromise = null;
        this.reconnectAttempts = 0;
        this.startPing();
        resolve();
      };

      this.ws.onmessage = (event) => {
        try {
          const message: Message = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (error) {
          console.error('Failed to parse message:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.isConnecting = false;
        this.connectingPromise = null;
        reject(error);
      };

      this.ws.onclose = (event) => {
        console.log('WebSocket closed', event.code);
        this.isConnecting = false;
        this.connectingPromise = null;
        this.stopPing();

        // 鉴权失败：清理本地凭据并重定向登录页，停止重连
        if (event.code === WS_AUTH_FAILED_CLOSE_CODE) {
          this.isManualClose = true;
          try {
            localStorage.removeItem('auth_token');
            localStorage.removeItem('auth_user');
            localStorage.removeItem('token');
          } catch {
            // ignore storage errors
          }
          if (window.location.pathname !== '/login') {
            window.location.href = buildLoginRedirectPath();
          }
          return;
        }

        if (!this.isManualClose) {
          this.scheduleReconnect();
        }
      };
    });

    return this.connectingPromise;
  }

  disconnect(): void {
    this.isManualClose = true;
    this.stopPing();
    this.stopReconnect();
    this.messageThrottlers.clearAll();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  on(type: string, handler: MessageHandler): void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set());
    }
    this.handlers.get(type)!.add(handler);
  }

  off(type: string, handler: MessageHandler): void {
    const handlers = this.handlers.get(type);
    if (handlers) {
      handlers.delete(handler);
    }
  }

  send(message: any): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  /** 发送弹幕 */
  sendChat(text: string, clientMsgId: string): void {
    if (this.ws?.readyState !== WebSocket.OPEN) return;
    if (!this.liveStreamId) return;
    const payload = {
      type: 'chat_send',
      timestamp: Date.now(),
      data: {
        live_stream_id: this.liveStreamId,
        text,
        client_msg_id: clientMsgId,
      },
    };
    this.ws.send(JSON.stringify(payload));
  }

  // 请求状态同步（重连后使用）
  requestSync(): void {
    this.send({
      type: 'sync_request',
      data: { auction_id: this.auctionId }
    });
  }

  private handleMessage(message: Message): void {
    this.lastMessage = message;

    // 特殊处理通知消息
    if (message.type === 'notification') {
      const notification = message.data as NotificationMessage;
      this.notificationCallbacks.forEach((callback) => callback(notification));

      const handlers = this.handlers.get('notification');
      if (handlers) {
        handlers.forEach((handler) => handler(notification));
      }
      return;
    }

    // 使用节流器处理特定消息类型
    const throttledTypes = ['rank_update', 'bid_placed'];
    if (throttledTypes.includes(message.type)) {
      this.messageThrottlers.processMessage(message.type, message.data || message);
    } else {
      // 其他消息类型直接处理
      const handlers = this.handlers.get(message.type);
      if (handlers) {
        handlers.forEach((handler) => handler(message.data || message));
      }
    }
  }

  // 获取最后收到的消息
  getLastMessage(): Message | null {
    return this.lastMessage;
  }

  // 订阅通知
  onNotification(callback: (notification: NotificationMessage) => void): () => void {
    this.notificationCallbacks.add(callback);
    return () => {
      this.notificationCallbacks.delete(callback);
    };
  }

  private startPing(): void {
    this.pingInterval = setInterval(() => {
      this.send({ type: 'ping' });
    }, 30000);
  }

  private stopPing(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnect attempts reached');
      return;
    }

    const delay = this.reconnectDelays[this.reconnectAttempts] || 30;
    this.reconnectAttempts++;

    console.log(`Reconnecting in ${delay}s (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

    this.reconnectTimer = setTimeout(() => {
      this.connect().catch((error) => {
        console.error('Reconnect failed:', error);
      });
    }, delay * 1000);
  }

  private stopReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  // 重置重连计数（连接成功后调用）
  resetReconnectAttempts(): void {
    this.reconnectAttempts = 0;
    this.stopReconnect();
  }
}

export default WebSocketService;
