import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import NotificationsPage from '../index';
import { notificationApi } from '../../../services/notification';

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

const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;

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
    expect(mockNavigate).toHaveBeenCalledWith('/live?id=88');
  });
});
