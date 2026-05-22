// 订单相关类型
export interface Order {
  id: number;
  auction_id: number;
  product_id: number;
  winner_id: number;
  final_price: number;
  status: OrderStatus;
  paid_at?: string;
  shipped_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export enum OrderStatus {
  Pending = 0,
  Paid = 1,
  Shipped = 2,
  Completed = 3,
}

// 商品相关类型
export interface Product {
  id: number;
  name: string;
  description: string;
  images: string[];
  status: ProductStatus;
  created_at: string;
}

export enum ProductStatus {
  Draft = 0,
  Published = 1,
}

// 竞拍相关类型
export interface Auction {
  id: number;
  product_id: number;
  status: AuctionStatus;
  current_price: number;
  winner_id?: number;
  start_time: string;
  end_time: string;
  delay_used: number;
  created_at: string;
}

export enum AuctionStatus {
  Pending = 0,
  Ongoing = 1,
  Delayed = 2,
  Ended = 3,
  Cancelled = 4,
}

// 出价相关类型
export interface Bid {
  id: number;
  auction_id: number;
  user_id: number;
  amount: number;
  created_at: string;
}

// 排名相关类型
export interface RankingItem {
  rank: number;
  userId: number;
  userName: string;
  userAvatar?: string;
  amount: number;
  bidTime: string;
}

// 用户相关类型
export interface User {
  id: number;
  name: string;
  avatar?: string;
  role: UserRole;
  created_at: string;
}

export enum UserRole {
  User = 0,
  Admin = 1,
}

// WebSocket 消息类型
export interface WebSocketMessage {
  type: MessageType;
  data: any;
}

export enum MessageType {
  // Server -> Client
  BidPlaced = 'bid_placed',
  RankUpdate = 'rank_update',
  Overtaken = 'overtaken',
  DelayTriggered = 'delay_triggered',
  AuctionEnded = 'auction_ended',
  TimeSync = 'time_sync',
  SyncResponse = 'sync_response',
  Pong = 'pong',

  // Client -> Server
  Ping = 'ping',
  SyncRequest = 'sync_request',
}

// 用户历史记录
export interface UserHistoryItem {
  auction_id: number;
  product_name: string;
  final_price: number;
  is_winner: boolean;
  bid_count: number;
  created_at: string;
}

// API 响应类型
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}

// 分页请求参数
export interface PaginationParams {
  page?: number;
  pageSize?: number;
}

// 分页响应数据
export interface PaginatedResponse<T> {
  total: number;
  page: number;
  pageSize: number;
  items: T[];
}
