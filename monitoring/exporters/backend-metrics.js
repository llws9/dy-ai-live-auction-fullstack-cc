const promClient = require('prom-client');

// 创建Registry
const register = new promClient.Registry();

// 添加默认标签
register.setDefaultLabels({
  app: 'live-auction-backend',
  environment: process.env.NODE_ENV || 'development'
});

// ==================== 系统指标 ====================
// 这些指标由 prom-client 的默认指标收集器自动收集
promClient.collectDefaultMetrics({ register });

// ==================== HTTP请求指标 ====================
const httpRequestDuration = new promClient.Histogram({
  name: 'http_request_duration_seconds',
  help: 'HTTP请求响应时间',
  labelNames: ['method', 'path', 'status'],
  buckets: [0.01, 0.05, 0.1, 0.3, 0.5, 1, 2, 5, 10],
  registers: [register]
});

const httpRequestTotal = new promClient.Counter({
  name: 'http_requests_total',
  help: 'HTTP请求总数',
  labelNames: ['method', 'path', 'status', 'service'],
  registers: [register]
});

const httpRequestsInProgress = new promClient.Gauge({
  name: 'http_requests_in_progress',
  help: '当前正在处理的HTTP请求数',
  labelNames: ['method', 'path'],
  registers: [register]
});

// ==================== 业务指标 ====================
// 在线用户数
const onlineUsers = new promClient.Gauge({
  name: 'live_auction_online_users',
  help: '当前在线用户数',
  registers: [register]
});

// 活跃竞拍数
const activeAuctions = new promClient.Gauge({
  name: 'live_auction_active_auctions',
  help: '当前活跃的竞拍数量',
  registers: [register]
});

// 出价计数器
const bidsTotal = new promClient.Counter({
  name: 'live_auction_bids_total',
  help: '出价总数',
  labelNames: ['auction_id', 'user_id'],
  registers: [register]
});

// 订单计数器
const ordersTotal = new promClient.Counter({
  name: 'live_auction_orders_total',
  help: '订单总数',
  labelNames: ['auction_id', 'status'],
  registers: [register]
});

// 收入计数器
const revenueTotal = new promClient.Counter({
  name: 'live_auction_revenue_total',
  help: '总收入',
  labelNames: ['auction_id', 'currency'],
  registers: [register]
});

// 用户行为计数器
const userActionsTotal = new promClient.Counter({
  name: 'live_auction_user_actions_total',
  help: '用户行为总数',
  labelNames: ['action', 'user_id'],
  registers: [register]
});

// 浏览量计数器
const viewsTotal = new promClient.Counter({
  name: 'live_auction_views_total',
  help: '浏览量总数',
  labelNames: ['auction_id'],
  registers: [register]
});

// 参与人数计数器
const participantsTotal = new promClient.Counter({
  name: 'live_auction_participants_total',
  help: '参与人数总数',
  labelNames: ['auction_id'],
  registers: [register]
});

// 活跃用户数
const activeUsers = new promClient.Gauge({
  name: 'live_auction_active_users',
  help: '活跃用户数',
  labelNames: ['user_type'],
  registers: [register]
});

// ==================== 数据库指标 ====================
const dbQueryDuration = new promClient.Histogram({
  name: 'db_query_duration_seconds',
  help: '数据库查询响应时间',
  labelNames: ['query_type', 'table'],
  buckets: [0.01, 0.05, 0.1, 0.3, 0.5, 1, 2, 5],
  registers: [register]
});

const dbConnectionsActive = new promClient.Gauge({
  name: 'db_connections_active',
  help: '当前活跃的数据库连接数',
  registers: [register]
});

const dbConnectionsIdle = new promClient.Gauge({
  name: 'db_connections_idle',
  help: '当前空闲的数据库连接数',
  registers: [register]
});

const dbSlowQueries = new promClient.Counter({
  name: 'db_slow_queries_total',
  help: '慢查询总数',
  labelNames: ['query_type', 'table'],
  registers: [register]
});

// ==================== Redis指标 ====================
const redisCacheHits = new promClient.Counter({
  name: 'redis_cache_hits_total',
  help: 'Redis缓存命中次数',
  registers: [register]
});

const redisCacheMisses = new promClient.Counter({
  name: 'redis_cache_misses_total',
  help: 'Redis缓存未命中次数',
  registers: [register]
});

const redisOperations = new promClient.Histogram({
  name: 'redis_operation_duration_seconds',
  help: 'Redis操作响应时间',
  labelNames: ['operation'],
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5],
  registers: [register]
});

// ==================== WebSocket指标 ====================
const wsConnections = new promClient.Gauge({
  name: 'websocket_connections_current',
  help: '当前WebSocket连接数',
  registers: [register]
});

const wsMessagesTotal = new promClient.Counter({
  name: 'websocket_messages_total',
  help: 'WebSocket消息总数',
  labelNames: ['type', 'direction'],
  registers: [register]
});

