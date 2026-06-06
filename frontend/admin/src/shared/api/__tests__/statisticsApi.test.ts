import { statisticsApi } from '..';
import { get } from '../request';

jest.mock('../request', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  del: jest.fn(),
  buildQuery: (params: Record<string, string | number | undefined>) =>
    new URLSearchParams(
      Object.entries(params)
        .filter(([, value]) => value !== undefined)
        .map(([key, value]) => [key, String(value)])
    ).toString(),
  ApiError: class ApiError extends Error {},
  setToastFunction: jest.fn(),
}));

describe('statisticsApi response normalization', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('returns daily revenue arrays for dashboard trend charts', async () => {
    const dailyRevenue = [
      { date: '2026-06-06', revenue: 1200 },
    ];
    (get as jest.Mock).mockResolvedValue({
      total_revenue: 1200,
      daily_revenue: dailyRevenue,
      category_distribution: [],
    });

    const result = await statisticsApi.getRevenueStats({ group_by: 'day' });

    expect(get).toHaveBeenCalledWith('/statistics/revenue?group_by=day');
    expect(result).toEqual(dailyRevenue);
  });

  it('returns category distribution arrays for dashboard composition charts', async () => {
    const categoryDistribution = [
      { category: '珠宝名表', revenue: 900 },
    ];
    (get as jest.Mock).mockResolvedValue({
      total_revenue: 900,
      daily_revenue: [],
      category_distribution: categoryDistribution,
    });

    const result = await statisticsApi.getRevenueStats({ group_by: 'category' });

    expect(get).toHaveBeenCalledWith('/statistics/revenue?group_by=category');
    expect(result).toEqual(categoryDistribution);
  });

  it('returns user statistics object contract unchanged', async () => {
    const userStats = {
      total_users: 120,
      active_users: 42,
      new_users: 9,
      paid_conversion_rate: 35.5,
      daily_users: [
        { date: '2026-06-01', new_users: 3, active_users: 12 },
      ],
    };
    (get as jest.Mock).mockResolvedValue(userStats);

    const result = await statisticsApi.getUserStats({ start_date: '2026-06-01', end_date: '2026-06-07' });

    expect(get).toHaveBeenCalledWith('/statistics/users?start_date=2026-06-01&end_date=2026-06-07');
    expect(result).toEqual(userStats);
  });
});
