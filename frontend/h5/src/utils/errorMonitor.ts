// 错误监控服务

interface ErrorReport {
  timestamp: string;
  message: string;
  stack?: string;
  url: string;
  userAgent: string;
  userId?: number;
  role?: number;
  additionalData?: Record<string, any>;
}

class ErrorMonitor {
  private static instance: ErrorMonitor;
  private errorQueue: ErrorReport[] = [];
  private isReporting = false;
  private maxQueueSize = 10;
  private reportEndpoint = '/api/v1/errors/report';

  private constructor() {
    this.setupGlobalErrorHandler();
  }

  static getInstance(): ErrorMonitor {
    if (!ErrorMonitor.instance) {
      ErrorMonitor.instance = new ErrorMonitor();
    }
    return ErrorMonitor.instance;
  }

  /**
   * 设置全局错误处理器
   */
  private setupGlobalErrorHandler() {
    // 捕获JavaScript错误
    window.onerror = (message, source, lineno, colno, error) => {
      this.captureError({
        message: message.toString(),
        stack: error?.stack,
        url: window.location.href,
        userAgent: navigator.userAgent,
        additionalData: {
          source,
          lineno,
          colno,
        },
      });
    };

    // 捕获Promise未处理的rejection
    window.addEventListener('unhandledrejection', (event) => {
      this.captureError({
        message: `Unhandled Promise Rejection: ${event.reason}`,
        stack: event.reason?.stack,
        url: window.location.href,
        userAgent: navigator.userAgent,
      });
    });

    // 捕获资源加载错误
    window.addEventListener('error', (event) => {
      if (event.target !== window) {
        this.captureError({
          message: `Resource Load Error: ${(event.target as HTMLElement).tagName}`,
          url: window.location.href,
          userAgent: navigator.userAgent,
          additionalData: {
            src: (event.target as any).src || (event.target as any).href,
          },
        });
      }
    }, true);
  }

  /**
   * 捕获错误
   */
  captureError(error: Omit<ErrorReport, 'timestamp'>) {
    const errorReport: ErrorReport = {
      ...error,
      timestamp: new Date().toISOString(),
      userId: this.getCurrentUserId(),
      role: this.getCurrentUserRole(),
    };

    this.errorQueue.push(errorReport);

    // 如果队列满了，立即上报
    if (this.errorQueue.length >= this.maxQueueSize) {
      this.flush();
    } else {
      // 延迟上报
      this.scheduleReport();
    }
  }

  /**
   * 手动捕获错误
   */
  captureException(error: Error, additionalData?: Record<string, any>) {
    this.captureError({
      message: error.message,
      stack: error.stack,
      url: window.location.href,
      userAgent: navigator.userAgent,
      additionalData,
    });
  }

  /**
   * 捕获自定义消息
   */
  captureMessage(message: string, level: 'info' | 'warning' | 'error' = 'info') {
    this.captureError({
      message: `[${level.toUpperCase()}] ${message}`,
      url: window.location.href,
      userAgent: navigator.userAgent,
    });
  }

  /**
   * 设置用户信息
   */
  setUser(userId: number, role: number) {
    localStorage.setItem('error_monitor_user', JSON.stringify({ userId, role }));
  }

  /**
   * 清除用户信息
   */
  clearUser() {
    localStorage.removeItem('error_monitor_user');
  }

  /**
   * 获取当前用户ID
   */
  private getCurrentUserId(): number | undefined {
    try {
      const userStr = localStorage.getItem('auth_user');
      if (userStr) {
        const user = JSON.parse(userStr);
        return user.id;
      }
    } catch (e) {
      // 忽略解析错误
    }
    return undefined;
  }

  /**
   * 获取当前用户角色
   */
  private getCurrentUserRole(): number | undefined {
    try {
      const userStr = localStorage.getItem('auth_user');
      if (userStr) {
        const user = JSON.parse(userStr);
        return user.role;
      }
    } catch (e) {
      // 忽略解析错误
    }
    return undefined;
  }

  /**
   * 调度上报
   */
  private scheduleReport() {
    if (this.isReporting) return;

    setTimeout(() => {
      this.flush();
    }, 5000);
  }

  /**
   * 立即上报所有错误
   */
  async flush() {
    if (this.isReporting || this.errorQueue.length === 0) return;

    this.isReporting = true;

    try {
      const errors = [...this.errorQueue];
      this.errorQueue = [];

      // 尝试发送到服务器
      if (navigator.onLine) {
        await this.sendToServer(errors);
      } else {
        // 离线时保存到localStorage
        this.saveToLocalStorage(errors);
      }

      // 输出到控制台（开发环境）
      if (process.env.NODE_ENV === 'development') {
        console.group('🚨 Error Monitor Report');
        errors.forEach((error) => {
          console.error(error.message, error);
        });
        console.groupEnd();
      }
    } catch (error) {
      console.error('Failed to report errors:', error);
    } finally {
      this.isReporting = false;
    }
  }

  /**
   * 发送到服务器
   */
  private async sendToServer(errors: ErrorReport[]) {
    try {
      // token 为空时不附 Authorization 头，避免发送 "Bearer null" 触发 401
      const token = localStorage.getItem('auth_token') || localStorage.getItem('token');
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }

      const response = await fetch(this.reportEndpoint, {
        method: 'POST',
        headers,
        body: JSON.stringify({ errors }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
    } catch (error) {
      // 如果发送失败，保存到localStorage以便下次发送
      this.saveToLocalStorage(errors);
      throw error;
    }
  }

  /**
   * 保存到localStorage
   */
  private saveToLocalStorage(errors: ErrorReport[]) {
    try {
      const existingErrors = this.getStoredErrors();
      const allErrors = [...existingErrors, ...errors];

      // 只保留最近100条错误
      const recentErrors = allErrors.slice(-100);

      localStorage.setItem('error_monitor_queue', JSON.stringify(recentErrors));
    } catch (e) {
      // localStorage已满，清除旧错误
      localStorage.removeItem('error_monitor_queue');
    }
  }

  /**
   * 获取存储的错误
   */
  private getStoredErrors(): ErrorReport[] {
    try {
      const errorsStr = localStorage.getItem('error_monitor_queue');
      return errorsStr ? JSON.parse(errorsStr) : [];
    } catch (e) {
      return [];
    }
  }

  /**
   * 获取错误统计
   */
  getErrorStats() {
    const storedErrors = this.getStoredErrors();
    return {
      total: storedErrors.length,
      recent: storedErrors.slice(-10),
    };
  }

  /**
   * 清除所有错误
   */
  clearErrors() {
    localStorage.removeItem('error_monitor_queue');
    this.errorQueue = [];
  }
}

// 导出单例实例
export const errorMonitor = ErrorMonitor.getInstance();

// 便捷方法
export const captureException = (error: Error, additionalData?: Record<string, any>) => {
  errorMonitor.captureException(error, additionalData);
};

export const captureMessage = (message: string, level?: 'info' | 'warning' | 'error') => {
  errorMonitor.captureMessage(message, level);
};
