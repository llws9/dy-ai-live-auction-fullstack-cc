/**
 * 埋点 SDK
 * 用于前端事件追踪，发送到后端 Prometheus 指标系统
 */

interface TrackEventParams {
  event_type: string;
  event_name?: string;
  params?: Record<string, string | number | boolean>;
  user_id?: string;
}

interface TrackingConfig {
  endpoint: string;
  debug?: boolean;
  batchSize?: number;
  flushInterval?: number;
}

class TrackingSDK {
  private endpoint: string;
  private debug: boolean;
  private queue: TrackEventParams[] = [];
  private batchSize: number;
  private flushInterval: number;
  private timer: NodeJS.Timeout | null = null;
  private userId: string = '';

  constructor(config: TrackingConfig) {
    this.endpoint = config.endpoint;
    this.debug = config.debug || false;
    this.batchSize = config.batchSize || 10;
    this.flushInterval = config.flushInterval || 5000;
    this.startFlushTimer();
  }

  /**
   * 设置用户ID
   */
  setUserId(userId: string) {
    this.userId = userId;
  }

  /**
   * 追踪事件
   */
  track(eventType: string, params?: Record<string, string | number | boolean>, eventName?: string) {
    const event: TrackEventParams = {
      event_type: eventType,
      event_name: eventName,
      params,
      user_id: this.userId,
      timestamp: Date.now(),
    };

    if (this.debug) {
      console.log('[Tracking]', eventType, params);
    }

    this.queue.push(event);

    // 如果达到批量大小，立即发送
    if (this.queue.length >= this.batchSize) {
      this.flush();
    }
  }

  /**
   * 直播间进入
   */
  trackLiveRoomEnter(roomId: string, userType: string = 'normal') {
    this.track('live_room_enter', {
      room_id: roomId,
      user_type: userType,
    });
  }

  /**
   * 直播间离开
   */
  trackLiveRoomLeave(roomId: string, duration: number) {
    this.track('live_room_leave', {
      room_id: roomId,
      duration_seconds: Math.floor(duration / 1000),
    });
  }

  /**
   * 竞拍浏览
   */
  trackAuctionView(auctionId: string, productId: string) {
    this.track('auction_view', {
      auction_id: auctionId,
      product_id: productId,
    });
  }

  /**
   * 出价按钮点击
   */
  trackBidClick(auctionId: string, currentPrice: number) {
    this.track('bid_click', {
      auction_id: auctionId,
      current_price: currentPrice,
    });
  }

  /**
   * 支付发起
   */
  trackPaymentStart(orderId: string, amount: number, method: string) {
    this.track('payment_start', {
      order_id: orderId,
      amount,
      method,
    });
  }

  /**
   * 用户注册
   */
  trackUserRegister(source: string = 'direct') {
    this.track('user_register', {
      source,
    });
  }

  /**
   * 用户登录
   */
  trackUserLogin(method: string = 'password') {
    this.track('user_login', {
      method,
    });
  }

  /**
   * 页面浏览
   */
  trackPageView(pageName: string) {
    this.track('page_view', {
      page: pageName,
      url: window.location.href,
      referrer: document.referrer,
    });
  }

  /**
   * 自定义事件
   */
  trackCustom(eventName: string, params?: Record<string, string | number | boolean>) {
    this.track('custom', params, eventName);
  }

  /**
   * 立即发送队列中的事件
   */
  async flush() {
    if (this.queue.length === 0) return;

    const events = [...this.queue];
    this.queue = [];

    try {
      // 发送批量事件
      for (const event of events) {
        await this.sendEvent(event);
      }
    } catch (error) {
      // 发送失败，重新加入队列
      console.error('[Tracking] Flush failed:', error);
      this.queue = [...events, ...this.queue];
    }
  }

  /**
   * 发送单个事件
   */
  private async sendEvent(event: TrackEventParams) {
    try {
      const response = await fetch(this.endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(event),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      if (this.debug) {
        console.log('[Tracking] Event sent:', event.event_type);
      }
    } catch (error) {
      console.error('[Tracking] Send failed:', error);
      throw error;
    }
  }

  /**
   * 启动定时刷新
   */
  private startFlushTimer() {
    this.timer = setInterval(() => {
      this.flush();
    }, this.flushInterval);
  }

  /**
   * 停止定时刷新
   */
  stop() {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
    this.flush();
  }
}

// 默认实例
let defaultTracker: TrackingSDK | null = null;

/**
 * 初始化埋点 SDK
 */
export function initTracking(config: TrackingConfig) {
  defaultTracker = new TrackingSDK(config);
  return defaultTracker;
}

/**
 * 获取埋点实例
 */
export function getTracker() {
  if (!defaultTracker) {
    throw new Error('Tracking not initialized. Call initTracking first.');
  }
  return defaultTracker;
}

export { TrackingSDK };
export type { TrackEventParams, TrackingConfig };
