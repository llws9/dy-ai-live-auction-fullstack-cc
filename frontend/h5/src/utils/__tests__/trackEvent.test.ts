import { getCountBucket, trackEvent } from '../trackEvent';

const originalFetch = global.fetch;

function readBlobAsText(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(reader.error);
    reader.readAsText(blob);
  });
}

describe('trackEvent', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(Date, 'now').mockReturnValue(1780300800000);
    global.fetch = jest.fn().mockResolvedValue({ ok: true }) as jest.Mock;
    Object.defineProperty(navigator, 'sendBeacon', {
      value: jest.fn(() => true),
      configurable: true,
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
    global.fetch = originalFetch;
  });

  it('sends touchpoint payload through sendBeacon as a JSON blob first', async () => {
    trackEvent('summary_exposed', {
      source: 'bottom_nav',
      entry: 'profile_tab',
      type: 'all',
      result: 'success',
      countBucket: '2_5',
    });

    expect(navigator.sendBeacon).toHaveBeenCalledTimes(1);
    const [url, body] = (navigator.sendBeacon as jest.Mock).mock.calls[0];
    expect(url).toBe('/api/track');
    expect(body).toBeInstanceOf(Blob);
    expect((body as Blob).type).toBe('application/json');
    expect(JSON.parse(await readBlobAsText(body as Blob))).toEqual({
      event_type: 'touchpoint_event',
      event_name: 'summary_exposed',
      params: {
        source: 'bottom_nav',
        entry: 'profile_tab',
        type: 'all',
        result: 'success',
        count_bucket: '2_5',
      },
      timestamp: 1780300800000,
    });
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it('falls back to fetch keepalive when sendBeacon returns false', () => {
    (navigator.sendBeacon as jest.Mock).mockReturnValue(false);

    trackEvent('entry_clicked', {
      source: 'profile',
      entry: 'auction_history',
      type: 'pending_payment',
      result: 'clicked',
    });

    expect(global.fetch).toHaveBeenCalledWith('/api/track', expect.objectContaining({
      method: 'POST',
      keepalive: true,
      headers: { 'Content-Type': 'application/json' },
    }));
  });

  it('does not throw when reporting fails', () => {
    (navigator.sendBeacon as jest.Mock).mockReturnValue(false);
    (global.fetch as jest.Mock).mockRejectedValue(new Error('network'));

    expect(() =>
      trackEvent('hot_pull_triggered', {
        source: 'notification_hook',
        entry: 'hot_pull',
        type: 'live_start',
        result: 'failed',
      }),
    ).not.toThrow();
  });

  it.each([
    [0, '0'],
    [1, '1'],
    [2, '2_5'],
    [5, '2_5'],
    [6, '6_10'],
    [10, '6_10'],
    [11, '10_plus'],
  ])('maps count %s to bucket %s', (count, expected) => {
    expect(getCountBucket(count)).toBe(expected);
  });
});
