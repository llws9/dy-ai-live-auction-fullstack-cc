import {
  createDemoFixedPriceItem,
  createDemoMerchantAuction,
  rechargeDemoUser,
  shortenDemoAuction,
  triggerOtherSkyLamp,
  triggerFollowBid,
} from '../demoApi';

describe('demoApi', () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem('auth_token', 'tk-123');
  });

  afterEach(() => {
    global.fetch = originalFetch;
    localStorage.clear();
    jest.restoreAllMocks();
  });

  it('posts follow-bid to /api/test/demo/follow-bid without /api/v1 baseURL', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    } as Response);
    global.fetch = fetchMock;

    await triggerFollowBid({ auctionId: 42, amount: 110, increment: 5 });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/test/demo/follow-bid');
    expect(url).not.toContain('/api/v1');
    expect(init).toMatchObject({
      method: 'POST',
      headers: expect.objectContaining({
        Authorization: 'Bearer tk-123',
        'Content-Type': 'application/json',
      }),
    });
    expect(JSON.parse(init.body as string)).toEqual({
      auction_id: 42,
      amount: '110',
      increment: '5',
    });
  });

  it('posts recharge with amount preserved as a decimal string', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    } as Response);
    global.fetch = fetchMock;

    await rechargeDemoUser({ userId: 9101, amount: '100.00' });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/test/demo/recharge');
    expect(JSON.parse(init.body as string)).toEqual({
      user_id: 9101,
      amount: '100.00',
    });
  });

  it('posts other sky lamp request for the current auction', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    } as Response);
    global.fetch = fetchMock;

    await triggerOtherSkyLamp({ auctionId: 42 });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/test/demo/sky-lamp');
    expect(JSON.parse(init.body as string)).toEqual({
      auction_id: 42,
    });
  });

  it('throws a readable Error on demo API error responses', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: '跟价冲突，请重试' }),
    } as Response);

    await expect(triggerFollowBid({ auctionId: 42 })).rejects.toThrow('跟价冲突，请重试');
  });

  it('posts merchant auction mode to the demo merchant endpoint', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    } as Response);
    global.fetch = fetchMock;

    await createDemoMerchantAuction('upcoming');

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/test/demo/merchant/auctions');
    expect(JSON.parse(init.body as string)).toEqual({ mode: 'upcoming' });
  });

  it('posts fixed-price auction and live stream ids to the demo merchant endpoint', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    } as Response);
    global.fetch = fetchMock;

    await createDemoFixedPriceItem({ auctionId: 123, liveStreamId: 456 });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/test/demo/merchant/fixed-price-items');
    expect(JSON.parse(init.body as string)).toEqual({ auction_id: 123, live_stream_id: 456 });
  });

  it('posts auction shorten request with a ten second remaining time', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    } as Response);
    global.fetch = fetchMock;

    await shortenDemoAuction({ auctionId: 456, remainingSeconds: 10 });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/test/demo/auctions/shorten');
    expect(JSON.parse(init.body as string)).toEqual({
      auction_id: 456,
      remaining_seconds: 10,
    });
  });
});
