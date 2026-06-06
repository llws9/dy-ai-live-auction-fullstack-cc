import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import OrderList from '../List';
import { ThemeProvider } from '../../../store/themeContext';
import { orderApi } from '../../../services/api';

jest.mock('../../../services/api', () => ({
  orderApi: {
    list: jest.fn(),
  },
}));

jest.mock('../../../components/ThemeToggle', () => ({
  __esModule: true,
  default: () => null,
}));

const renderPage = () =>
  render(
    <ThemeProvider>
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <OrderList />
      </MemoryRouter>
    </ThemeProvider>
  );

const mockedOrderApi = orderApi as jest.Mocked<typeof orderApi>;

describe('OrderList page', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    mockedOrderApi.list.mockResolvedValue({
      list: [
        {
          id: 56,
          auction_id: 12,
          product_id: 34,
          product_name: '山海鎏金香炉',
          product_image: 'https://cdn.example.com/products/incense-burner.jpg',
          seller_name: '山海商家',
          final_price: '6800.00',
          status: 0,
          created_at: '2026-06-06T17:00:21+08:00',
        },
        {
          id: 57,
          auction_id: 18,
          product_id: 35,
          final_price: '4200.00',
          status: 1,
          created_at: '2026-06-05T21:14:00+08:00',
        },
        {
          id: 58,
          auction_id: 21,
          product_id: 36,
          final_price: '12800.00',
          status: 2,
          created_at: '2026-06-04T19:35:00+08:00',
        },
      ],
      total: 3,
    });
  });

  it('loads backend orders and renders the accepted card design', async () => {
    renderPage();

    expect(screen.getByRole('heading', { name: '我的订单' })).toBeInTheDocument();
    expect(await screen.findByText('山海鎏金香炉')).toBeInTheDocument();
    expect(screen.getByText('山海商家')).toBeInTheDocument();
    expect(screen.getByAltText('山海鎏金香炉')).toHaveAttribute('src', 'https://cdn.example.com/products/incense-burner.jpg');
    expect(screen.getAllByText('AUCTION ORDER')[0]).toBeInTheDocument();
    expect(mockedOrderApi.list).toHaveBeenCalledWith({ page: 1, page_size: 20 });
    expect(screen.getByRole('button', { name: '待支付' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '待发货' })).toBeInTheDocument();
    expect(screen.getByText('06/06')).toBeInTheDocument();
    expect(screen.getByLabelText('成交价 ¥6,800')).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: '查看订单' })[0]).toHaveAttribute('href', '/order/56');
    expect(screen.getAllByText('待支付').length).toBeGreaterThanOrEqual(3);
    expect(screen.queryByText('订单 #56')).not.toBeInTheDocument();
    expect(screen.queryByText('查看订单 #56')).not.toBeInTheDocument();
    expect(screen.queryByText('等待买家确认支付')).not.toBeInTheDocument();
  });

  it('filters backend orders by status without navigating away', async () => {
    renderPage();

    expect(await screen.findByText('山海鎏金香炉')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '待发货' }));

    expect(screen.getByText('商品 #35')).toBeInTheDocument();
    expect(screen.queryByText('山海鎏金香炉')).not.toBeInTheDocument();
  });

  it('shows a user-friendly empty state when a status has no orders', async () => {
    renderPage();

    expect(await screen.findByText('山海鎏金香炉')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '已完成' }));

    expect(screen.getByText('还没有已完成订单')).toBeInTheDocument();
    expect(screen.getByText('后续订单完成后，会自动归档到这里。')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '去看直播竞拍' })).toHaveAttribute('href', '/');
  });

  it('shows retry state when backend order list fails', async () => {
    mockedOrderApi.list.mockRejectedValue(new Error('network down'));

    renderPage();

    expect(await screen.findByText('订单暂时无法加载')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重试' })).toBeInTheDocument();
  });

  it('loads the next backend page when more orders are available', async () => {
    mockedOrderApi.list
      .mockResolvedValueOnce({
        list: [
          {
            id: 56,
            auction_id: 12,
            product_id: 34,
            final_price: '6800.00',
            status: 0,
            created_at: '2026-06-06T17:00:21+08:00',
          },
        ],
        total: 21,
      })
      .mockResolvedValueOnce({
        list: [
          {
            id: 59,
            auction_id: 22,
            product_id: 41,
            final_price: '9900.00',
            status: 3,
            created_at: '2026-06-03T17:00:21+08:00',
          },
        ],
        total: 21,
      });

    renderPage();

    expect(await screen.findByText('商品 #34')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '加载更多' }));

    expect(await screen.findByText('商品 #41')).toBeInTheDocument();
    expect(mockedOrderApi.list).toHaveBeenLastCalledWith({ page: 2, page_size: 20 });
  });
});
