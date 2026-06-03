// services/api.ts

import { getErrorMessage, logError } from '../utils/errorMessages';

const API_BASE_URL = '/api/v1';

// 请求配置
const REQUEST_TIMEOUT = 30000; // 30秒超时

// 业务成功码 SSOT：与后端 handler 中 c.JSON(200, {"code": 200, ...}) 对齐；
// 历史代码也使用 0 表示成功，过渡期同时兼容。
export const SUCCESS_CODES: ReadonlySet<number> = new Set([0, 200]);

export function getStoredToken(): string | null {
  return localStorage.getItem('auth_token') || localStorage.getItem('token');
}

function clearStoredAuth() {
  localStorage.removeItem('auth_token');
  localStorage.removeItem('auth_user');
  localStorage.removeItem('token');
  localStorage.removeItem('userInfo');
}

export function buildLoginRedirectPath() {
  const currentPath = `${window.location.pathname}${window.location.search}`;
  if (window.location.pathname === '/login') {
    return '/login';
  }

  return `/login?redirect=${encodeURIComponent(currentPath)}`;
}

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
    // 降级使用 alert（H5环境更适合移动端）
    console.error('Error:', errorMsg.message);
    // 可以使用移动端的原生提示
    if (typeof window !== 'undefined' && (window as any).WeixinJSBridge) {
      // 微信环境
      (window as any).WeixinJSBridge.invoke('showToast', {
        message: errorMsg.message,
      });
    }
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
    clearStoredAuth();

    const loginPath = buildLoginRedirectPath();
    if (window.location.pathname !== '/login') {
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

type RequestConfig = {
  showError?: boolean;
  timeout?: number;
  headers?: HeadersInit;
};

// 通用请求方法
async function request<T>(
  path: string,
  options?: RequestInit,
  config?: RequestConfig
): Promise<T> {
  const { showError = true, timeout = REQUEST_TIMEOUT } = config || {};

  // 获取 token
  const token = getStoredToken();

  const url = `${API_BASE_URL}${path}`;

  try {
    const response = await fetchWithTimeout(
      url,
      {
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
          ...config?.headers,
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
      if (data.code !== undefined && !SUCCESS_CODES.has(data.code)) {
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
export function get<T>(path: string, config?: RequestConfig): Promise<T> {
  return request<T>(path, { method: 'GET' }, config);
}

// POST 请求
export function post<T>(path: string, data?: any, config?: RequestConfig): Promise<T> {
  return request<T>(path, {
    method: 'POST',
    body: data ? JSON.stringify(data) : undefined,
  }, config);
}

// PUT 请求
export function put<T>(path: string, data?: any, config?: RequestConfig): Promise<T> {
  return request<T>(path, {
    method: 'PUT',
    body: data ? JSON.stringify(data) : undefined,
  }, config);
}

// DELETE 请求
export function del<T>(path: string, config?: RequestConfig): Promise<T> {
  return request<T>(path, { method: 'DELETE' }, config);
}

// 用户相关 API
export const userApi = {
  // 获取用户信息
  getProfile: () => get<any>('/user/profile'),

  // 更新用户信息
  updateProfile: (data: any) => put<any>('/user/profile', data),

  // 获取余额（T3.1 F-A2，返回 { available_amount, frozen_amount, currency }）
  getBalance: () => get<any>('/user/balance'),

  // 获取个人统计（T2.7 F-A1，返回 { following_count, auction_history_count, won_count }）
  getStats: () => get<any>('/users/me/stats'),
};

// 收货地址 API（T3.2 F-A3）
export const addressApi = {
  list: () => get<any>('/users/me/addresses'),
  create: (data: any) => post<any>('/users/me/addresses', data),
  update: (id: number, data: any) => put<any>(`/users/me/addresses/${id}`, data),
  remove: (id: number) => del<any>(`/users/me/addresses/${id}`),
  setDefault: (id: number) => post<any>(`/users/me/addresses/${id}/default`),
};

// 商品 API
export const productApi = {
  // 获取商品列表
  list: (params?: { status?: number; page?: number; page_size?: number }) => {
    const query = new URLSearchParams();
    if (params?.status !== undefined) query.set('status', String(params.status));
    if (params?.page) query.set('page', String(params.page));
    if (params?.page_size) query.set('page_size', String(params.page_size));

    return get<any>(`/products?${query.toString()}`);
  },

  // 获取商品详情
  get: (id: number) => get<any>(`/products/${id}`),

  // 获取分类列表（公开接口，T2.10 Home tabs 数据源）
  listCategories: () => get<any>(`/categories`),
};

// 竞拍 API
export const auctionApi = {
  // 获取竞拍列表
  list: (params?: { status?: string; page?: number; page_size?: number; category_id?: number }) => {
    const query = new URLSearchParams();
    if (params?.status) query.set('status', params.status);
    if (params?.page) query.set('page', String(params.page));
    if (params?.page_size) query.set('page_size', String(params.page_size));
    if (params?.category_id) query.set('category_id', String(params.category_id));

    return get<any>(`/auctions?${query.toString()}`);
  },

  // 获取竞拍详情
  get: (id: number) => get<any>(`/auctions/${id}`),

  // 获取出价记录
  getBids: (id: number) => get<any>(`/auctions/${id}/bids`),

  // 出价
  bid: (id: number, amount: number) => post<any>(`/auctions/${id}/bid`, { amount }),

  // 获取竞拍结果
  getResult: (id: number) => get<any>(`/auctions/${id}/result`),
};

// 订单 API
export const orderApi = {
  // 获取订单列表
  list: (params?: { page?: number; page_size?: number }) => {
    const query = new URLSearchParams();
    query.set('page', String(params?.page ?? 1));
    query.set('page_size', String(params?.page_size ?? 20));

    return get<any>(`/orders?${query.toString()}`);
  },

  // 获取用户竞拍历史
  history: (params?: { page?: number; page_size?: number }) => {
    const query = new URLSearchParams();
    query.set('page', String(params?.page ?? 1));
    query.set('page_size', String(params?.page_size ?? 20));

    return get<any>(`/orders/history?${query.toString()}`);
  },

  // 获取订单详情
  get: (id: number) => get<any>(`/orders/${id}`),

  // 创建订单
  create: (data: any) => post<any>('/orders', data),

  // 支付订单
  pay: (id: number) => post<any>(`/orders/${id}/pay`),

  // 取消订单
  cancel: (id: number) => put<any>(`/orders/${id}/cancel`),
};

// 出价 API
export const bidApi = {
  // 用户出价
  placeBid: (auctionId: number, amount: number) => {
    return post<any>(`/auctions/${auctionId}/bids`, { amount });
  },

  // 获取竞拍排名
  getRanking: (auctionId: number, limit: number = 10) => {
    return get<any>(`/auctions/${auctionId}/ranking?limit=${limit}`);
  },
};

// 关注 API
export const followApi = {
  // 关注直播间
  followLiveStream: (liveStreamId: number) => {
    return post<any>(`/live-streams/${liveStreamId}/follow`);
  },

  // 取消关注直播间
  unfollowLiveStream: (liveStreamId: number) => {
    return del<any>(`/live-streams/${liveStreamId}/follow`);
  },

  // 获取用户关注的直播间列表
  getFollowedLiveStreams: (page: number = 1, pageSize: number = 20) => {
    return get<any>(`/user/followed-live-streams?page=${page}&page_size=${pageSize}`);
  },

  // 获取直播间关注统计
  getFollowersStats: (liveStreamId: number) => {
    return get<any>(`/live-streams/${liveStreamId}/followers/stats`);
  },

  // 获取当前登录用户对指定直播间的关注状态（后端权威）
  getFollowStatus: (liveStreamId: number) => {
    return get<{ is_following: boolean }>(`/live-streams/${liveStreamId}/follow-status`);
  },
};

// 直播间 API
export const liveStreamApi = {
  // 获取直播间列表
  list: (page: number = 1, pageSize: number = 20) => {
    return get<any>(`/live-streams?page=${page}&page_size=${pageSize}`);
  },

  // 获取直播间详情
  get: (liveStreamId: number) => {
    return get<any>(`/live-streams/${liveStreamId}`);
  },
};
