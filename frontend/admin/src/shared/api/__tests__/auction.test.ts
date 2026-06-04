import { auctionApi } from '../index';

describe('auctionApi.list', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  it('repairs mojibake auction product and live stream text before returning the list', async () => {
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
              id: 6,
              product: {
                id: 1,
                name: 'ç¨€æœ‰ç å®',
                description: 'ç²¾é€‰æ‹å“',
              },
              live_stream_id: 2,
              live_stream_name: 'ç¿¡ç¿ ä¸“åœº',
              status: 1,
              current_price: 8800,
              bid_count: 0,
              start_time: '2026-06-03T03:18:39Z',
            },
          ],
          total: 1,
        },
      }),
    });

    const result = await auctionApi.list({ page: 1, page_size: 20 });

    expect(result.list[0].product).toMatchObject({
      name: '稀有珠宝',
      description: '精选拍品',
    });
    expect(result.list[0].live_stream_name).toBe('翡翠专场');
  });
});
