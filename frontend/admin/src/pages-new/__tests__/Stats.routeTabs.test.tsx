import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Stats from '@/pages-new/Stats';

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
    getRevenueStats: jest.fn(),
    getUserStats: jest.fn(),
  },
}));

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
  it('activates the revenue tab when the current route is /stats/revenue', () => {
    renderStatsAt('/stats/revenue');

    expect(screen.getByRole('tab', { name: '收入统计' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tab', { name: '竞拍统计' })).toHaveAttribute('aria-selected', 'false');
  });

  it('activates the user tab when the current route is /stats/user', () => {
    renderStatsAt('/stats/user');

    expect(screen.getByRole('tab', { name: '用户统计' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tab', { name: '竞拍统计' })).toHaveAttribute('aria-selected', 'false');
  });
});
