import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import type { FixedPriceItem } from '../../api/fixedPrice';
import { fetchItems } from '../../api/fixedPrice';
import { reduceItems, useFixedPriceItems } from '../useFixedPriceItems';

jest.mock('../../api/fixedPrice', () => ({
  fetchItems: jest.fn(),
}));

const mockWsInstances: Array<{
  auctionId: number;
  token?: string;
  liveStreamId?: number;
  handlers: Map<string, Set<(data: unknown) => void>>;
  connect: jest.Mock<() => Promise<void>>;
  disconnect: jest.Mock;
  on: jest.Mock;
  off: jest.Mock;
  emit: (type: string, data: unknown) => void;
}> = [];

jest.mock('../../services/websocket', () => ({
  __esModule: true,
  default: class MockSocket {
    handlers = new Map<string, Set<(data: unknown) => void>>();
    connect = jest.fn<() => Promise<void>>().mockResolvedValue(undefined);
    disconnect = jest.fn();
    on = jest.fn((type: string, handler: (data: unknown) => void) => {
      const handlers = this.handlers.get(type) ?? new Set<(data: unknown) => void>();
      handlers.add(handler);
      this.handlers.set(type, handlers);
    });
    off = jest.fn((type: string, handler: (data: unknown) => void) => {
      this.handlers.get(type)?.delete(handler);
    });

    auctionId: number;
    token?: string;
    liveStreamId?: number;

    constructor(auctionId: number, token?: string, liveStreamId?: number) {
      this.auctionId = auctionId;
      this.token = token;
      this.liveStreamId = liveStreamId;
      mockWsInstances.push(this);
    }

    emit(type: string, data: unknown) {
      this.handlers.get(type)?.forEach((handler) => handler(data));
    }
  },
}));

const baseItem: FixedPriceItem = {
  id: 7001,
  product_id: 5001,
  price: '99.00',
  total_stock: 100,
  remaining_stock: 100,
  status: 'on_sale',
  product_brief: { id: 5001, title: '翡翠' },
};

describe('reduceItems', () => {
  it('adds a newly listed item from websocket payload', () => {
    const next = reduceItems([], {
      type: 'fixed_price_listed',
      payload: { item: baseItem },
    });

    expect(next).toEqual([baseItem]);
  });

  it('updates remaining_stock for fixed_price_stock', () => {
    const next = reduceItems([baseItem], {
      type: 'fixed_price_stock',
      payload: { item_id: 7001, remaining_stock: 87 },
    });

    expect(next[0].remaining_stock).toBe(87);
  });

  it('sets sold_out status and zero stock for fixed_price_sold_out', () => {
    const next = reduceItems([baseItem], {
      type: 'fixed_price_sold_out',
      payload: { item_id: 7001 },
    });

    expect(next[0].status).toBe('sold_out');
    expect(next[0].remaining_stock).toBe(0);
  });

  it('removes an offline item', () => {
    const next = reduceItems([baseItem], {
      type: 'fixed_price_offline',
      payload: { item_id: 7001 },
    });

    expect(next).toHaveLength(0);
  });

  it('returns the previous state reference for unknown message types', () => {
    const state = [baseItem];
    const next = reduceItems(state, { type: 'noop', payload: {} });

    expect(next).toBe(state);
  });
});

