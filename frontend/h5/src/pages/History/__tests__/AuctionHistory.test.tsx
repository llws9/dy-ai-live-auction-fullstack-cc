import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import HistoryPage from '../index';
import { orderApi } from '../../../services/api';

jest.mock('../../../services/api', () => ({
  orderApi: {
    history: jest.fn(),
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

const mockedOrderApi = orderApi as jest.Mocked<typeof orderApi>;

describe('AuctionHistory migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    mockedOrderApi.history.mockResolvedValue({
      list: [
        {
          auction_id: 12,
          product_name: '鎏金香炉',
          final_price: 6800,
          is_winner: true,
          bid_count: 5,
          created_at: '2026-05-29T12:00:00Z',
        },
        {
          auction_id: 13,
          product_name: '宋瓷盏',
          final_price: 4200,
          is_winner: false,
          bid_count: 2,
          created_at: '2026-05-28T12:00:00Z',
        },
      ],
      total: 2,
    });
  });

  it('loads documented history records without order payment behavior', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <HistoryPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('鎏金香炉')).toBeInTheDocument();
    expect(screen.getByText('宋瓷盏')).toBeInTheDocument();
    expect(screen.getAllByText('竞拍成功').length).toBeGreaterThan(0);
    expect(screen.getAllByText('未中标').length).toBeGreaterThan(0);
    expect(screen.getByText('出价 5 次')).toBeInTheDocument();
    expect(screen.getAllByText('¥6,800').length).toBeGreaterThan(0);

    expect(screen.queryByRole('button', { name: /立即支付/ })).not.toBeInTheDocument();
    expect(screen.queryByText(/模拟支付/)).not.toBeInTheDocument();
    await waitFor(() => expect(mockedOrderApi.history).toHaveBeenCalledWith({ page: 1, page_size: 20 }));
  });
});
