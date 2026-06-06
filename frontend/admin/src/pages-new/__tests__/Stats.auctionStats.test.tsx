import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Stats from '@/pages-new/Stats';
import { statisticsApi } from '@/shared/api';

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

const mockedStatisticsApi = jest.mocked(statisticsApi);

function renderStats() {
  return render(
    <MemoryRouter initialEntries={['/stats/auction']}>
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
});
