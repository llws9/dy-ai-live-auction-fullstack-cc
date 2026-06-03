import { ENV } from '../utils/env';

const API_BASE_URL = ENV.API_BASE_URL;

interface ApiResponse<T = any> {
  code?: number;
  message?: string;
  data?: T;
  [key: string]: any;
}

async function parseResponse<T>(response: Response): Promise<ApiResponse<T>> {
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data?.message || '请求失败');
  }
  return data;
}

export async function startSkyLampSubscription(auctionId: number, token: string): Promise<ApiResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/sky-lamp/subscriptions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ auction_id: auctionId }),
  });

  return parseResponse(response);
}

export async function stopSkyLampSubscription(subscriptionId: number, token: string): Promise<ApiResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/sky-lamp/subscriptions/${subscriptionId}/stop`, {
    method: 'PUT',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  return parseResponse(response);
}

export async function getSkyLampSubscriptions(token: string, status?: number): Promise<ApiResponse> {
  const query = status !== undefined ? `?status=${status}` : '';
  const response = await fetch(`${API_BASE_URL}/api/v1/sky-lamp/subscriptions${query}`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  return parseResponse(response);
}
