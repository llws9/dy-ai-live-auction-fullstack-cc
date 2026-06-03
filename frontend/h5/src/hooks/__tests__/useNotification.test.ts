import { act, renderHook, waitFor } from '@testing-library/react';
import { useNotification } from '../useNotification';
import { notificationApi } from '../../services/notification';
import { trackEvent } from '../../utils/trackEvent';

jest.mock('../../services/notification', () => ({
  notificationApi: {
    list: jest.fn(),
    getUnreadCount: jest.fn(),
    markAsRead: jest.fn(),
    markAllAsRead: jest.fn(),
    hotPull: jest.fn(),
  },
}));

jest.mock('../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
  getCountBucket: (count: number) =>
    count <= 0 ? '0' : count === 1 ? '1' : count <= 5 ? '2_5' : count <= 10 ? '6_10' : '10_plus',
}));

const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;
const mockTrackEvent = trackEvent as jest.MockedFunction<typeof trackEvent>;

describe('useNotification touchpoint tracking', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedNotificationApi.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 });
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 0 });
    mockedNotificationApi.hotPull.mockResolvedValue({
      notifications: [
        {
          id: 1,
          type: 'live_stream_now_live',
          title: '开播',
          content: '已开播',
          created_at: '2026-06-02T00:00:00Z',
        },
      ],
      has_more: false,
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('tracks hot pull success with returned notification bucket', async () => {
    const { result } = renderHook(() => useNotification());

    await act(async () => {
      await result.current.hotPullNotifications();
    });

    expect(mockTrackEvent).toHaveBeenCalledWith('hot_pull_triggered', {
      source: 'notification_hook',
      entry: 'hot_pull',
      type: 'live_start',
      result: 'success',
      countBucket: '1',
    });
  });

  it('tracks debounce skip when hot pull is called twice quickly', async () => {
    const { result } = renderHook(() => useNotification());

    await act(async () => {
      await result.current.hotPullNotifications();
      await result.current.hotPullNotifications();
    });

    await waitFor(() =>
      expect(mockTrackEvent).toHaveBeenCalledWith('hot_pull_triggered', {
        source: 'notification_hook',
        entry: 'hot_pull',
        type: 'live_start',
        result: 'debounced',
        countBucket: '0',
      })
    );
  });

  it('tracks hot pull failure with zero bucket', async () => {
    mockedNotificationApi.hotPull.mockRejectedValue(new Error('network'));
    jest.spyOn(console, 'error').mockImplementation(() => undefined);
    const { result } = renderHook(() => useNotification());

    await act(async () => {
      await result.current.hotPullNotifications();
    });

    expect(mockTrackEvent).toHaveBeenCalledWith('hot_pull_triggered', {
      source: 'notification_hook',
      entry: 'hot_pull',
      type: 'live_start',
      result: 'failed',
      countBucket: '0',
    });
  });
});
