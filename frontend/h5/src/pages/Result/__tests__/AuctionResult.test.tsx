import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import ResultPage from '../index';
import { auctionApi, orderApi, productApi } from '../../../services/api';

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
});
