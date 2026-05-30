import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import OrderDetail from '../Detail';
import { orderApi } from '../../../services/api';
import { ApiError } from '../../../services/api';

jest.mock('../../../services/api', () => {
  class ApiError extends Error {
    status: number;
    code?: string;
    data?: unknown;
    constructor(message: string, status: number, code?: string, data?: unknown) {
      super(message);
      this.name = 'ApiError';
      this.status = status;
      this.code = code;
      this.data = data;
    }
  }
  return {
    __esModule: true,
    ApiError,
    orderApi: {
      get: jest.fn(),
    },
  };
});

jest.mock('../../../components/ThemeToggle', () => ({
  __esModule: true,
  default: () => null,
}));

const mockedOrderApi = orderApi as jest.Mocked<typeof orderApi>;

const renderAt = (id: string) =>
  render(
    <MemoryRouter
      initialEntries={[`/order/${id}`]}
      future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
    >
      <Routes>
        <Route path="/order/:id" element={<OrderDetail />} />
      </Routes>
    </MemoryRouter>
  );

const baseOrder = {
  id: 56,
  auction_id: 12,
  product_id: 34,
  winner_id: 9,
  final_price: 6800,
  status: 0,
  created_at: '2026-05-30T08:00:00Z',
};

describe('OrderDetail page (T3.5 / F-D1)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders loading state before data resolves', () => {
    let resolveFn: (value: unknown) => void = () => {};
    mockedOrderApi.get.mockReturnValue(
      new Promise((resolve) => {
        resolveFn = resolve;
      })
    );

    renderAt('56');

    expect(screen.getByRole('status')).toBeInTheDocument();
    // Resolve to avoid open handle
    resolveFn(baseOrder);
  });

  it('renders status badge "待支付" for status=0 and shows shipping placeholder', async () => {
    mockedOrderApi.get.mockResolvedValue({ ...baseOrder, status: 0 });

    renderAt('56');

    expect(await screen.findByText('待支付')).toBeInTheDocument();
    // 时间线：未支付/未发货占位
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
    // 金额展示（商品摘要 + 订单金额各一次）
    expect(screen.getAllByText('¥6,800').length).toBeGreaterThanOrEqual(1);
    // path id 透传
    expect(mockedOrderApi.get).toHaveBeenCalledWith(56);
  });

  it('renders status badge "已支付" for status=1 with paid_at', async () => {
    mockedOrderApi.get.mockResolvedValue({
      ...baseOrder,
      status: 1,
      paid_at: '2026-05-30T08:30:00Z',
    });

    renderAt('56');

    expect(await screen.findByText('已支付')).toBeInTheDocument();
  });

  it('renders status badge "已发货" for status=2 with shipped_at', async () => {
    mockedOrderApi.get.mockResolvedValue({
      ...baseOrder,
      status: 2,
      paid_at: '2026-05-30T08:30:00Z',
      shipped_at: '2026-05-31T09:00:00Z',
    });

    renderAt('56');

    expect(await screen.findByText('已发货')).toBeInTheDocument();
  });

  it('renders status badge "已完成" for status=3 with completed_at', async () => {
    mockedOrderApi.get.mockResolvedValue({
      ...baseOrder,
      status: 3,
      paid_at: '2026-05-30T08:30:00Z',
      shipped_at: '2026-05-31T09:00:00Z',
      completed_at: '2026-06-01T10:00:00Z',
    });

    renderAt('56');

    expect(await screen.findByText('已完成')).toBeInTheDocument();
  });

  it('renders 404 empty state when backend returns ApiError 404', async () => {
    mockedOrderApi.get.mockRejectedValue(new ApiError('订单不存在', 404, '404'));

    renderAt('999');

    expect(await screen.findByText('订单不存在')).toBeInTheDocument();
    // 返回按钮（页面正文中的"返回"按钮，名称完全匹配；PageHeader 的返回按钮 aria-label 同名，故用 getAllByRole 验证存在）
    expect(screen.getAllByRole('button', { name: '返回' }).length).toBeGreaterThanOrEqual(1);
  });

  it('renders error state with retry button on network failure (non-404)', async () => {
    mockedOrderApi.get.mockRejectedValueOnce(new ApiError('网络错误', 0, 'NETWORK_ERROR'));

    renderAt('56');

    expect(await screen.findByRole('button', { name: '重试' })).toBeInTheDocument();
    expect(screen.getByText(/加载失败/)).toBeInTheDocument();

    // 点击重试 → 第二次成功
    mockedOrderApi.get.mockResolvedValueOnce(baseOrder);
    fireEvent.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(mockedOrderApi.get).toHaveBeenCalledTimes(2));
    expect(await screen.findByText('待支付')).toBeInTheDocument();
  });

  it('does NOT render any pay button (支付链路不在本期)', async () => {
    mockedOrderApi.get.mockResolvedValue({ ...baseOrder, status: 0 });

    renderAt('56');

    expect(await screen.findByText('待支付')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /支付/ })).not.toBeInTheDocument();
  });

  it('contact-customer-service button shows toast placeholder when clicked', async () => {
    mockedOrderApi.get.mockResolvedValue({ ...baseOrder, status: 1, paid_at: '2026-05-30T08:30:00Z' });

    renderAt('56');

    fireEvent.click(await screen.findByRole('button', { name: '联系客服' }));

    expect(await screen.findByText('客服功能即将上线')).toBeInTheDocument();
  });
});
