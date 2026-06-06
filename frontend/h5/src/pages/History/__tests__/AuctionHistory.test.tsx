import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import HistoryPage from '../index';
import { orderApi } from '../../../services/api';
import { ThemeProvider } from '../../../store/themeContext';

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
          status: 0,
          bid_count: 5,
          created_at: '2026-05-29T12:00:00Z',
        },
        {
          auction_id: 14,
          product_name: '青花瓷茶具',
          final_price: 570,
          is_winner: true,
          status: 1,
          bid_count: 1,
          created_at: '2026-05-30T12:00:00Z',
        },
        {
          auction_id: 13,
          product_name: '宋瓷盏',
          final_price: 4200,
          is_winner: false,
          status: 0,
          bid_count: 2,
          created_at: '2026-05-28T12:00:00Z',
        },
      ],
      total: 3,
    });
  });

  it('loads documented history records without order payment behavior', async () => {
    render(
      <ThemeProvider>
        <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
          <HistoryPage />
        </MemoryRouter>
      </ThemeProvider>
    );

    expect(await screen.findByText('鎏金香炉')).toBeInTheDocument();
    expect(screen.getByText('青花瓷茶具')).toBeInTheDocument();
    expect(screen.getByText('宋瓷盏')).toBeInTheDocument();
    expect(screen.getByText('待处理')).toBeInTheDocument();
    expect(screen.getAllByText('竞拍成功').length).toBeGreaterThan(0);
    expect(screen.getAllByText('未中标').length).toBeGreaterThan(0);
    expect(screen.getByText('出价 5 次')).toBeInTheDocument();
    expect(screen.getAllByText('¥6,800').length).toBeGreaterThan(0);

    expect(screen.queryByRole('button', { name: /立即支付/ })).not.toBeInTheDocument();
    expect(screen.queryByText(/模拟支付/)).not.toBeInTheDocument();
    await waitFor(() => expect(mockedOrderApi.history).toHaveBeenCalledWith({ page: 1, page_size: 20 }));
  });

  it('distinguishes pending won records from processed won records', async () => {
    render(
      <ThemeProvider>
        <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
          <HistoryPage />
        </MemoryRouter>
      </ThemeProvider>
    );

    expect(await screen.findByRole('article', { name: /LOT 12 待处理/ })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: /LOT 14 已处理/ })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: /LOT 13 未中标/ })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '查看并处理' })).toHaveAttribute('href', '/result?id=12');
    expect(screen.getByRole('link', { name: '查看结果' })).toHaveAttribute('href', '/result?id=14');
    expect(screen.getByRole('link', { name: '查看详情' })).toHaveAttribute('href', '/detail?id=13');
  });

  it('opens won filter from profile deep link', async () => {
    render(
      <ThemeProvider>
        <MemoryRouter initialEntries={['/history?filter=won']} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
          <HistoryPage />
        </MemoryRouter>
      </ThemeProvider>
    );

    expect(await screen.findByText('鎏金香炉')).toBeInTheDocument();
    expect(screen.getByText('青花瓷茶具')).toBeInTheDocument();
    expect(screen.queryByText('宋瓷盏')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '竞拍成功' })).toHaveClass('filterActive');
  });
});
