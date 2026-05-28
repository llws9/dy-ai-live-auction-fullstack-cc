// services/skyLamp.ts

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

// 开启天灯
export async function activateSkyLamp(auctionId: number, token: string) {
  const response = await fetch(`${API_BASE_URL}/api/v1/auctions/${auctionId}/sky-lamp`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  });
  return response.json();
}

// 取消天灯
export async function cancelSkyLamp(auctionId: number, token: string) {
  const response = await fetch(`${API_BASE_URL}/api/v1/auctions/${auctionId}/sky-lamp`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });
  return response.json();
}

// 查询天灯状态
export async function getSkyLampStatus(auctionId: number, token: string) {
  const response = await fetch(`${API_BASE_URL}/api/v1/auctions/${auctionId}/sky-lamp`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });
  return response.json();
}