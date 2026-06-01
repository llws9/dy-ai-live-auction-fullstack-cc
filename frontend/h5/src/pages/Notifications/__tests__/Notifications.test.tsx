import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import NotificationsPage from '../index';
import { notificationApi } from '../../../services/notification';
import { trackEvent } from '../../../utils/trackEvent';

const mockNavigate = jest.fn();

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
}));

jest.mock('../../../services/notification', () => ({
  notificationApi: {
    list: jest.fn(),
    getUnreadCount: jest.fn(),
    markAsRead: jest.fn(),
    markAllAsRead: jest.fn(),
  },
}));

jest.mock('../../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
  getCountBucket: (count: number) =>
    count <= 0 ? '0' : count === 1 ? '1' : count <= 5 ? '2_5' : count <= 10 ? '6_10' : '10_plus',
}));

jest.mock('../../../components/ThemeToggle', () => ({
  __esModule: true,
  default: () => null,
}));

const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;
const mockTrackEvent = trackEvent as jest.MockedFunction<typeof trackEvent>;

describe('Notifications migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    mockedNotificationApi.list.mockResolvedValue({
      items: [
        {
          id: 1,
          type: 'live_stream_now_live',
          title: '直播开播',
          content: '林掌柜的苏州玉器直播间已开播',
          data: { live_stream_id: 88 },
          created_at: '2026-05-30T08:00:00Z',
        },
        {
          id: 2,
          type: 'auction_won',
          title: '竞拍成功',
          content: '你已成功拍下鎏金香炉',
          data: { auction_id: 66 },
          read_at: '2026-05-30T08:10:00Z',
          created_at: '2026-05-30T08:05:00Z',
        },
      ],
      total: 2,
      page: 1,
      page_size: 20,
    });
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 1 });
    mockedNotificationApi.markAsRead.mockResolvedValue(undefined);
    mockedNotificationApi.markAllAsRead.mockResolvedValue(undefined);
  });

  it('loads notifications from notificationApi and renders unread state', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <NotificationsPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('林掌柜的苏州玉器直播间已开播')).toBeInTheDocument();
    expect(screen.getByText('你已成功拍下鎏金香炉')).toBeInTheDocument();
    expect(screen.getByText('1 条未读')).toBeInTheDocument();
    expect(screen.getByText('2 条消息')).toBeInTheDocument();

    await waitFor(() => expect(mockedNotificationApi.list).toHaveBeenCalledWith(1, 20));
    expect(mockedNotificationApi.getUnreadCount).toHaveBeenCalled();
    expect(mockTrackEvent).toHaveBeenCalledWith('notification_list_exposed', {
      source: 'notification_center',
      entry: 'notification_center',
      type: 'notification',
      result: 'success',
      countBucket: '2_5',
    });
  });

  it('marks unread notifications as read before navigating to the mapped target', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <NotificationsPage />
      </MemoryRouter>
    );

    const liveNotice = await screen.findByRole('button', { name: /林掌柜的苏州玉器直播间已开播/ });
    fireEvent.click(liveNotice);

    await waitFor(() => expect(mockedNotificationApi.markAsRead).toHaveBeenCalledWith(1));
    expect(mockTrackEvent).toHaveBeenCalledWith('notification_item_clicked', {
      source: 'notification_center',
      entry: 'notification_item',
      type: 'live_start',
      result: 'clicked',
    });
    expect(mockNavigate).toHaveBeenCalledWith('/live?id=88');
  });

  it('tracks mark all read success', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <NotificationsPage />
      </MemoryRouter>
    );

    await screen.findByText('1 条未读');
    const button = await screen.findByRole('button', { name: '全部已读' });
    fireEvent.click(button);

    await waitFor(() => expect(mockedNotificationApi.markAllAsRead).toHaveBeenCalled());
    expect(mockTrackEvent).toHaveBeenCalledWith('mark_read', {
      source: 'notification_center',
      entry: 'mark_all_read',
      type: 'all',
      result: 'success',
    });
  });

  it('navigates order notifications to /order/:id when data.order_id is present (T3.6)', async () => {
    mockedNotificationApi.list.mockResolvedValue({
      items: [
        {
          id: 9,
          type: 'order_paid',
          title: '订单已支付',
          content: '你的订单 #1234 已成功支付',
          data: { order_id: 1234 },
          created_at: '2026-05-30T09:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
    });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <NotificationsPage />
      </MemoryRouter>
    );

    const orderNotice = await screen.findByRole('button', { name: /订单 #1234 已成功支付/ });
    fireEvent.click(orderNotice);

    await waitFor(() => expect(mockedNotificationApi.markAsRead).toHaveBeenCalledWith(9));
    expect(mockNavigate).toHaveBeenCalledWith('/order/1234');
  });
});
