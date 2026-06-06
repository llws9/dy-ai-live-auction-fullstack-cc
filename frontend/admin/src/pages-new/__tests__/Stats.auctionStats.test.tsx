import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Stats from '@/pages-new/Stats';
import { statisticsApi } from '@/shared/api';
import { useAuth } from '@/shared/auth';
import { ADMIN_ROLE, MERCHANT_ROLE } from '@/shared/auth/roles';

jest.mock('recharts', () => ({
  Area: () => null,
  AreaChart: ({ data }: { data?: unknown }) => <div data-testid="area-chart">{JSON.stringify(data)}</div>,
  Bar: () => null,
  BarChart: ({ data }: { data?: unknown }) => <div data-testid="bar-chart">{JSON.stringify(data)}</div>,
  CartesianGrid: () => null,
  Legend: () => null,
  Line: () => null,
  LineChart: ({ data }: { data?: unknown }) => <div data-testid="line-chart">{JSON.stringify(data)}</div>,
  ResponsiveContainer: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
  Tooltip: () => null,
  XAxis: () => null,
  YAxis: () => null,
}));

jest.mock('@/shared/api', () => ({
  statisticsApi: {
    getAuctionStats: jest.fn(),
    getRevenueStats: jest.fn(),
    getUserStats: jest.fn(),
  },
}));

jest.mock('@/shared/auth', () => ({
  useAuth: jest.fn(),
}));

const mockedStatisticsApi = jest.mocked(statisticsApi);
const mockedUseAuth = jest.mocked(useAuth);

function renderStats() {
  return render(
    <MemoryRouter initialEntries={['/stats/auction']}>
      <Routes>
        <Route path="/stats/:kind" element={<Stats />} />
      </Routes>
    </MemoryRouter>
  );
}

function renderStatsAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/stats/:kind" element={<Stats />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('Stats auction statistics', () => {
  let consoleErrorSpy: jest.SpyInstance;

  beforeEach(() => {
    jest.clearAllMocks();
    consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
    mockedUseAuth.mockReturnValue({
      user: { id: 1003, name: '系统管理员', email: 'admin@example.com', role: ADMIN_ROLE },
      token: 'token',
      isAuthenticated: true,
      isLoading: false,
      login: jest.fn(),
      logout: jest.fn(),
      refreshUser: jest.fn(),
    });
    mockedStatisticsApi.getRevenueStats.mockResolvedValue([]);
    mockedStatisticsApi.getUserStats.mockResolvedValue([]);
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  it('renders real auction statistics returned by API', async () => {
    mockedStatisticsApi.getAuctionStats.mockResolvedValue([
      { date: '2026-06-01', auction_count: 2, bid_count: 3, avg_price: 120, success_rate: 50 },
      { date: '2026-06-02', auction_count: 1, bid_count: 1, avg_price: 80, success_rate: 100 },
    ]);

    renderStats();

    await waitFor(() => expect(mockedStatisticsApi.getAuctionStats).toHaveBeenCalled());
    expect(await screen.findByText('3')).toBeInTheDocument();
    expect(screen.getByText('75.0%')).toBeInTheDocument();
    expect(screen.getByText('1.3')).toBeInTheDocument();
    expect(screen.getByTestId('bar-chart')).toHaveTextContent('"count":2');
  });

  it('does not render static fallback values when API fails', async () => {
    mockedStatisticsApi.getAuctionStats.mockRejectedValue(new Error('network error'));

    renderStats();

    await waitFor(() => expect(mockedStatisticsApi.getAuctionStats).toHaveBeenCalled());
    expect(screen.queryByText(/"count":35/)).not.toBeInTheDocument();
    expect(screen.getByText('0')).toBeInTheDocument();
    expect(screen.getByText('0.0%')).toBeInTheDocument();
  });

  it('keeps auction chart data when revenue statistics API fails', async () => {
    mockedStatisticsApi.getAuctionStats.mockResolvedValue([
      { date: '2026-06-01', auction_count: 2, bid_count: 3, avg_price: 120, success_rate: 50 },
    ]);
    mockedStatisticsApi.getRevenueStats.mockRejectedValue(new Error('revenue error'));

    renderStats();

    await waitFor(() => expect(mockedStatisticsApi.getRevenueStats).toHaveBeenCalled());
    expect(screen.getByTestId('bar-chart')).toHaveTextContent('"count":2');
    expect(screen.queryByText('暂无竞拍数据')).not.toBeInTheDocument();
  });

  it('shows an explicit empty state when auction statistics has no data', async () => {
    mockedStatisticsApi.getAuctionStats.mockResolvedValue([]);

    renderStats();

    await waitFor(() => expect(mockedStatisticsApi.getAuctionStats).toHaveBeenCalled());
    expect(screen.getByText('暂无竞拍数据')).toBeInTheDocument();
    expect(screen.getByText('当前统计周期内没有竞拍场次，图表为空不是渲染失败。')).toBeInTheDocument();
  });

  it('does not call platform-only user statistics API for merchants', async () => {
    mockedUseAuth.mockReturnValue({
      user: { id: 1002, name: '商家用户', email: 'merchant@example.com', role: MERCHANT_ROLE },
      token: 'token',
      isAuthenticated: true,
      isLoading: false,
      login: jest.fn(),
      logout: jest.fn(),
      refreshUser: jest.fn(),
    });
    mockedStatisticsApi.getAuctionStats.mockResolvedValue([]);

    renderStats();

    await waitFor(() => expect(mockedStatisticsApi.getAuctionStats).toHaveBeenCalled());
    expect(mockedStatisticsApi.getRevenueStats).toHaveBeenCalled();
    expect(mockedStatisticsApi.getUserStats).not.toHaveBeenCalled();
  });

  it('renders admin user statistics from object API contract', async () => {
    mockedStatisticsApi.getAuctionStats.mockResolvedValue([]);
    mockedStatisticsApi.getUserStats.mockResolvedValue({
      total_users: 120,
      active_users: 42,
      new_users: 9,
      paid_conversion_rate: 35.5,
      daily_users: [
        { date: '2026-06-01', new_users: 3, active_users: 12 },
        { date: '2026-06-02', new_users: 6, active_users: 18 },
      ],
    });

    renderStatsAt('/stats/user');

    await waitFor(() => expect(mockedStatisticsApi.getUserStats).toHaveBeenCalled());
    expect(screen.getByText('120')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText('35.5%')).toBeInTheDocument();
    expect(screen.getByTestId('line-chart')).toHaveTextContent('"new":3');
    expect(screen.getByTestId('line-chart')).toHaveTextContent('"active":18');
  });
});
