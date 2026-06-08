import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import ProductDetail from '../index';
import { auctionApi, productApi, productReminderApi } from '../../../services/api';
import { ThemeProvider } from '../../../store/themeContext';

jest.mock('../../../services/api', () => ({
  auctionApi: {
    get: jest.fn(),
    getBids: jest.fn(),
  },
  productApi: {
    get: jest.fn(),
  },
  productReminderApi: {
    subscribe: jest.fn(),
    list: jest.fn(),
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
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;
const mockedProductReminderApi = productReminderApi as jest.Mocked<typeof productReminderApi>;

const LocationProbe = () => {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}{location.search}</div>;
};

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
    mockedProductReminderApi.list.mockResolvedValue({ items: [] });
  });

  it('进行中详情页展示参与竞拍入口，不在详情页直接出价', async () => {
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
    expect(screen.getByText('进行中')).toBeInTheDocument();

    expect(mockedAuctionApi.get).toHaveBeenCalledWith(12);
    expect(mockedProductApi.get).toHaveBeenCalledWith(34);
    expect(mockedAuctionApi.getBids).toHaveBeenCalledWith(12);
    expect(screen.queryByRole('button', { name: '+¥100' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '出价' })).not.toBeInTheDocument();

    const participate = screen.getByRole('link', { name: '参与竞拍' });
    expect(participate).toHaveAttribute('href', '/live?id=5&auction_id=12');
    expect(participate.closest('footer')).toBeNull();
  });

  it('从直播间进入商品详情时，顶部返回回到上一页直播间', async () => {
    render(
      <ThemeProvider>
        <MemoryRouter
          initialEntries={[
            '/live?id=5&auction_id=12',
            { pathname: '/detail', search: '?id=12', state: { from: 'live' } },
          ]}
          initialIndex={1}
          future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        >
          <Routes>
            <Route path="/live" element={<LocationProbe />} />
            <Route path="/detail" element={<ProductDetail />} />
            <Route path="/" element={<LocationProbe />} />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    );

    expect(await screen.findByText('清代青花瓷瓶')).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('返回'));

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/live?id=5&auction_id=12'));
  });

  it('repairs mojibake product copy on detail page', async () => {
    mockedProductApi.get.mockResolvedValueOnce({
      id: 34,
      name: 'è€è±é’»çŸ³æˆ’æŒ‡',
      description: 'ç²¾é€‰ä¸»çŸ³ï¼Œç«å½©å‡ºè‰²',
      images: ['/ring.jpg'],
      rules: {
        start_price: 1000,
        increment: 100,
      },
    });

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

    expect(await screen.findByText('老花钻石戒指')).toBeInTheDocument();
    expect(screen.getByText('精选主石，火彩出色')).toBeInTheDocument();
    expect(screen.queryByText('è€è±é’»çŸ³æˆ’æŒ‡')).not.toBeInTheDocument();
  });

  it('待开始详情页展示起拍价并提供订阅入口，不展示竞拍结果', async () => {
    mockedAuctionApi.get.mockResolvedValueOnce({
      id: 12,
      product_id: 34,
      live_stream_id: 5,
      status: 0,
      current_price: 399,
      start_price: 0,
      start_time: '2026-06-04T18:39:00+08:00',
      end_time: new Date(Date.now() + 60_000).toISOString(),
    });
    mockedAuctionApi.getBids.mockResolvedValueOnce([]);
    mockedProductApi.get.mockResolvedValueOnce({
      id: 34,
      name: '手作陶瓷茶具套装',
      description: '即将开拍的家居生活商品',
      images: ['/tea-set.jpg'],
      rules: {
        start_price: 0,
        increment: 100,
      },
    });
    mockedProductReminderApi.subscribe.mockResolvedValueOnce({ product_id: 34 });

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

    expect(await screen.findByText('手作陶瓷茶具套装')).toBeInTheDocument();
    expect(screen.getByText('起拍价')).toBeInTheDocument();
    expect(screen.getByText(/^开拍 /)).toBeInTheDocument();
    expect(screen.queryByText(/截止/)).not.toBeInTheDocument();
    expect(screen.getAllByText('¥0').length).toBeGreaterThan(0);
    expect(screen.queryByText('¥399')).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '查看竞拍结果' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '订阅开拍提醒' }).closest('footer')).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: '订阅开拍提醒' }));

    await waitFor(() => expect(mockedProductReminderApi.subscribe).toHaveBeenCalledWith(34));
    expect(screen.getByRole('button', { name: '已订阅' })).toBeDisabled();
  });

  it('待开始详情页刷新后根据我的商品提醒列表回填已订阅状态', async () => {
    mockedAuctionApi.get.mockResolvedValueOnce({
      id: 12,
      product_id: 34,
      live_stream_id: 5,
      status: 0,
      current_price: 399,
      start_price: 0,
      start_time: '2026-06-05T01:40:00+08:00',
      end_time: new Date(Date.now() + 60_000).toISOString(),
    });
    mockedAuctionApi.getBids.mockResolvedValueOnce([]);
    mockedProductApi.get.mockResolvedValueOnce({
      id: 34,
      name: '手作陶瓷茶具套装',
      description: '即将开拍的家居生活商品',
      images: ['/tea-set.jpg'],
      rules: {
        start_price: 0,
        increment: 100,
      },
    });
    mockedProductReminderApi.list.mockResolvedValueOnce({
      items: [{ product_id: 34 }],
      total: 1,
    });

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

    expect(await screen.findByText('手作陶瓷茶具套装')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: '已订阅' })).toBeDisabled();
    expect(mockedProductReminderApi.subscribe).not.toHaveBeenCalled();
  });

  it('已结束详情页展示成交价和成交时间', async () => {
    mockedAuctionApi.get.mockResolvedValueOnce({
      id: 12,
      product_id: 34,
      live_stream_id: 5,
      status: 3,
      current_price: 399,
      start_price: 0,
      start_time: '2026-06-04T18:09:00+08:00',
      end_time: '2026-06-04T18:39:00+08:00',
    });
    mockedAuctionApi.getBids.mockResolvedValueOnce([
      { id: 1, user_id: 2, user_name: '张三', amount: 399, created_at: '2026-06-04T18:20:00+08:00' },
    ]);
    mockedProductApi.get.mockResolvedValueOnce({
      id: 34,
      name: '手作陶瓷茶具套装',
      description: '已成交商品',
      images: ['/tea-set.jpg'],
      rules: {
        start_price: 0,
        increment: 100,
      },
    });

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

    expect(await screen.findByText('手作陶瓷茶具套装')).toBeInTheDocument();
    expect(screen.getByText('成交价')).toBeInTheDocument();
    expect(screen.getAllByText('¥399').length).toBeGreaterThan(0);
    expect(screen.getByText(/成交时间/)).toBeInTheDocument();
    expect(screen.queryByText(/截止/)).not.toBeInTheDocument();
    const resultLink = screen.getByRole('link', { name: '查看竞拍结果' });
    expect(resultLink).toHaveAttribute('href', '/result?id=12');
    expect(resultLink.closest('footer')).toBeNull();
  });

  it('过期的进行中竞拍在详情页按已结束展示', async () => {
    mockedAuctionApi.get.mockResolvedValueOnce({
      id: 12,
      product_id: 34,
      live_stream_id: 5,
      status: 1,
      current_price: 399,
      start_price: 0,
      start_time: new Date(Date.now() - 3_600_000).toISOString(),
      end_time: new Date(Date.now() - 60_000).toISOString(),
    });
    mockedAuctionApi.getBids.mockResolvedValueOnce([]);
    mockedProductApi.get.mockResolvedValueOnce({
      id: 34,
      name: '已过期竞拍',
      description: '后端状态未及时归档，但时间已结束',
      images: ['/ended.jpg'],
      rules: {
        start_price: 0,
        increment: 100,
      },
    });

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

    expect(await screen.findByText('已过期竞拍')).toBeInTheDocument();
    expect(screen.getByText('已结束')).toBeInTheDocument();
    expect(screen.getByText('成交价')).toBeInTheDocument();
    expect(screen.getByText(/成交时间/)).toBeInTheDocument();
    expect(screen.queryByText('进行中')).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '参与竞拍' })).not.toBeInTheDocument();
  });
});
