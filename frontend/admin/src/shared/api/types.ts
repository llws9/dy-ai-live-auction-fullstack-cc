// 共享类型定义

// 分页响应
export interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
}

// 用户
export interface User {
  id: number;
  name: string;
  email: string;
  phone?: string;
  avatar?: string;
  role: number; // 0=普通用户, 1=商家/主播, 2=管理员
  created_at: string;
}

// 商品
export interface Product {
  id: number;
  name: string;
  description: string;
  images: string[];
  category_id?: number | null;
  category_name?: string;
  status: number; // 0=未发布, 1=已发布, 2=已下架
  created_at: string;
  updated_at: string;
  rules?: AuctionRule;
}

// 商品分类
export interface Category {
  id: number;
  name: string;
  code: string;
  status?: number;
}

// 竞拍规则
export interface AuctionRule {
  id: number;
  product_id: number;
  start_price: number;
  increment: number; // 加价幅度
  cap_price: number; // 封顶价
  duration: number; // 持续时间（秒）
  delay_duration: number; // 延时时间（秒）
  max_delay_time: number; // 最大延时次数
  trigger_delay_before: number; // 延时触发时间（秒）
}

// 竞拍场次
export interface Auction {
  id: number;
  product_id: number;
  product?: Product;
  live_stream_id: number;
  live_stream_name?: string;
  status: number; // 0=待开始, 1=进行中, 2=延时中, 3=已结束, 4=已取消
  current_price: number;
  winner_id?: number;
  winner_name?: string;
  start_time: string;
  end_time?: string;
  bid_count: number;
  created_at: string;
}

// 出价记录
export interface Bid {
  id: number;
  auction_id: number;
  user_id: number;
  user_name: string;
  price: number;
  rank: number;
  created_at: string;
}

// 竞拍结果
export interface AuctionResult {
  auction_id: number;
  winner_id: number;
  winner_name: string;
  final_price: number;
  end_time: string;
  bid_count: number;
}

// 订单
export interface Order {
  id: number;
  auction_id: number;
  product_id: number;
  product_name: string;
  product_image?: string;
  user_id: number;
  user_name?: string;
  status: number; // 0=待支付, 1=已支付, 2=已发货, 3=已完成, 4=已取消
  final_price: number;
  created_at: string;
  paid_at?: string;
  shipped_at?: string;
}

// 直播间
export interface LiveStream {
  id: number;
  name: string;
  streamer_id: number;
  streamer_name?: string;
  streamer_avatar?: string;
  status: number; // 0=未开播, 1=直播中, 2=已结束, 3=已封禁
  viewer_count: number;
  auction_count: number;
  created_at: string;
  is_followed?: boolean;
}

// 通知
export interface Notification {
  id: number;
  user_id: number;
  type: string; // auction_win, auction_end, price_update, etc.
  title: string;
  content: string;
  is_read: boolean;
  created_at: string;
}

// 统计概览
export interface StatisticsOverview {
  total_auctions: number;
  total_revenue: number;
  total_orders: number;
  total_users: number;
  ongoing_auctions: number;
  today_revenue: number;
}

// 竞拍统计
export interface AuctionStatistics {
  date: string;
  auction_count: number;
  bid_count: number;
  avg_price: number;
  success_rate: number;
}

// 收入统计
export interface RevenueStatistics {
  date: string;
  revenue: number;
  order_count: number;
  category?: string;
}

// 用户统计
export interface DailyUserStatistics {
  date: string;
  new_users: number;
  active_users: number;
}

export interface UserStatistics {
  total_users: number;
  active_users: number;
  new_users: number;
  paid_conversion_rate: number;
  daily_users: DailyUserStatistics[];
}

// WebSocket消息类型
export interface WSMessage {
  type: 'price_update' | 'countdown' | 'auction_end' | 'system' | 'ping' | 'pong';
  data?: {
    user_id?: number;
    user_name?: string;
    price?: number;
    rank?: number;
    remaining_ms?: number;
    winner_id?: number;
    final_price?: number;
  };
}

// API响应格式
export interface ApiResponse<T> {
  code: number;
  message?: string;
  msg?: string;
  data: T;
}
