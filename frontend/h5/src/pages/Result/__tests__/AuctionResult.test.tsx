import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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

  it('loads authoritative result and pays the winner order', async () => {
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

    fireEvent.click(screen.getByRole('button', { name: '立即支付' }));

    await waitFor(() => expect(mockedOrderApi.pay).toHaveBeenCalledWith(56));
    expect(await screen.findByText('支付成功，订单已更新')).toBeInTheDocument();
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

  it('shows 查看订单 button that navigates to /order/:id when order_id exists (T3.6)', async () => {
    render(
      <MemoryRouter
        initialEntries={['/result?id=12']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <ResultPage />
      </MemoryRouter>
    );

    const viewOrderBtn = await screen.findByRole('button', { name: '查看订单' });
    expect(viewOrderBtn).not.toBeDisabled();
    fireEvent.click(viewOrderBtn);
    expect(mockNavigate).toHaveBeenCalledWith('/order/56');
  });

  it('disables 查看订单 with 订单生成中 fallback when order_id is missing (T3.6)', async () => {
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

    const fallbackBtn = await screen.findByRole('button', { name: '订单生成中' });
    expect(fallbackBtn).toBeDisabled();
  });
});
