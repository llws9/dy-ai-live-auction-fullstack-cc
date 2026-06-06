import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import AuctionList from '@/pages-new/AuctionList';
import MerchantLiveList from '@/pages-new/MerchantLiveList';
import PlatformLiveList from '@/pages-new/PlatformLiveList';
import { AuthProvider } from '@/shared/auth';

const mockLiveStreamAdminList = jest.fn();
const mockLiveStreamCreate = jest.fn();
const mockAuctionList = jest.fn();
const mockGetOverview = jest.fn();

jest.mock('@/shared/api', () => ({
  authApi: {
    getCurrentUser: jest.fn(),
  },
  liveStreamApi: {
    adminList: (...args: unknown[]) => mockLiveStreamAdminList(...args),
    create: (...args: unknown[]) => mockLiveStreamCreate(...args),
  },
  auctionApi: {
    list: (...args: unknown[]) => mockAuctionList(...args),
  },
  statisticsApi: {
    getOverview: (...args: unknown[]) => mockGetOverview(...args),
  },
}));

function renderWithRole(role: number, ui: React.ReactElement) {
  localStorage.setItem('admin_auth_token', 'token');
  localStorage.setItem('admin_auth_user', JSON.stringify({
    id: role === 2 ? 1003 : 1002,
    name: role === 2 ? '系统管理员' : '商家用户',
    email: role === 2 ? 'admin@example.com' : 'merchant@example.com',
    role,
    created_at: '2026-06-05T00:00:00Z',
  }));

  return render(
    <MemoryRouter>
      <AuthProvider>
        {ui}
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('management page role visibility', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
    mockLiveStreamAdminList.mockResolvedValue({
      list: [
        {
          id: 101,
          name: '平台直播间 A',
          status: 1,
          streamer_name: '主播 A',
          viewer_count: 12,
          auction_count: 2,
        },
      ],
    });
    mockLiveStreamCreate.mockResolvedValue({
      id: 101,
      name: '商家用户的直播间',
      status: 1,
    });
    mockAuctionList.mockResolvedValue({
      list: [
        {
          id: 201,
          status: 1,
          start_time: '2026-06-06T10:00:00Z',
          current_price: 1200,
          live_stream_id: 101,
          live_stream_name: '平台直播间 A',
          product: { name: '竞拍商品 A' },
        },
      ],
      total: 1,
    });
    mockGetOverview.mockResolvedValue({
      total_auctions: 1,
      total_users: 3,
      today_revenue: 1200,
    });
  });

  it('shows platform live-room copy and hides merchant live-room creation actions for admins', async () => {
    renderWithRole(2, <PlatformLiveList />);

    expect(await screen.findByText('平台直播间 A')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '平台直播间管理' })).toBeInTheDocument();
    expect(screen.getByText('管理平台直播间和直播排期')).toBeInTheDocument();
    expect(screen.queryByText('管理您的直播间和直播排期')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '创建直播间' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /创建新直播间/ })).not.toBeInTheDocument();
  });

  it('keeps merchant live-room creation actions and owner copy for merchants', async () => {
    renderWithRole(1, <MerchantLiveList />);

    expect(await screen.findByText('平台直播间 A')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '我的直播间' })).toBeInTheDocument();
    expect(screen.getByText('管理我的直播间和直播排期')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建直播间' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /创建新直播间/ })).toBeInTheDocument();
  });

  it('lets merchants click create live-room and call the backend create endpoint', async () => {
    renderWithRole(1, <MerchantLiveList />);

    const createButton = await screen.findByRole('button', { name: '创建直播间' });
    expect(createButton).toBeEnabled();

    await userEvent.click(createButton);

    expect(mockLiveStreamCreate).toHaveBeenCalledWith({
      name: '商家用户的直播间',
      description: '商家用户的直播间排期',
      streamer_name: '商家用户',
    });
  });

  it('hides auction rule and creation actions for platform admins', async () => {
    renderWithRole(2, <AuctionList />);

    expect(await screen.findByText('竞拍商品 A')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '规则模板' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '创建竞拍场次' })).not.toBeInTheDocument();
  });
});
