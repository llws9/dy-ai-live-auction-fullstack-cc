import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Profile from '../Index';
import { orderApi, userApi } from '../../../services/api';
import { notificationApi } from '../../../services/notification';
import { getLiveRoomFootprints } from '../../../utils/liveRoomFootprints';

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

jest.mock('../../../utils/liveRoomFootprints', () => ({
  getLiveRoomFootprints: jest.fn(),
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

const mockedUserApi = userApi as jest.Mocked<typeof userApi>;
const mockedOrderApi = orderApi as jest.Mocked<typeof orderApi>;
const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;
const mockedGetLiveRoomFootprints = getLiveRoomFootprints as jest.MockedFunction<typeof getLiveRoomFootprints>;

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
      won_count: 2,
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
    mockedGetLiveRoomFootprints.mockReturnValue([
      {
        live_stream_id: 3,
        name: '玉石夜拍',
        cover: 'https://example.com/live.jpg',
        enteredAt: 1781020000000,
      },
    ]);
  });

  it('loads profile, balance and order entry from service wrappers', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Profile />
      </MemoryRouter>
    );

    expect(await screen.findByText('林见山')).toBeInTheDocument();
    const historyLinks = screen.getAllByRole('link').filter((el) => el.getAttribute('href') === '/history');
    expect(historyLinks.length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByRole('link', { name: /2\s*中标/ })).not.toBeInTheDocument();
    expect(screen.getByText('玉石夜拍')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /钱包/ })).toHaveAttribute('href', '/wallet');
    expect(screen.getByRole('link', { name: /个人卖家申请/ })).toHaveAttribute('href', '/');
    expect(screen.getByRole('link', { name: /企业商家入驻/ })).toHaveAttribute('href', '/');

    expect(mockedUserApi.getProfile).toHaveBeenCalledTimes(1);
    expect(mockedUserApi.getBalance).toHaveBeenCalledTimes(1);
    expect(mockedUserApi.getStats).toHaveBeenCalledTimes(1);
    expect(mockedOrderApi.list).toHaveBeenCalledTimes(1);
  });

  it('repairs mojibake profile names before rendering the heading', async () => {
    mockedUserApi.getProfile.mockResolvedValue({
      id: 9,
      name: 'æµ‹è¯•ç”¨æˆ·',
      email: 'buyer@example.com',
      avatar: '',
      role: 0,
      created_at: '2026-05-01T00:00:00Z',
    });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Profile />
      </MemoryRouter>
    );

    expect(await screen.findByRole('heading', { name: '测试用户' })).toBeInTheDocument();
    expect(screen.queryByText('æµ‹è¯•ç”¨æˆ·')).not.toBeInTheDocument();
  });

  it('wires retained profile entry buttons and logout', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Profile />
      </MemoryRouter>
    );

    expect(await screen.findByText('林见山')).toBeInTheDocument();

    expect(await screen.findByLabelText('1 条待处理提醒')).toHaveTextContent('1');
    expect(mockedNotificationApi.getTouchpointSummary).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('link', { name: /1 件中标待支付/ })).toHaveAttribute('href', '/orders');
    expect(screen.getByRole('link', { name: /设置（暂未开放）/ })).toHaveAttribute('href', '/');
    const notificationLinks = screen.getAllByRole('link', { name: /消息通知/ });
    expect(notificationLinks).toHaveLength(1);
    expect(notificationLinks[0]).toHaveAttribute('href', '/notifications');
    expect(notificationLinks[0].closest('section')).toHaveAttribute('aria-label', '我的竞拍');
    expect(screen.queryByLabelText('中标数量')).not.toBeInTheDocument();
    const addressLinks = screen.getAllByRole('link').filter((el) => el.getAttribute('href') === '/addresses');
    expect(addressLinks.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByRole('link', { name: /收货地址/ })).toHaveAttribute('href', '/addresses');

    fireEvent.click(screen.getByRole('button', { name: '退出登录' }));

    await waitFor(() => expect(mockLogout).toHaveBeenCalledTimes(1));
    expect(mockNavigate).toHaveBeenCalledWith('/login');
  });

  it('shows unread notification badge on the notification center entry', async () => {
    mockedNotificationApi.getTouchpointSummary.mockResolvedValue({
      unreadTotal: 1,
      pendingPayment: 4,
      wonNotPaid: 0,
      outbid: 1,
      endingSoon: 0,
    });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Profile />
      </MemoryRouter>
    );

    expect(await screen.findByText('林见山')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /消息通知/ })).toHaveAttribute('href', '/notifications');
    expect(screen.getByRole('link', { name: /消息通知/ }).closest('section')).toHaveAttribute(
      'aria-label',
      '我的竞拍'
    );
    expect(screen.getByRole('link', { name: /消息通知/ })).toContainElement(
      screen.getByLabelText('1 条待处理提醒')
    );
    expect(screen.queryByRole('link', { name: /我的竞拍.*4/ })).not.toBeInTheDocument();
  });
});
