jest.mock('../../utils/errorMessages', () => ({
  getErrorMessage: (error: Error) => ({ message: error.message || '请求失败' }),
  logError: jest.fn(),
}));

import { auctionApi, buildLoginRedirectPath, orderApi, userApi } from '../api';

describe('api service auth header', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.restoreAllMocks();
  });

  it('uses authContext token key for authenticated requests', async () => {
    localStorage.setItem('auth_token', 'auth-token-1');
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: () => 'application/json',
      },
      json: async () => ({
        code: 200,
        data: { id: 9, name: '林见山' },
      }),
    } as Response);
    global.fetch = fetchMock;

    await userApi.getProfile();

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/user/profile',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer auth-token-1',
        }),
      })
    );
  });

  it('requests documented user auction history endpoint with pagination', async () => {
    localStorage.setItem('auth_token', 'auth-token-1');
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: () => 'application/json',
      },
      json: async () => ({
        code: 200,
        data: {
          list: [],
          total: 0,
        },
      }),
    } as Response);
    global.fetch = fetchMock;

    await orderApi.history({ page: 2, page_size: 10 });

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/orders/history?page=2&page_size=10',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          Authorization: 'Bearer auth-token-1',
        }),
      })
    );
  });

  it('builds the login path with the current page as redirect target', () => {
    window.history.pushState({}, '', '/profile?from=history');

    expect(buildLoginRedirectPath()).toBe('/login?redirect=%2Fprofile%3Ffrom%3Dhistory');
  });

  it('passes category_id to /auctions list query when provided (T2.10)', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: async () => ({ code: 200, data: { list: [], total: 0 } }),
    } as Response);
    global.fetch = fetchMock;

    await auctionApi.list({ page: 1, page_size: 20, category_id: 12 });

    const calledUrl = fetchMock.mock.calls[0][0] as string;
    expect(calledUrl).toContain('/api/v1/auctions');
    expect(calledUrl).toContain('category_id=12');
  });

  it('omits category_id from /auctions query when not provided (T2.10)', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: async () => ({ code: 200, data: { list: [], total: 0 } }),
    } as Response);
    global.fetch = fetchMock;

    await auctionApi.list({ page: 1, page_size: 20 });

    const calledUrl = fetchMock.mock.calls[0][0] as string;
    expect(calledUrl).not.toContain('category_id');
  });
});
