import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import HomePage from '@/pages/Home';
import { AuthProvider } from '@/store/authContext';
import { ThemeProvider } from '@/store/themeContext';
import { auctionApi, productApi } from '@/services/api';

jest.mock('@/services/api', () => ({
  auctionApi: {
    list: jest.fn(),
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

const mockAuctions = [
  {
    id: 1,
    product_id: 1,
    product_name: '测试商品1',
    product: { id: 1, name: '测试商品1', images: ['https://example.com/image1.jpg'] },
    product_image: 'https://example.com/image1.jpg',
    status: 1,
    current_price: 100,
    end_time: new Date(Date.now() + 3600000).toISOString(),
    start_time: new Date().toISOString(),
    bidder_count: 10,
  },
  {
    id: 2,
    product_id: 2,
    product_name: '测试商品2',
    product: { id: 2, name: '测试商品2', images: ['https://example.com/image2.jpg'] },
    product_image: 'https://example.com/image2.jpg',
    status: 1,
    current_price: 200,
    end_time: new Date(Date.now() + 1800000).toISOString(),
    start_time: new Date().toISOString(),
    bidder_count: 5,
  },
];

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
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
  });

  it('shows loading state initially', async () => {
    mockedAuctionApi.list.mockImplementation(() =>
      new Promise((resolve) =>
        setTimeout(() =>
          resolve({ auctions: mockAuctions }),
          100
        )
      )
    );

    renderHome();

    // Should show skeleton loaders
    expect(screen.getByRole('status')).toHaveTextContent('加载竞拍中...');
  });

  it('loads and displays auction list', async () => {
    mockedAuctionApi.list.mockResolvedValue({ auctions: mockAuctions });

    renderHome();

    await waitFor(() => {
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
      expect(screen.getByText('测试商品1')).toBeInTheDocument();
      expect(screen.getByText('测试商品2')).toBeInTheDocument();
      expect(screen.getByText('¥100')).toBeInTheDocument();
      expect(screen.getByText('¥200')).toBeInTheDocument();
    });
  });

  it('displays error state when fetch fails', async () => {
    mockedAuctionApi.list.mockRejectedValue(new Error('Network error'));

    renderHome();

    await waitFor(() => {
      expect(screen.getByText('暂无竞拍数据')).toBeInTheDocument();
    });
  });

  it('filters auctions by tab', async () => {
    mockedAuctionApi.list.mockResolvedValue({ auctions: mockAuctions });

    renderHome();

    await waitFor(() => {
      expect(screen.getByText('测试商品1')).toBeInTheDocument();
    });

    // Click "收藏" tab
    const followingTab = screen.getByRole('button', { name: '收藏' });
    fireEvent.click(followingTab);

    expect(await screen.findByText('暂无收藏直播间')).toBeInTheDocument();
  });

  it('displays empty state when no auctions', async () => {
    mockedAuctionApi.list.mockResolvedValue({ auctions: [] });

    renderHome();

    await waitFor(() => {
      expect(screen.getByText('暂无竞拍数据')).toBeInTheDocument();
    });
  });

  it('renders navigation elements', async () => {
    mockedAuctionApi.list.mockResolvedValue({ auctions: mockAuctions });

    renderHome();

    await waitFor(() => {
      expect(screen.getByText('奢华竞拍')).toBeInTheDocument();
    });

    expect(screen.getByLabelText('我的收藏')).toBeInTheDocument();
    expect(screen.getByLabelText('消息通知')).toBeInTheDocument();
  });
});
