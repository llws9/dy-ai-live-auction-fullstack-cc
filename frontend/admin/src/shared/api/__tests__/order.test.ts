import { orderApi } from '../index';

describe('orderApi buyer text normalization', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  it('repairs mojibake buyer names in admin order list', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
      },
      json: async () => ({
        code: 0,
        data: {
          list: [
            {
              id: 101,
              user_id: 9101,
              user_name: 'æ¼”ç¤ºä¹°å®¶A',
              final_price: 1200,
              status: 1,
              created_at: '2026-06-08T00:06:00Z',
            },
          ],
          total: 1,
          page: 1,
          page_size: 20,
        },
      }),
    });

    const result = await orderApi.list({ page: 1, page_size: 20 });

    expect(result.list[0].user_name).toBe('演示买家A');
  });

  it('repairs mojibake buyer names in admin order detail', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
      },
      json: async () => ({
        code: 0,
        data: {
          id: 101,
          user_id: 9101,
          user_name: 'æ¼”ç¤ºä¹°å®¶A',
          final_price: 1200,
          status: 1,
          created_at: '2026-06-08T00:06:00Z',
        },
      }),
    });

    const result = await orderApi.get(101);

    expect(result.user_name).toBe('演示买家A');
  });
});
