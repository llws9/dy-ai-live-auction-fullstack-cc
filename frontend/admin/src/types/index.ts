// types/index.ts

// 商品状态
export enum ProductStatus {
  Draft = 0,
  Published = 1,
  Unpublished = 2,
}

// 商品
export interface Product {
  id: number;
  name: string;
  description: string;
  images: string[];
  status: ProductStatus;
  created_at: string;
}

// 竞拍规则
export interface AuctionRule {
  id: number;
  auction_id: number;
  start_price: number;
  increment: number;
  cap_price?: number;
  duration: number;
  delay_duration: number;
  max_delay_time: number;
  trigger_delay_before: number;
}

// 出价记录
export interface Bid {
  id: number;
  auction_id: number;
  user_id: number;
  amount: number;
  created_at: string;
}

// 竞拍
export interface Auction {
  id: number;
  product_id: number;
  status: number;
  current_price: number;
  winner_id?: number;
  start_time: string;
  end_time: string;
  delay_used: number;
  created_at: string;
}

// 订单
export interface Order {
  id: number;
  auction_id: number;
  product_id: number;
  winner_id: number;
  final_price: number;
  status: number;
  created_at: string;
}

// 用户
export interface User {
  id: number;
  name: string;
  avatar: string;
  created_at: string;
}

// API 响应
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// 分页响应
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}
