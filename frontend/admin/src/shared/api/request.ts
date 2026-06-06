// API请求基础封装

import { ApiResponse } from './types';

const API_BASE_URL = '/api/v1';
const REQUEST_TIMEOUT = 30000;

// 自定义错误类
export class ApiError extends Error {
  status: number;
  code?: number;
  data?: any;

  constructor(message: string, status: number, code?: number, data?: any) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.data = data;
  }
}

// Toast提示函数类型
type ToastFunction = (message: string, type?: 'success' | 'error' | 'warning' | 'info') => void;
let toastFunction: ToastFunction | null = null;

export function setToastFunction(fn: ToastFunction) {
  toastFunction = fn;
}

// 显示错误提示
function showErrorToast(error: any) {
  if (toastFunction) {
    toastFunction(error.message, 'error');
  } else {
    console.error('API Error:', error.message);
  }
}

// 默认错误消息
function getDefaultMessage(status: number): string {
  const messages: Record<number, string> = {
    400: '请求参数有误',
    401: '登录已过期，请重新登录',
    403: '您没有权限访问此功能',
    404: '请求的资源不存在',
    408: '请求超时，请稍后重试',
    409: '资源冲突，请刷新后重试',
    422: '数据验证失败',
    429: '请求过于频繁，请稍后再试',
    500: '服务器繁忙，请稍后重试',
    502: '服务暂时不可用',
    503: '服务正在维护',
    504: '请求超时，请稍后重试',
  };
  return messages[status] || '请求失败，请稍后重试';
}

function isSuccessBusinessCode(code: number): boolean {
  return code === 0 || (code >= 200 && code < 300);
}

// 处理错误响应
async function handleErrorResponse(response: Response): Promise<never> {
  let errorData: any = {};

  try {
    errorData = await response.json();
  } catch (e) {
    // JSON解析失败
  }

  const error = new ApiError(
    errorData.message || errorData.msg || getDefaultMessage(response.status),
    response.status,
    errorData.code,
    errorData.data
  );

  // 401未授权
  if (response.status === 401) {
    localStorage.removeItem('admin_auth_token');
    localStorage.removeItem('admin_auth_user');
    localStorage.removeItem('token');
    localStorage.removeItem('userInfo');

    if (window.location.hash !== '#/admin-login') {
      window.location.hash = '/admin-login';
    }

    throw error;
  }

  throw error;
}

// 带超时的fetch
async function fetchWithTimeout(url: string, options: RequestInit, timeout: number): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const response = await fetch(url, {
      ...options,
      signal: controller.signal,
    });
    return response;
  } finally {
    clearTimeout(timeoutId);
  }
}

// 通用请求方法
async function request<T>(
  path: string,
  options?: RequestInit,
  config?: {
    showError?: boolean;
    timeout?: number;
  }
): Promise<T> {
  const { showError = true, timeout = REQUEST_TIMEOUT } = config || {};

  const token = localStorage.getItem('admin_auth_token');
  const url = `${API_BASE_URL}${path}`;

  try {
    const response = await fetchWithTimeout(
      url,
      {
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
          ...options?.headers,
        },
        ...options,
      },
      timeout
    );

    if (!response.ok) {
      await handleErrorResponse(response);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      const data: ApiResponse<T> = await response.json();

      // 检查业务错误码
      if (typeof data.code === 'number' && !isSuccessBusinessCode(data.code)) {
        const error = new ApiError(
          data.message || data.msg || '操作失败',
          response.status,
          data.code,
          data.data
        );

        if (showError) {
          showErrorToast(error);
        }

        throw error;
      }

      if (Object.prototype.hasOwnProperty.call(data, 'data')) {
        return data.data;
      }

      return data as unknown as T;
    }

    return response as any;
  } catch (error: any) {
    // AbortError - 请求超时
    if (error.name === 'AbortError') {
      const timeoutError = new ApiError('请求超时，请稍后重试', 408, 0);
      if (showError) {
        showErrorToast(timeoutError);
      }
      throw timeoutError;
    }

    // 网络错误
    if (error.name === 'TypeError' && error.message === 'Failed to fetch') {
      const networkError = new ApiError('网络连接失败，请检查网络设置', 0, 0);
      if (showError) {
        showErrorToast(networkError);
      }
      throw networkError;
    }

    // 已经是ApiError
    if (error instanceof ApiError) {
      if (showError && error.status !== 401) {
        showErrorToast(error);
      }
      throw error;
    }

    // 其他错误
    if (showError) {
      showErrorToast(error);
    }
    throw error;
  }
}

// GET请求
export function get<T>(path: string, config?: { showError?: boolean; timeout?: number }): Promise<T> {
  return request<T>(path, { method: 'GET' }, config);
}

// POST请求
export function post<T>(path: string, data?: any, config?: { showError?: boolean; timeout?: number }): Promise<T> {
  return request<T>(path, {
    method: 'POST',
    body: data ? JSON.stringify(data) : undefined,
  }, config);
}

// PUT请求
export function put<T>(path: string, data?: any, config?: { showError?: boolean; timeout?: number }): Promise<T> {
  return request<T>(path, {
    method: 'PUT',
    body: data ? JSON.stringify(data) : undefined,
  }, config);
}

// DELETE请求
export function del<T>(path: string, config?: { showError?: boolean; timeout?: number }): Promise<T> {
  return request<T>(path, { method: 'DELETE' }, config);
}

// 构建查询参数
export function buildQuery(params: Record<string, any>): string {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      query.set(key, String(value));
    }
  });
  return query.toString();
}
