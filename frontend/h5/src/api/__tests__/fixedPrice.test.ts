import { fetchItems, fetchMyPurchase, generateIdempotencyKey, purchase } from '../fixedPrice';
import { setToastFunction } from '../../services/api';

const jsonResponse = (data: unknown): Response => ({
  ok: true,
  headers: {
    get: () => 'application/json',
  },
  json: async () => ({ code: 200, data }),
} as Response);

describe('fixedPrice API', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.restoreAllMocks();
  });

  it('generateIdempotencyKey returns an RFC4122 UUID v4 string', () => {
    expect(generateIdempotencyKey()).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
    );
  });

  it('fetchItems requests live stream fixed-price items through the gateway API', async () => {
    const fetchMock = jest.fn().mockResolvedValue(jsonResponse({ items: [] }));
    global.fetch = fetchMock;

    const result = await fetchItems(1001);

    expect(result).toEqual({ items: [] });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/live-streams/1001/fixed-price/items',
      expect.objectContaining({ method: 'GET' })
    );
  });

  it('purchase sends X-Idempotency-Key and normalizes the backend voucher id', async () => {
    localStorage.setItem('auth_token', 'token-1');
    const fetchMock = jest.fn().mockResolvedValue(jsonResponse({
      order_id: 9,
      item_id: 7001,
      price: '99.00',
      remaining_stock: 86,
      status: 'success',
    }));
    global.fetch = fetchMock;

    const result = await purchase({
      itemId: 7001,
      idempotencyKey: '550e8400-e29b-41d4-a716-446655440000',
    });

    expect(result.purchase_id).toBe(9);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/fixed-price/items/7001/purchase',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          Authorization: 'Bearer token-1',
          'X-Idempotency-Key': '550e8400-e29b-41d4-a716-446655440000',
        }),
      })
    );
  });

  it('purchase lets the modal handle insufficient balance without request-layer error toast', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: false,
      status: 402,
      url: '/api/v1/fixed-price/items/7001/purchase',
      headers: { get: () => 'application/json' },
      json: async () => ({
        code: 'INSUFFICIENT_BALANCE',
        message: '余额不足',
      }),
    } as Response);
    global.fetch = fetchMock;
    const toastSpy = jest.fn();
    setToastFunction(toastSpy);

    await expect(purchase({
      itemId: 7001,
      idempotencyKey: '550e8400-e29b-41d4-a716-446655440000',
    })).rejects.toMatchObject({
      status: 402,
      code: 'INSUFFICIENT_BALANCE',
    });

    expect(toastSpy).not.toHaveBeenCalled();
  });

  it('fetchMyPurchase normalizes backend i_bought and voucher id', async () => {
    localStorage.setItem('auth_token', 'token-1');
    const fetchMock = jest.fn().mockResolvedValue(jsonResponse({
      i_bought: true,
      order_id: 9,
      price: '99.00',
      created_at: '2026-06-04T10:00:00Z',
    }));
    global.fetch = fetchMock;

    const result = await fetchMyPurchase(7001);

    expect(result).toEqual({
      i_bought: true,
      purchase_id: 9,
      price: '99.00',
      created_at: '2026-06-04T10:00:00Z',
    });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/fixed-price/items/7001/my-purchase',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          Authorization: 'Bearer token-1',
        }),
      })
    );
  });
});