describe('useFixedPriceItems', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockWsInstances.length = 0;
  });

  it('loads initial items and exposes a byId index', async () => {
    jest.mocked(fetchItems).mockResolvedValue({ items: [baseItem] });

    const { result } = renderHook(() => useFixedPriceItems(7001, 1001, 'token-1'));

    await waitFor(() => expect(result.current.items).toEqual([baseItem]));
    expect(result.current.byId[7001]).toEqual(baseItem);
    expect(result.current.latestListedItem).toBeNull();
    expect(fetchItems).toHaveBeenCalledWith(7001);
  });

  it('skips REST and WS setup until auction and liveStream are available', () => {
    const { result } = renderHook(() => useFixedPriceItems(0, 0));

    expect(fetchItems).not.toHaveBeenCalled();
    expect(mockWsInstances).toHaveLength(0);
    expect(result.current.items).toEqual([]);
    expect(result.current.socket).toBeNull();
  });

  it('subscribes to fixed-price websocket messages and applies incremental updates', async () => {
    jest.mocked(fetchItems).mockResolvedValue({ items: [baseItem] });
    const { result } = renderHook(() => useFixedPriceItems(7001, 1001, 'token-1'));

    await waitFor(() => expect(mockWsInstances).toHaveLength(1));
    await waitFor(() => expect(result.current.socket).toBe(mockWsInstances[0]));
    expect(mockWsInstances[0]).toMatchObject({ auctionId: 1001, token: 'token-1' });

    act(() => {
      mockWsInstances[0].emit('fixed_price_stock', { item_id: 7001, remaining_stock: 86 });
    });

    await waitFor(() => expect(result.current.byId[7001].remaining_stock).toBe(86));
    expect(mockWsInstances[0].on).toHaveBeenCalledWith('fixed_price_listed', expect.any(Function));
    expect(mockWsInstances[0].on).toHaveBeenCalledWith('fixed_price_stock', expect.any(Function));
    expect(mockWsInstances[0].on).toHaveBeenCalledWith('fixed_price_sold_out', expect.any(Function));
    expect(mockWsInstances[0].on).toHaveBeenCalledWith('fixed_price_offline', expect.any(Function));
  });

  it('exposes listed item events for websocket fixed_price_listed messages', async () => {
    jest.mocked(fetchItems).mockResolvedValue({ items: [baseItem] });
    const listedItem: FixedPriceItem = {
      ...baseItem,
      id: 7002,
      product_brief: { id: 5002, title: '新上架翡翠' },
    };
    const { result } = renderHook(() => useFixedPriceItems(7001, 1001, 'token-1'));

    await waitFor(() => expect(result.current.items).toEqual([baseItem]));
    expect(result.current.latestListedItem).toBeNull();

    act(() => {
      mockWsInstances[0].emit('fixed_price_listed', { item: listedItem });
    });

    await waitFor(() => {
      expect(result.current.byId[7002]).toEqual(listedItem);
      expect(result.current.latestListedItem).toEqual({ item: listedItem, sequence: 1 });
    });
  });

  it('waits for an auth token before opening the fixed-price websocket and reconnects after login', async () => {
    jest.mocked(fetchItems).mockResolvedValue({ items: [baseItem] });
    const { rerender, result } = renderHook(
      ({ token }) => useFixedPriceItems(7001, 1001, token),
      { initialProps: { token: null as string | null } },
    );

    await waitFor(() => expect(result.current.items).toEqual([baseItem]));
    expect(mockWsInstances).toHaveLength(0);
    expect(result.current.socket).toBeNull();

    rerender({ token: 'merchant-token' });

    await waitFor(() => expect(mockWsInstances).toHaveLength(1));
    expect(mockWsInstances[0]).toMatchObject({ auctionId: 1001, token: 'merchant-token' });
    expect(result.current.socket).toBe(mockWsInstances[0]);
  });

  it('adds same-page demo fixed-price events to the item list and latest listed event', async () => {
    jest.mocked(fetchItems).mockResolvedValue({ items: [baseItem] });
    const listedItem: FixedPriceItem = {
      ...baseItem,
      id: 7003,
      product_id: 5003,
      product_brief: { id: 5003, title: 'Demo 一口价商品' },
    };
    const { result } = renderHook(() => useFixedPriceItems(7001, 1001, 'token-1'));

    await waitFor(() => expect(result.current.items).toEqual([baseItem]));

    act(() => {
      window.dispatchEvent(new CustomEvent('demo:fixed-price-listed', {
        detail: {
          auctionId: 7001,
          liveStreamId: 1001,
          item: listedItem,
        },
      }));
    });

    await waitFor(() => {
      expect(result.current.byId[7003]).toEqual(listedItem);
      expect(result.current.latestListedItem).toEqual({ item: listedItem, sequence: 1 });
    });
  });
});
