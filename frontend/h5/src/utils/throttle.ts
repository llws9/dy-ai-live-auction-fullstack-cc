// utils/throttle.ts

/**
 * 节流函数 - 在指定时间窗口内只执行最后一次调用
 */
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle = false;
  let lastArgs: Parameters<T> | null = null;

  return function (this: any, ...args: Parameters<T>) {
    if (!inThrottle) {
      func.apply(this, args);
      inThrottle = true;
      setTimeout(() => {
        inThrottle = false;
        if (lastArgs) {
          func.apply(this, lastArgs);
          lastArgs = null;
        }
      }, limit);
    } else {
      lastArgs = args;
    }
  };
}

/**
 * 消息队列节流器 - 在指定时间窗口内只处理最新消息
 */
export class MessageThrottler<T = any> {
  private queue: T[] = [];
  private timer: NodeJS.Timeout | null = null;
  private processing = false;
  private lastProcessTime = 0;

  constructor(
    private handler: (message: T) => void,
    private windowMs: number = 200
  ) {}

  /**
   * 添加消息到队列
   */
  add(message: T): void {
    this.queue.push(message);
    this.scheduleProcess();
  }

  /**
   * 立即处理队列中的最新消息
   */
  flush(): void {
    if (this.queue.length === 0) return;

    const now = Date.now();
    if (now - this.lastProcessTime >= this.windowMs) {
      this.processLatest();
    } else {
      this.scheduleProcess();
    }
  }

  /**
   * 清空队列
   */
  clear(): void {
    this.queue = [];
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }

  /**
   * 获取队列长度
   */
  getQueueLength(): number {
    return this.queue.length;
  }

  private scheduleProcess(): void {
    if (this.processing) return;

    const now = Date.now();
    const timeSinceLastProcess = now - this.lastProcessTime;
    const delay = Math.max(0, this.windowMs - timeSinceLastProcess);

    if (delay === 0) {
      this.processLatest();
    } else {
      this.timer = setTimeout(() => {
        this.processLatest();
      }, delay);
    }
  }

  private processLatest(): void {
    if (this.queue.length === 0) return;

    this.processing = true;
    const latest = this.queue[this.queue.length - 1];
    this.queue = [];

    this.lastProcessTime = Date.now();
    this.handler(latest);

    this.processing = false;

    // 如果队列中还有新消息,继续调度处理
    if (this.queue.length > 0) {
      this.scheduleProcess();
    }
  }
}

/**
 * 创建消息类型节流器集合
 */
export class MessageTypeThrottlers {
  private throttlers: Map<string, MessageThrottler> = new Map();
  private defaultWindowMs: number;

  constructor(defaultWindowMs: number = 200) {
    this.defaultWindowMs = defaultWindowMs;
  }

  /**
   * 为特定消息类型创建节流器
   */
  createThrottler(
    messageType: string,
    handler: (data: any) => void,
    windowMs?: number
  ): void {
    this.throttlers.set(
      messageType,
      new MessageThrottler(handler, windowMs ?? this.defaultWindowMs)
    );
  }

  /**
   * 处理消息
   */
  processMessage(messageType: string, data: any): void {
    const throttler = this.throttlers.get(messageType);
    if (throttler) {
      throttler.add(data);
    }
  }

  /**
   * 清空所有节流器
   */
  clearAll(): void {
    this.throttlers.forEach((throttler) => throttler.clear());
  }

  /**
   * 获取节流器
   */
  getThrottler(messageType: string): MessageThrottler | undefined {
    return this.throttlers.get(messageType);
  }
}
