// 统计API

import { get, buildQuery } from './request';
import { StatisticsOverview, AuctionStatistics, RevenueStatistics, UserStatistics } from './types';

export interface StatisticsParams {
  start_date?: string; // YYYY-MM-DD
  end_date?: string;   // YYYY-MM-DD
  category?: string;
  group_by?: string;   // day, week, month, category
}

export const statisticsApi = {
  // 获取数据概览
  getOverview: () => get<StatisticsOverview>('/statistics/overview'),

  // 获取竞拍统计
  getAuctionStats: (params?: StatisticsParams) => {
    const query = buildQuery(params || {});
    return get<AuctionStatistics[]>(`/statistics/auctions?${query}`);
  },

  // 获取收入统计
  getRevenueStats: (params?: StatisticsParams) => {
    const query = buildQuery(params || {});
    return get<RevenueStatistics[]>(`/statistics/revenue?${query}`);
  },

  // 获取用户统计
  getUserStats: (params?: StatisticsParams) => {
    const query = buildQuery(params || {});
    return get<UserStatistics>(`/statistics/users?${query}`);
  },
};
