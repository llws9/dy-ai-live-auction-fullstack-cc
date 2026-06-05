import { trackBusinessEvent } from '../businessEvent';

const originalFetch = global.fetch;

describe('trackBusinessEvent', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
    localStorage.setItem('auth_token', 'token-1');
    global.fetch = jest.fn().mockResolvedValue({ ok: true }) as jest.Mock;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    localStorage.clear();
  });

  it('reports authenticated business events to the gateway events endpoint', () => {
    trackBusinessEvent('live_room_enter', {
      source: 'live_reminder',
      liveStreamId: 1001,
      auctionId: 2002,
      productId: 3003,
      metadata: { client_event_id: 'evt-1' },
    });

    expect(global.fetch).toHaveBeenCalledWith('/api/v1/events', expect.objectContaining({
      method: 'POST',
      keepalive: true,
      headers: {
        'Authorization': 'Bearer token-1',
        'Content-Type': 'application/json',
      },
    }));
    const body = JSON.parse((global.fetch as jest.Mock).mock.calls[0][1].body);
    expect(body).toEqual({
      event_type: 'live_room_enter',
      source: 'live_reminder',
      live_stream_id: 1001,
      auction_id: 2002,
      product_id: 3003,
      metadata: { client_event_id: 'evt-1' },
    });
  });

  it('does not report when the user is not authenticated', () => {
    localStorage.clear();

    trackBusinessEvent('bid_button_click', { source: 'live_room' });

    expect(global.fetch).not.toHaveBeenCalled();
  });
});
