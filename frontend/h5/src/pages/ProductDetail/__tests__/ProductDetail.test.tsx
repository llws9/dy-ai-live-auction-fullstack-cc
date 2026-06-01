import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import ProductDetail from '../index';
import { auctionApi, bidApi, productApi } from '../../../services/api';
import { ThemeProvider } from '../../../store/themeContext';

jest.mock('../../../services/api', () => ({
  auctionApi: {
    get: jest.fn(),
    getBids: jest.fn(),
  },
  bidApi: {
    placeBid: jest.fn(),
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
const mockedBidApi = bidApi as jest.Mocked<typeof bidApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;

describe('ProductDetail migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    mockedAuctionApi.get.mockResolvedValue({
      id: 12,
      product_id: 34,
      live_stream_id: 5,
      status: 1,
      current_price: 1200,
      end_time: new Date(Date.now() + 60_000).toISOString(),
    });
    mockedAuctionApi.getBids.mockResolvedValue([
      { id: 1, user_id: 2, user_name: '张三', amount: 1200, created_at: new Date().toISOString() },
    ]);
    mockedProductApi.get.mockResolvedValue({
      id: 34,
      name: '清代青花瓷瓶',
      description: '釉色温润，保存完整。',
      images: ['/porcelain.jpg'],
      rules: {
        start_price: 1000,
        increment: 100,
        cap_price: 5000,
        trigger_delay_before: 30,
      },
    });
    mockedBidApi.placeBid.mockResolvedValue({
      current_price: 1300,
      ranking: [{ rank: 1, user_id: 9, user_name: '测试用户', amount: 1300 }],
    });
  });

  it('loads auction, product, bid records and places a quick bid', async () => {
    render(
      <ThemeProvider>
        <MemoryRouter
          initialEntries={['/detail?id=12']}
          future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        >
          <ProductDetail />
        </MemoryRouter>
      </ThemeProvider>
    );

    expect(await screen.findByText('清代青花瓷瓶')).toBeInTheDocument();
    expect(screen.getByText('釉色温润，保存完整。')).toBeInTheDocument();
    expect(screen.getByText('张三')).toBeInTheDocument();
    expect(screen.getAllByText('¥1,200').length).toBeGreaterThan(0);
    expect(screen.getByText('¥5,000')).toBeInTheDocument();

    expect(mockedAuctionApi.get).toHaveBeenCalledWith(12);
    expect(mockedProductApi.get).toHaveBeenCalledWith(34);
    expect(mockedAuctionApi.getBids).toHaveBeenCalledWith(12);

    fireEvent.click(screen.getByRole('button', { name: '+¥100' }));
    fireEvent.click(screen.getByRole('button', { name: '出价' }));

    await waitFor(() => expect(mockedBidApi.placeBid).toHaveBeenCalledWith(12, 1300));
    expect(await screen.findByText('出价成功！¥1,300')).toBeInTheDocument();
  });
});