const wsErrors = new promClient.Counter({
  name: 'websocket_errors_total',
  help: 'WebSocket错误总数',
  labelNames: ['type'],
  registers: [register]
});

// ==================== 中间件函数 ====================
/**
 * HTTP请求监控中间件
 */
function httpMetricsMiddleware(req, res, next) {
  const start = Date.now();
  const path = req.route ? req.route.path : req.path;

  // 增加正在处理的请求计数
  httpRequestsInProgress.inc({ method: req.method, path });

  // 监听响应完成事件
  res.on('finish', () => {
    const duration = (Date.now() - start) / 1000;

    // 记录响应时间
    httpRequestDuration.observe(
      { method: req.method, path, status: res.statusCode },
      duration
    );

    // 增加请求计数
    httpRequestTotal.inc({
      method: req.method,
      path,
      status: res.statusCode,
      service: 'backend'
    });

    // 减少正在处理的请求计数
    httpRequestsInProgress.dec({ method: req.method, path });
  });

  next();
}

/**
 * 数据库查询监控包装函数
 */
function trackDbQuery(queryType, table, queryFn) {
  const end = dbQueryDuration.startTimer({ query_type: queryType, table });

  return queryFn()
    .then(result => {
      end();
      return result;
    })
    .catch(error => {
      end();
      // 如果查询时间超过阈值,记录为慢查询
      throw error;
    });
}

/**
 * Redis操作监控包装函数
 */
function trackRedisOperation(operation, operationFn) {
  const end = redisOperations.startTimer({ operation });

  return operationFn()
    .then(result => {
      end();
      return result;
    })
    .catch(error => {
      end();
      throw error;
    });
}

// ==================== 业务指标更新函数 ====================
/**
 * 更新在线用户数
 */
function updateOnlineUsers(count) {
  onlineUsers.set(count);
}

/**
 * 更新活跃竞拍数
 */
function updateActiveAuctions(count) {
  activeAuctions.set(count);
}

/**
 * 记录出价
 */
function recordBid(auctionId, userId, bidAmount) {
  bidsTotal.inc({ auction_id: auctionId, user_id: userId });
}

/**
 * 记录订单
 */
function recordOrder(auctionId, status, amount) {
  ordersTotal.inc({ auction_id: auctionId, status });
  revenueTotal.inc({ auction_id: auctionId, currency: 'CNY' }, amount);
}

/**
 * 记录用户行为
 */
function recordUserAction(action, userId) {
  userActionsTotal.inc({ action, user_id: userId });
}

/**
 * 记录浏览
 */
function recordView(auctionId) {
  viewsTotal.inc({ auction_id: auctionId });
}

/**
 * 记录参与
 */
function recordParticipant(auctionId) {
  participantsTotal.inc({ auction_id: auctionId });
}

/**
 * 更新活跃用户数
 */
function updateActiveUsers(counts) {
  // counts = { 'bidder': 100, 'viewer': 500, 'seller': 10 }
  Object.entries(counts).forEach(([userType, count]) => {
    activeUsers.set({ user_type: userType }, count);
  });
}

// ==================== 数据库指标更新函数 ====================
/**
 * 更新数据库连接状态
 */
function updateDbConnections(active, idle) {
  dbConnectionsActive.set(active);
  dbConnectionsIdle.set(idle);
}

/**
 * 记录慢查询
 */
function recordSlowQuery(queryType, table) {
  dbSlowQueries.inc({ query_type: queryType, table });
}

// ==================== Redis指标更新函数 ====================
/**
 * 记录Redis缓存命中
 */
function recordCacheHit() {
  redisCacheHits.inc();
}

/**
 * 记录Redis缓存未命中
 */
function recordCacheMiss() {
  redisCacheMisses.inc();
}

// ==================== WebSocket指标更新函数 ====================
/**
 * 更新WebSocket连接数
 */
function updateWsConnections(count) {
  wsConnections.set(count);
}

/**
 * 记录WebSocket消息
 */
function recordWsMessage(type, direction) {
  wsMessagesTotal.inc({ type, direction });
}

/**
 * 记录WebSocket错误
 */
function recordWsError(type) {
  wsErrors.inc({ type });
}

// ==================== 指标端点 ====================
/**
 * 获取指标数据
 */
async function getMetrics() {
  return await register.metrics();
}

/**
 * 获取内容类型
 */
function getContentType() {
  return register.contentType;
}

module.exports = {
  // Registry
  register,

  // 中间件
  httpMetricsMiddleware,

  // 业务指标更新函数
  updateOnlineUsers,
  updateActiveAuctions,
  recordBid,
  recordOrder,
  recordUserAction,
  recordView,
  recordParticipant,
  updateActiveUsers,

  // 数据库指标更新函数
  trackDbQuery,
  updateDbConnections,
  recordSlowQuery,

  // Redis指标更新函数
  trackRedisOperation,
  recordCacheHit,
  recordCacheMiss,

  // WebSocket指标更新函数
  updateWsConnections,
  recordWsMessage,
  recordWsError,

  // 指标端点
  getMetrics,
  getContentType
};
