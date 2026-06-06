import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import ResultPage from '../index';
import { auctionApi, orderApi, productApi } from '../../../services/api';

const mockNavigate = jest.fn();

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
}));

jest.mock('../../../services/api', () => ({
  auctionApi: {
    getResult: jest.fn(),
    get: jest.fn(),
    getBids: jest.fn(),
  },
  orderApi: {
    pay: jest.fn(),
  },
  productApi: {
    get: jest.fn(),
  },
}));

jest.mock('../../../store/authContext', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    user: { id: 9, name: '测试用户', role: 0 },
    token: 'token-1',
    loading: false,
  }),
}));

jest.mock('../../../components/ThemeToggle', () => ({
  __esModule: true,
  default: () => null,
}));

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedOrderApi = orderApi as jest.Mocked<typeof orderApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;

describe('AuctionResult migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    mockedAuctionApi.getResult.mockResolvedValue({
      auction_id: 12,
      id: 12,
      product_id: 34,
      status: 3,
      final_price: 6800,
      winner_id: 9,
      order_id: 56,
      ended_at: new Date().toISOString(),
      won_bid: {
        id: 7,
        user_id: 9,
        user_name: '测试用户',
        amount: 6800,
        created_at: new Date().toISOString(),
      },
    });
    mockedProductApi.get.mockResolvedValue({
      id: 34,
      name: '鎏金香炉',
      images: ['/incense-burner.jpg'],
    });
    mockedOrderApi.pay.mockResolvedValue({
      id: 56,
      status: 1,
      final_price: 6800,
      paid_at: new Date().toISOString(),
    });
  });

  it('loads authoritative result and shows payment placeholder dialog for winner', async () => {
    render(
      <MemoryRouter
        initialEntries={['/result?id=12']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <ResultPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('鎏金香炉')).toBeInTheDocument();
    expect(screen.getByText('恭喜中标')).toBeInTheDocument();
    expect(screen.getAllByText('¥6,800').length).toBeGreaterThan(0);

    expect(mockedAuctionApi.getResult).toHaveBeenCalledWith(12);
    expect(mockedProductApi.get).toHaveBeenCalledWith(34);

    expect(screen.queryByRole('button', { name: '订单待生成' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '订单生成中' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '查看订单' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '去支付' }));

    expect(await screen.findByRole('dialog', { name: '支付链路待完善' })).toBeInTheDocument();
    expect(screen.getByText('当前支付链路仍在建设中，暂时无法在 H5 内完成支付。')).toBeInTheDocument();
    expect(mockedOrderApi.pay).not.toHaveBeenCalled();
  });

  it('repairs mojibake product name on result page', async () => {
    mockedProductApi.get.mockResolvedValueOnce({
      id: 34,
      name: 'ç¨€æœ‰ç å®',
      images: ['/jewelry.jpg'],
    });

    render(
      <MemoryRouter
        initialEntries={['/result?id=12']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <ResultPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('稀有珠宝')).toBeInTheDocument();
    expect(screen.queryByText('ç¨€æœ‰ç å®')).not.toBeInTheDocument();
  });

  it('repairs mojibake winner name and renders bid time value before label', async () => {
    mockedAuctionApi.getResult.mockResolvedValueOnce({
      auction_id: 12,
      id: 12,
      product_id: 34,
      status: 3,
      final_price: 6800,
      winner_id: 9,
      order_id: 56,
      ended_at: new Date('2026-06-07T02:20:00Z').toISOString(),
      won_bid: {
        id: 7,
        user_id: 9,
        user_name: 'æœ¬åœ°æµ‹è¯•ç”¨æˆ·',
        amount: 6800,
        created_at: new Date('2026-06-07T02:16:40Z').toISOString(),
      },
    });

    render(
      <MemoryRouter
        initialEntries={['/result?id=12']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <ResultPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('本地测试用户')).toBeInTheDocument();
    expect(screen.queryByText('æœ¬åœ°æµ‹è¯•ç”¨æˆ·')).not.toBeInTheDocument();

    const bidTimeLabel = screen.getByText('出价时间');
    const bidTimeValue = bidTimeLabel.previousElementSibling;
    expect(bidTimeValue?.tagName).toBe('STRONG');
    expect(bidTimeValue).toHaveTextContent('2026');
  });

  it('does not show 查看订单 action when order_id exists', async () => {
    render(
      <MemoryRouter
        initialEntries={['/result?id=12']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <ResultPage />
      </MemoryRouter>
    );

    expect(await screen.findByRole('button', { name: '去支付' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '查看订单' })).not.toBeInTheDocument();
  });

  it('shows enabled 去支付 action when order_id is missing', async () => {
    mockedAuctionApi.getResult.mockResolvedValueOnce({
      auction_id: 12,
      id: 12,
      product_id: 34,
      status: 3,
      final_price: 6800,
      winner_id: 9,
      ended_at: new Date().toISOString(),
    });

    render(
      <MemoryRouter
        initialEntries={['/result?id=12']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <ResultPage />
      </MemoryRouter>
    );

    const payButton = await screen.findByRole('button', { name: '去支付' });
    expect(payButton).toBeEnabled();
    expect(screen.queryByRole('button', { name: '订单待生成' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '订单生成中' })).not.toBeInTheDocument();
  });

  it('shows a single primary 返回首页 action instead of 继续竞拍 for ended non-winner result', async () => {
    mockedAuctionApi.getResult.mockResolvedValueOnce({
      auction_id: 12,
      id: 12,
      product_id: 34,
      status: 3,
      final_price: 6800,
      winner_id: 18,
      ended_at: new Date().toISOString(),
      won_bid: {
        id: 7,
        user_id: 18,
        user_name: '其他用户',
        amount: 6800,
        created_at: new Date().toISOString(),
      },
    });

    render(
      <MemoryRouter
        initialEntries={['/result?id=12']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <ResultPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('竞拍已结束')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '继续竞拍' })).not.toBeInTheDocument();

    const homeLinks = screen.getAllByRole('link', { name: '返回首页' });
    expect(homeLinks).toHaveLength(1);
    expect(homeLinks[0]).toHaveAttribute('href', '/');
    expect(homeLinks[0]).toHaveClass('primaryLink');
  });
});
