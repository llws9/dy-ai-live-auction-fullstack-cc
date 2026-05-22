// services/websocket.ts

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
  private handlers: Map<string, Set<MessageHandler>> = new Map();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private pingInterval: NodeJS.Timeout | null = null;
  private isConnecting = false;
  private isManualClose = false;
  private lastMessage: Message | null = null;
  private notificationCallbacks: Set<(notification: NotificationMessage) => void> = new Set();

  // 指数退避延迟序列（秒）
  private reconnectDelays = [1, 2, 4, 8, 16, 30, 30, 30, 30, 30];

  constructor(auctionId: number, token?: string) {
    this.auctionId = auctionId;
    this.token = token || null;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      if (this.isConnecting) {
        return;
      }

      this.isConnecting = true;
      this.isManualClose = false;

      // 构建WebSocket URL，添加token参数
      let wsUrl = `ws://${window.location.host}/api/v1/ws?auction_id=${this.auctionId}`;
      if (this.token) {
        wsUrl += `&token=${encodeURIComponent(this.token)}`;
      }

      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.isConnecting = false;
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
        reject(error);
      };

      this.ws.onclose = () => {
        console.log('WebSocket closed');
        this.isConnecting = false;
        this.stopPing();
        if (!this.isManualClose) {
          this.scheduleReconnect();
        }
      };
    });
  }

  disconnect(): void {
    this.isManualClose = true;
    this.stopPing();
    this.stopReconnect();
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
      this.notificationCallbacks.forEach((callback) => {
        callback(message.data as NotificationMessage);
      });
    }

    const handlers = this.handlers.get(message.type);
    if (handlers) {
      handlers.forEach((handler) => handler(message.data || message));
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
