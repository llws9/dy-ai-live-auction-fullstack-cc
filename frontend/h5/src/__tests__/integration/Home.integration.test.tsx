import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import HomePage from '@/pages/Home';
import { AuthProvider } from '@/store/authContext';
import { ThemeProvider } from '@/store/themeContext';
import { auctionApi, liveStreamApi, productApi } from '@/services/api';

jest.mock('@/services/api', () => ({
  auctionApi: {
    list: jest.fn(),
  },
  liveStreamApi: {
    list: jest.fn(),
    get: jest.fn(),
  },
  followApi: {
    getFollowedLiveStreams: jest.fn(),
  },
  productApi: {
    listCategories: jest.fn(),
  },
}));

jest.mock('@/services/notification', () => ({
  notificationApi: {
    getUnreadCount: jest.fn(),
  },
}));

const mockLiveRooms = [
  {
    id: 1,
    name: '测试直播间1',
    status: 1,
    current_auction_id: 11,
    current_price: '100',
    recent_deals: [],
  },
  {
    id: 2,
    name: '测试直播间2',
    status: 1,
    current_auction_id: 12,
    current_price: '200',
    recent_deals: [],
  },
];

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedLiveStreamApi = liveStreamApi as jest.Mocked<typeof liveStreamApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;

function renderHome() {
  return render(
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <HomePage />
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

describe('Home Page Integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedProductApi.listCategories.mockResolvedValue([]);
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0 });
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0 });
  });

  it('shows loading state initially', async () => {
    mockedLiveStreamApi.list.mockImplementation(() =>
      new Promise((resolve) =>
        setTimeout(() =>
          resolve({ list: mockLiveRooms, total: mockLiveRooms.length }),
          100
        )
      )
    );

    renderHome();

    // Should show skeleton loaders
    expect(screen.getByRole('status')).toHaveTextContent('加载竞拍中...');
  });

  it('loads and displays live room list', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: mockLiveRooms, total: mockLiveRooms.length });

    renderHome();

    await waitFor(() => {
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
      expect(screen.getByRole('heading', { name: '测试直播间1' })).toBeInTheDocument();
      expect(screen.getByRole('heading', { name: '测试直播间2' })).toBeInTheDocument();
      expect(screen.getByText('当前 ¥100')).toBeInTheDocument();
      expect(screen.getByText('当前 ¥200')).toBeInTheDocument();
    });
  });

  it('displays error state when fetch fails', async () => {
    mockedLiveStreamApi.list.mockRejectedValue(new Error('Network error'));

    renderHome();

    await waitFor(() => {
      expect(screen.getByText('暂无竞拍数据')).toBeInTheDocument();
    });
  });

  it('filters by tab', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: mockLiveRooms, total: mockLiveRooms.length });

    renderHome();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: '测试直播间1' })).toBeInTheDocument();
    });

    // Click "收藏" tab
    const followingTab = screen.getByRole('button', { name: '收藏' });
    fireEvent.click(followingTab);

    expect(await screen.findByText('暂无收藏直播间')).toBeInTheDocument();
  });

  it('displays empty state when no live rooms', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0 });

    renderHome();

    await waitFor(() => {
      expect(screen.getByText('暂无竞拍数据')).toBeInTheDocument();
    });
  });

  it('renders navigation elements', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: mockLiveRooms, total: mockLiveRooms.length });

    renderHome();

    await waitFor(() => {
      expect(screen.getByText('奢华竞拍')).toBeInTheDocument();
    });

    expect(screen.getByLabelText('我的收藏')).toBeInTheDocument();
    expect(screen.getByLabelText('消息通知')).toBeInTheDocument();
  });
});
