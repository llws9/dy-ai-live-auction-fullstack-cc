import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Stats from '@/pages-new/Stats';
import { useAuth } from '@/shared/auth';
import { ADMIN_ROLE, MERCHANT_ROLE } from '@/shared/auth/roles';

jest.mock('recharts', () => ({
  Area: () => null,
  AreaChart: () => <div data-testid="area-chart" />,
  Bar: () => null,
  BarChart: () => <div data-testid="bar-chart" />,
  CartesianGrid: () => null,
  Legend: () => null,
  Line: () => null,
  LineChart: () => <div data-testid="line-chart" />,
  ResponsiveContainer: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
  Tooltip: () => null,
  XAxis: () => null,
  YAxis: () => null,
}));

jest.mock('@/shared/api', () => ({
  statisticsApi: {
    getAuctionStats: jest.fn(() => new Promise(() => {})),
    getRevenueStats: jest.fn(() => new Promise(() => {})),
    getUserStats: jest.fn(() => new Promise(() => {})),
  },
}));

jest.mock('@/shared/auth', () => ({
  useAuth: jest.fn(),
}));

const mockedUseAuth = jest.mocked(useAuth);

function renderStatsAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/stats/:kind" element={<Stats />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('Stats route tabs', () => {
  beforeEach(() => {
    mockedUseAuth.mockReturnValue({
      user: { id: 1003, name: '系统管理员', email: 'admin@example.com', role: ADMIN_ROLE },
      token: 'token',
      isAuthenticated: true,
      isLoading: false,
      login: jest.fn(),
      logout: jest.fn(),
      refreshUser: jest.fn(),
    });
  });

  it('activates the revenue tab when the current route is /stats/revenue', () => {
    renderStatsAt('/stats/revenue');

    expect(screen.getByRole('tab', { name: '收入统计' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tab', { name: '竞拍统计' })).toHaveAttribute('aria-selected', 'false');
    expect(screen.getAllByText('全平台维度').length).toBeGreaterThan(0);
  });

  it('activates the user tab when the current route is /stats/user', () => {
    renderStatsAt('/stats/user');

    expect(screen.getByRole('tab', { name: '用户统计' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tab', { name: '竞拍统计' })).toHaveAttribute('aria-selected', 'false');
  });

  it('hides platform-only user statistics tab for merchants', () => {
    mockedUseAuth.mockReturnValue({
      user: { id: 1002, name: '商家用户', email: 'merchant@example.com', role: MERCHANT_ROLE },
      token: 'token',
      isAuthenticated: true,
      isLoading: false,
      login: jest.fn(),
      logout: jest.fn(),
      refreshUser: jest.fn(),
    });

    renderStatsAt('/stats/auction');

    expect(screen.queryByRole('tab', { name: '用户统计' })).not.toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '竞拍统计' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '收入统计' })).toBeInTheDocument();
    expect(screen.getAllByText('商家维度').length).toBeGreaterThan(0);
  });
});
