// utils/errorMessages.ts - 错误消息映射

import { IS_DEV, IS_PROD } from '@/utils/env';

export interface ErrorMessage {
  title: string;
  message: string;
  action?: string;
}

export const ERROR_MESSAGES: Record<string | number, ErrorMessage> = {
  // HTTP 状态码错误
  400: {
    title: '请求错误',
    message: '请求参数有误，请检查输入内容',
  },
  401: {
    title: '登录已过期',
    message: '登录已过期，请重新登录',
    action: 'redirect_login',
  },
  403: {
    title: '权限不足',
    message: '您没有权限访问此功能',
  },
  404: {
    title: '资源不存在',
    message: '请求的资源不存在',
  },
  408: {
    title: '请求超时',
    message: '请求超时，请稍后重试',
  },
  409: {
    title: '资源冲突',
    message: '该资源已被占用，请刷新后重试',
  },
  422: {
    title: '参数验证失败',
    message: '提交的数据格式不正确',
  },
  429: {
    title: '请求过于频繁',
    message: '您的操作过于频繁，请稍后再试',
  },
  500: {
    title: '服务器错误',
    message: '服务器繁忙，请稍后重试',
  },
  502: {
    title: '网关错误',
    message: '服务暂时不可用，请稍后重试',
  },
  503: {
    title: '服务不可用',
    message: '服务正在维护，请稍后重试',
  },
  504: {
    title: '网关超时',
    message: '请求超时，请稍后重试',
  },

  // 网络错误
  NETWORK_ERROR: {
    title: '网络连接失败',
    message: '网络连接失败，请检查网络设置',
  },
  TIMEOUT_ERROR: {
    title: '请求超时',
    message: '请求超时，请稍后重试',
  },
  UNKNOWN_ERROR: {
    title: '未知错误',
    message: '发生未知错误，请稍后重试',
  },

  // 业务错误码
  PRODUCT_NOT_FOUND: {
    title: '商品不存在',
    message: '该商品不存在或已下架',
  },
  AUCTION_NOT_FOUND: {
    title: '竞拍不存在',
    message: '该竞拍活动不存在或已结束',
  },
  AUCTION_ENDED: {
    title: '竞拍已结束',
    message: '该竞拍活动已结束',
  },
  AUCTION_NOT_STARTED: {
    title: '竞拍未开始',
    message: '该竞拍活动尚未开始',
  },
  BID_TOO_LOW: {
    title: '出价过低',
    message: '您的出价低于当前最高价',
  },
  INSUFFICIENT_BALANCE: {
    title: '余额不足',
    message: '您的账户余额不足',
  },
  ORDER_NOT_FOUND: {
    title: '订单不存在',
    message: '该订单不存在或已被删除',
  },
  ORDER_ALREADY_PAID: {
    title: '订单已支付',
    message: '该订单已完成支付',
  },
  WEBSOCKET_ERROR: {
    title: '连接异常',
    message: '实时连接异常，请刷新页面重试',
  },
};

/**
 * 根据错误信息获取用户友好的错误消息
 */
export function getErrorMessage(error: any): ErrorMessage {
  // 网络错误
  if (!navigator.onLine) {
    return ERROR_MESSAGES.NETWORK_ERROR;
  }

  // 请求超时
  if (error.name === 'AbortError' || error.message?.includes('timeout')) {
    return ERROR_MESSAGES.TIMEOUT_ERROR;
  }

  // HTTP 状态码错误
  if (error.status || error.response?.status) {
    const status = error.status || error.response?.status;
    return ERROR_MESSAGES[status] || ERROR_MESSAGES.UNKNOWN_ERROR;
  }

  // 后端返回的业务错误
  if (error.code) {
    return ERROR_MESSAGES[error.code] || {
      title: '操作失败',
      message: error.message || error.msg || '操作失败，请稍后重试',
    };
  }

  // 默认错误消息
  if (error.message) {
    return {
      title: '操作失败',
      message: error.message,
    };
  }

  return ERROR_MESSAGES.UNKNOWN_ERROR;
}

/**
 * 格式化错误消息为字符串
 */
export function formatErrorMessage(error: any): string {
  const errorMsg = getErrorMessage(error);
  return errorMsg.message;
}

/**
 * 判断是否需要跳转登录页
 */
export function shouldRedirectToLogin(error: any): boolean {
  const errorMsg = getErrorMessage(error);
  return errorMsg.action === 'redirect_login';
}

/**
 * 错误日志记录
 */
export function logError(error: any, context?: string): void {
  const timestamp = new Date().toISOString();
  const errorInfo = {
    timestamp,
    context,
    error: {
      message: error.message,
      stack: error.stack,
      status: error.status || error.response?.status,
      code: error.code,
      data: error.data,
    },
  };

  // 开发环境打印错误
  if (IS_DEV) {
    console.error('[Error Log]', errorInfo);
  }

  // 生产环境可以上报到错误监控系统
  // 如：Sentry, BugSnag 等
  if (IS_PROD) {
    // TODO: 集成错误上报服务
    // Sentry.captureException(error, { extra: errorInfo });
  }

  // 存储到本地用于排查
  try {
    const errorLogs = JSON.parse(localStorage.getItem('error_logs') || '[]');
    errorLogs.unshift(errorInfo);
    // 只保留最近20条错误
    if (errorLogs.length > 20) {
      errorLogs.pop();
    }
    localStorage.setItem('error_logs', JSON.stringify(errorLogs));
  } catch (e) {
    // 忽略存储错误
  }
}
