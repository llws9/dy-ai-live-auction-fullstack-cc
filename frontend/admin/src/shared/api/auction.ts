// 竞拍API

import { get, post, put, buildQuery } from './request';
import { Auction, Bid, AuctionResult, PaginatedResponse } from './types';
import { normalizeAuctionListResponse, normalizeAuctionText } from './auctionEncoding';

export interface AuctionListParams {
  status?: number;
  live_stream_id?: number;
  live_stream_name?: string;
  search?: string;
  page?: number;
  page_size?: number;
}

export interface PlaceBidData {
  amount: number;
}

export const auctionApi = {
  // 获取竞拍列表
  list: (params?: AuctionListParams) => {
    const query = buildQuery(params || {});
    return get<PaginatedResponse<Auction>>(`/auctions?${query}`)
      .then(normalizeAuctionListResponse);
  },

  // 获取竞拍详情
  get: (id: number) => get<Auction>(`/auctions/${id}`).then(normalizeAuctionText),

  // 创建竞拍场次
  create: (data: { product_id: number; duration: number; live_stream_id?: number }) => post<Auction>('/auctions', data).then(normalizeAuctionText),

  // 获取出价记录
  getBids: (id: number) => get<Bid[]>(`/auctions/${id}/bids`),

  // 获取竞拍排名
  getRanking: (id: number) => get<Bid[]>(`/auctions/${id}/ranking`),

  // 用户出价
  placeBid: (id: number, amount: number) => post<Bid>(`/auctions/${id}/bids`, { amount }),

  // 取消竞拍
  cancel: (id: number) => put<void>(`/auctions/${id}/cancel`),

  // 获取竞拍结果
  getResult: (id: number) => get<AuctionResult>(`/auctions/${id}/result`),
};
