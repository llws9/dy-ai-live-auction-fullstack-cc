// services/api.ts

import { Product, AuctionRule, PaginatedResponse } from '../types';
import { getErrorMessage, logError } from '../utils/errorMessages';

const API_BASE_URL = '/api/v1';

// 请求配置
const REQUEST_TIMEOUT = 30000; // 30秒超时

// 自定义错误类
export class ApiError extends Error {
  status: number;
  code?: string;
  data?: any;

  constructor(message: string, status: number, code?: string, data?: any) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.data = data;
  }
}

// Toast 提示函数类型
type ToastFunction = (message: string, type?: 'success' | 'error' | 'warning' | 'info') => void;
let toastFunction: ToastFunction | null = null;

// 设置 Toast 函数
export function setToastFunction(fn: ToastFunction) {
  toastFunction = fn;
}

// 显示错误提示
function showErrorToast(error: any) {
  const errorMsg = getErrorMessage(error);
  if (toastFunction) {
    toastFunction(errorMsg.message, 'error');
  } else {
    // 降级使用 alert
    alert(errorMsg.message);
  }
}

// 处理错误响应
async function handleErrorResponse(response: Response): Promise<never> {
  let errorData: any = {};

  try {
    errorData = await response.json();
  } catch (e) {
    // JSON 解析失败
  }

  const error = new ApiError(
    errorData.message || errorData.msg || getDefaultMessage(response.status),
    response.status,
    errorData.code,
    errorData.data
  );

  // 记录错误日志
  logError(error, `API Response Error: ${response.url}`);

  // 特殊处理：401 未授权
  if (response.status === 401) {
    // 清除本地存储的认证信息
    localStorage.removeItem('token');
    localStorage.removeItem('userInfo');

    // 跳转到登录页
    const loginPath = '/login';
    if (window.location.pathname !== loginPath) {
      window.location.href = loginPath;
    }

    throw error;
  }

  throw error;
}

// 获取默认错误消息
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

// 带超时的 fetch
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
    showError?: boolean;  // 是否显示错误提示
    timeout?: number;     // 自定义超时时间
  }
): Promise<T> {
  const { showError = true, timeout = REQUEST_TIMEOUT } = config || {};

  // 获取 token
  const token = localStorage.getItem('token');

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

    // 检查响应内容
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      const data = await response.json();

      // 检查业务错误码
      if (data.code && data.code !== 0 && data.code !== 200) {
        const error = new ApiError(
          data.message || data.msg || '操作失败',
          response.status,
          data.code,
          data.data
        );
        logError(error, `API Business Error: ${path}`);

        if (showError) {
          showErrorToast(error);
        }

        throw error;
      }

      return data.data || data;
    }

    return response as any;
  } catch (error: any) {
    // AbortError - 请求超时
    if (error.name === 'AbortError') {
      const timeoutError = new ApiError('请求超时，请稍后重试', 408, 'TIMEOUT');
      logError(timeoutError, `API Timeout: ${path}`);

      if (showError) {
        showErrorToast(timeoutError);
      }

      throw timeoutError;
    }

    // 网络错误
    if (error.name === 'TypeError' && error.message === 'Failed to fetch') {
      const networkError = new ApiError(
        '网络连接失败，请检查网络设置',
        0,
        'NETWORK_ERROR'
      );
      logError(networkError, `API Network Error: ${path}`);

      if (showError) {
        showErrorToast(networkError);
      }

      throw networkError;
    }

    // 已经是 ApiError
    if (error instanceof ApiError) {
      if (showError && error.status !== 401) { // 401 已经特殊处理，不需要再显示 toast
        showErrorToast(error);
      }
      throw error;
    }

    // 其他错误
    logError(error, `API Unknown Error: ${path}`);

    if (showError) {
      showErrorToast(error);
    }

    throw error;
  }
}

// GET 请求
export function get<T>(path: string, config?: { showError?: boolean; timeout?: number }): Promise<T> {
  return request<T>(path, { method: 'GET' }, config);
}

// POST 请求
export function post<T>(path: string, data?: any, config?: { showError?: boolean; timeout?: number }): Promise<T> {
  return request<T>(path, {
    method: 'POST',
    body: data ? JSON.stringify(data) : undefined,
  }, config);
}

// PUT 请求
export function put<T>(path: string, data?: any, config?: { showError?: boolean; timeout?: number }): Promise<T> {
  return request<T>(path, {
    method: 'PUT',
    body: data ? JSON.stringify(data) : undefined,
  }, config);
}

// DELETE 请求
export function del<T>(path: string, config?: { showError?: boolean; timeout?: number }): Promise<T> {
  return request<T>(path, { method: 'DELETE' }, config);
}

// 商品 API
export const productApi = {
  // 获取商品列表
  list: (params?: { status?: number; page?: number; page_size?: number }) => {
    const query = new URLSearchParams();
    if (params?.status !== undefined) query.set('status', String(params.status));
    if (params?.page) query.set('page', String(params.page));
    if (params?.page_size) query.set('page_size', String(params.page_size));

    return get<PaginatedResponse<Product>>(`/products?${query.toString()}`);
  },

  // 获取商品详情
  get: (id: number) => get<Product>(`/products/${id}`),

  // 创建商品
  create: (data: Partial<Product>) => post<Product>('/products', data),

  // 更新商品
  update: (id: number, data: Partial<Product>) => put<Product>(`/products/${id}`, data),

  // 删除商品
  delete: (id: number) => del<void>(`/products/${id}`),
};

// 竞拍规则 API
export const ruleApi = {
  // 获取规则
  get: (productId: number) => get<AuctionRule>(`/products/${productId}/rules`),

  // 创建规则
  create: (productId: number, data: Partial<AuctionRule>) =>
    post<AuctionRule>(`/products/${productId}/rules`, data),
};

// 竞拍 API
export const auctionApi = {
  // 获取竞拍详情
  get: (id: number) => get<any>(`/auctions/${id}`),

  // 获取出价记录
  getBids: (id: number) => get<any>(`/auctions/${id}/bids`),

  // 取消竞拍
  cancel: (id: number) => put<void>(`/auctions/${id}/cancel`),

  // 获取竞拍结果
  getResult: (id: number) => get<any>(`/auctions/${id}/result`),
};

// 订单 API
export const orderApi = {
  // 获取订单列表
  list: () => get<any>('/orders'),

  // 模拟支付
  pay: (id: number) => post<any>(`/orders/${id}/pay`),
};
