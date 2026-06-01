import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Profile from '../Index';
import { orderApi, userApi } from '../../../services/api';
import { notificationApi } from '../../../services/notification';
import { trackEvent } from '../../../utils/trackEvent';

const mockNavigate = jest.fn();
const mockLogout = jest.fn();
const mockAuthUser = { id: 9, email: 'buyer@example.com', name: '测试用户', role: 0 };

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
}));

jest.mock('../../../services/api', () => ({
  userApi: {
    getProfile: jest.fn(),
    getBalance: jest.fn(),
    getStats: jest.fn(),
  },
  orderApi: {
    list: jest.fn(),
  },
}));

jest.mock('../../../services/notification', () => ({
  notificationApi: {
    getTouchpointSummary: jest.fn(),
  },
}));

jest.mock('../../../store/authContext', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    user: mockAuthUser,
    token: 'token-1',
    loading: false,
    logout: mockLogout,
  }),
}));

jest.mock('../../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
  getCountBucket: (count: number) =>
    count <= 0 ? '0' : count === 1 ? '1' : count <= 5 ? '2_5' : count <= 10 ? '6_10' : '10_plus',
}));

const mockedUserApi = userApi as jest.Mocked<typeof userApi>;
const mockedOrderApi = orderApi as jest.Mocked<typeof orderApi>;
const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;
const mockTrackEvent = trackEvent as jest.MockedFunction<typeof trackEvent>;

describe('Profile migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    mockedUserApi.getProfile.mockResolvedValue({
      id: 9,
      name: '林见山',
      email: 'buyer@example.com',
      avatar: '',
      role: 0,
      created_at: '2026-05-01T00:00:00Z',
    });
    mockedUserApi.getBalance.mockResolvedValue({
      available_amount: 12288,
      frozen_amount: 600,
      currency: 'CNY',
    });
    mockedUserApi.getStats.mockResolvedValue({
      following_count: 8,
      auction_history_count: 3,
      won_count: 1,
    });
    mockedOrderApi.list.mockResolvedValue([
      {
        id: 56,
        auction_id: 12,
        product_id: 34,
        product_name: '鎏金香炉',
        final_price: 6800,
        status: 0,
        created_at: '2026-05-29T12:00:00Z',
      },
    ]);
    mockedNotificationApi.getTouchpointSummary.mockResolvedValue({
      unreadTotal: 7,
      pendingPayment: 2,
      wonNotPaid: 1,
      outbid: 3,
      endingSoon: 1,
    });
  });

  it('loads profile, balance and order entry from service wrappers', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Profile />
      </MemoryRouter>
    );

    expect(await screen.findByText('林见山')).toBeInTheDocument();
    expect(screen.getByText('¥12,288')).toBeInTheDocument();
    expect(screen.getByText('冻结 ¥600')).toBeInTheDocument();
    expect(screen.getByText('鎏金香炉')).toBeInTheDocument();

    expect(mockedUserApi.getProfile).toHaveBeenCalledTimes(1);
    expect(mockedUserApi.getBalance).toHaveBeenCalledTimes(1);
    expect(mockedUserApi.getStats).toHaveBeenCalledTimes(1);
    expect(mockedOrderApi.list).toHaveBeenCalledTimes(1);
  });

  it('wires retained profile entry buttons and logout', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Profile />
      </MemoryRouter>
    );

    expect(await screen.findByText('林见山')).toBeInTheDocument();

    expect(await screen.findByLabelText('2 条待处理提醒')).toHaveTextContent('2');
    expect(mockedNotificationApi.getTouchpointSummary).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('link', { name: /我的竞拍/ })).toHaveAttribute('href', '/history');
    expect(screen.getByRole('link', { name: /我的收藏/ })).toHaveAttribute('href', '/following');
    expect(screen.getByRole('link', { name: /消息通知/ })).toHaveAttribute('href', '/notifications');
    const addressLinks = screen.getAllByRole('link').filter((el) => el.getAttribute('href') === '/addresses');
    expect(addressLinks.length).toBeGreaterThanOrEqual(2);
    expect(screen.getByRole('link', { name: '管理收货地址' })).toHaveAttribute('href', '/addresses');

    fireEvent.click(screen.getByRole('button', { name: '退出登录' }));

    await waitFor(() => expect(mockLogout).toHaveBeenCalledTimes(1));
    expect(mockNavigate).toHaveBeenCalledWith('/login');
  });

  it('tracks profile touchpoint entry clicks', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Profile />
      </MemoryRouter>
    );

    expect(await screen.findByText('林见山')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('link', { name: /我的竞拍/ }));
    expect(mockTrackEvent).toHaveBeenCalledWith('entry_clicked', {
      source: 'profile',
      entry: 'auction_history',
      type: 'pending_payment',
      result: 'clicked',
    });

    fireEvent.click(screen.getByRole('link', { name: /消息通知/ }));
    expect(mockTrackEvent).toHaveBeenCalledWith('entry_clicked', {
      source: 'profile',
      entry: 'notification_center',
      type: 'notification',
      result: 'clicked',
    });
  });
});
