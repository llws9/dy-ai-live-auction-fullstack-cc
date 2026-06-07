import { auctionApi } from '../index';
import { post } from '../request';

jest.mock('../request', () => {
  const actual = jest.requireActual('../request');
  return {
    ...actual,
    post: jest.fn(),
  };
});

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

  it('creates auction with duration contract', async () => {
    (post as jest.Mock).mockResolvedValue({ id: 7001, product_id: 501, status: 0 });

    await auctionApi.create({ product_id: 501, duration: 3600 });

    expect(post).toHaveBeenCalledWith('/auctions', { product_id: 501, duration: 3600 });
  });

  it('creates scheduled auction with start_time contract', async () => {
    (post as jest.Mock).mockResolvedValue({ id: 1, product_id: 501, title: 'Demo' });

    await auctionApi.create({ product_id: 501, duration: 3600, start_time: '2026-06-08T10:30:00.000Z' });

    expect(post).toHaveBeenCalledWith('/auctions', {
      product_id: 501,
      duration: 3600,
      start_time: '2026-06-08T10:30:00.000Z',
    });
  });

  it('normalizes bid amount and buyer names before auction detail renders them', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
      },
      json: async () => ([
        {
          id: 11,
          auction_id: 7001,
          user_id: 9101,
          amount: '1200.00',
          created_at: '2026-06-08T00:06:00Z',
        },
        {
          id: 12,
          auction_id: 7001,
          user_id: 9102,
          user_name: 'æ¼”ç¤ºä¹°å®¶B',
          amount: 1300,
          created_at: '2026-06-08T00:06:10Z',
        },
      ]),
    });

    const result = await auctionApi.getBids(7001);

    expect(result).toEqual([
      expect.objectContaining({
        user_id: 9101,
        user_name: '演示买家A',
        amount: 1200,
        price: 1200,
      }),
      expect.objectContaining({
        user_id: 9102,
        user_name: '演示买家B',
        amount: 1300,
        price: 1300,
      }),
    ]);
  });
});
