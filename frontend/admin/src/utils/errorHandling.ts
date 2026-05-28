// utils/errorHandling.ts - 通用错误处理工具函数

import { logError } from './errorMessages';

/**
 * 安全执行异步函数，捕获错误
 */
export async function safeAsync<T>(
  fn: () => Promise<T>,
  errorHandler?: (error: any) => void
): Promise<T | undefined> {
  try {
    return await fn();
  } catch (error) {
    if (errorHandler) {
      errorHandler(error);
    } else {
      logError(error, 'safeAsync');
    }
    return undefined;
  }
}

/**
 * 重试函数
 */
export async function retry<T>(
  fn: () => Promise<T>,
  options: {
    maxRetries?: number;
    delay?: number;
    shouldRetry?: (error: any) => boolean;
  } = {}
): Promise<T> {
  const { maxRetries = 3, delay = 1000, shouldRetry = () => true } = options;
  let lastError: any;

  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;

      // 判断是否应该重试
      if (!shouldRetry(error) || i === maxRetries - 1) {
        throw error;
      }

      // 等待一段时间后重试
      await new Promise(resolve => setTimeout(resolve, delay * (i + 1)));
    }
  }

  throw lastError;
}

/**
 * 防抖错误处理
 */
export function debounceError(
  fn: (error: any) => void,
  delay: number = 1000
): (error: any) => void {
  let lastErrorTime = 0;

  return (error: any) => {
    const now = Date.now();
    if (now - lastErrorTime >= delay) {
      fn(error);
      lastErrorTime = now;
    }
  };
}

/**
 * 批量处理错误
 */
export function batchErrorHandler(
  errors: any[],
  handler: (error: any, index: number) => void
): void {
  errors.forEach((error, index) => {
    if (error) {
      handler(error, index);
    }
  });
}

/**
 * 创建错误报告
 */
export function createErrorReport(error: any, context?: string): string {
  const timestamp = new Date().toISOString();
  const report = {
    timestamp,
    context,
    error: {
      name: error.name,
      message: error.message,
      stack: error.stack,
      status: error.status,
      code: error.code,
    },
    userAgent: navigator.userAgent,
    url: window.location.href,
  };

  return JSON.stringify(report, null, 2);
}

/**
 * 导出错误日志
 */
export function exportErrorLogs(): string {
  try {
    const logs = JSON.parse(localStorage.getItem('error_logs') || '[]');
    return JSON.stringify(logs, null, 2);
  } catch {
    return '[]';
  }
}

/**
 * 清除错误日志
 */
export function clearErrorLogs(): void {
  localStorage.removeItem('error_logs');
}

/**
 * 检查网络状态
 */
export function checkNetworkStatus(): {
  online: boolean;
  type?: string;
  downlink?: number;
} {
  const nav = navigator as any;

  return {
    online: navigator.onLine,
    type: nav.connection?.effectiveType,
    downlink: nav.connection?.downlink,
  };
}

/**
 * 监听网络状态变化
 */
export function onNetworkStatusChange(callback: (online: boolean) => void): () => void {
  const handleOnline = () => callback(true);
  const handleOffline = () => callback(false);

  window.addEventListener('online', handleOnline);
  window.addEventListener('offline', handleOffline);

  // 返回清理函数
  return () => {
    window.removeEventListener('online', handleOnline);
    window.removeEventListener('offline', handleOffline);
  };
}
