import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Dashboard from '@/pages-new/Dashboard';
import { AuthProvider } from '@/shared/auth';

jest.mock('recharts', () => ({
  Area: () => null,
  AreaChart: () => <div data-testid="area-chart" />,
  CartesianGrid: () => null,
  Cell: () => null,
  Legend: () => null,
  Line: () => null,
  Pie: () => null,
  PieChart: () => <div data-testid="pie-chart" />,
  ResponsiveContainer: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
  Tooltip: () => null,
  XAxis: () => null,
  YAxis: () => null,
}));

jest.mock('@/shared/api', () => ({
  authApi: {
    getCurrentUser: jest.fn(),
  },
  liveStreamApi: {
    start: jest.fn(),
  },
  statisticsApi: {
    getOverview: jest.fn().mockResolvedValue({
      total_auctions: 42,
      ongoing_auctions: 7,
      total_revenue: 128000,
      total_orders: 35,
      total_users: 14,
      active_users: 9,
      today_revenue: 5600,
      success_rate: 0.83,
    }),
    getRevenueStats: jest.fn().mockResolvedValue([
      { date: '2026-06-01', revenue: 1200, order_count: 3, category: '珠宝' },
      { date: '2026-06-02', revenue: 2400, order_count: 5, category: '数码' },
    ]),
  },
}));

function renderDashboardWithRole(role: number) {
  localStorage.setItem('admin_auth_token', 'token');
  localStorage.setItem('admin_auth_user', JSON.stringify({
    id: role === 2 ? 1003 : 1002,
    name: role === 2 ? '系统管理员' : '商家用户',
    email: role === 2 ? 'admin@example.com' : 'merchant@example.com',
    role,
    created_at: '2026-06-05T00:00:00Z',
  }));

  return render(
    <MemoryRouter initialEntries={['/dashboard']}>
      <AuthProvider>
        <Dashboard />
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('Dashboard role visibility', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('shows platform-level dashboard without merchant operation buttons for admins', async () => {
    renderDashboardWithRole(2);

    expect(await screen.findByRole('heading', { name: /欢迎，系统管理员/ })).toBeInTheDocument();
    expect(screen.queryAllByRole('button', { name: /发布商品/ })).toHaveLength(0);
    expect(screen.queryAllByRole('button', { name: /开启直播/ })).toHaveLength(0);

    expect(screen.getByText('全站竞拍单量')).toBeInTheDocument();
    expect(screen.getByText('平台累计 GMV')).toBeInTheDocument();
    expect(screen.getByText('注册用户数')).toBeInTheDocument();
    expect(screen.getByText('近7日活跃用户')).toBeInTheDocument();
    expect(screen.getByText('平台交易趋势')).toBeInTheDocument();
    expect(screen.getByText('平台类目 GMV')).toBeInTheDocument();
    expect(screen.getByText('平台治理看板')).toBeInTheDocument();
    expect(screen.getByText('平台运营入口')).toBeInTheDocument();
  });

  it('keeps merchant operation buttons and business metrics for merchants', async () => {
    renderDashboardWithRole(1);

    expect(await screen.findByRole('heading', { name: /欢迎，商家用户/ })).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: /发布商品/ }).length).toBeGreaterThan(0);
    expect(screen.queryByRole('button', { name: /开启直播|开始直播/ })).not.toBeInTheDocument();

    expect(screen.getByText('总收入')).toBeInTheDocument();
    expect(screen.getByText('参与用户')).toBeInTheDocument();
    expect(screen.getByText('今日成交')).toBeInTheDocument();
    expect(screen.getByText('近期趋势')).toBeInTheDocument();
    expect(screen.getByText('收入构成')).toBeInTheDocument();
  });
});
