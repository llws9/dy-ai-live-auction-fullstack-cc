const WebSocket = require('ws');
const promClient = require('prom-client');

// 创建Registry
const register = new promClient.Registry();

// 添加默认标签
register.setDefaultLabels({
  app: 'live-auction-websocket',
  environment: process.env.NODE_ENV || 'development'
});

// ==================== WebSocket指标 ====================
// 连接数
const wsConnections = new promClient.Gauge({
  name: 'websocket_connections_current',
  help: '当前WebSocket连接数',
  registers: [register]
});

// 总连接数
const wsConnectionsTotal = new promClient.Counter({
  name: 'websocket_connections_total',
  help: 'WebSocket连接总数',
  registers: [register]
});

// 断开连接数
const wsDisconnectionsTotal = new promClient.Counter({
  name: 'websocket_disconnections_total',
  help: 'WebSocket断开连接总数',
  labelNames: ['reason'],
  registers: [register]
});

// 消息计数
const wsMessagesTotal = new promClient.Counter({
  name: 'websocket_messages_total',
  help: 'WebSocket消息总数',
  labelNames: ['type', 'direction'],
  registers: [register]
});

// 消息大小
const wsMessageSize = new promClient.Histogram({
  name: 'websocket_message_size_bytes',
  help: 'WebSocket消息大小',
  labelNames: ['type', 'direction'],
  buckets: [100, 500, 1000, 5000, 10000, 50000, 100000],
  registers: [register]
});

// 消息处理时间
const wsMessageDuration = new promClient.Histogram({
  name: 'websocket_message_duration_seconds',
  help: 'WebSocket消息处理时间',
  labelNames: ['type'],
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2],
  registers: [register]
});

// 错误计数
const wsErrorsTotal = new promClient.Counter({
  name: 'websocket_errors_total',
  help: 'WebSocket错误总数',
  labelNames: ['type'],
  registers: [register]
});

// 重连次数
const wsReconnectsTotal = new promClient.Counter({
  name: 'websocket_reconnects_total',
  help: 'WebSocket重连总数',
  registers: [register]
});

// ==================== 竞拍房间指标 ====================
// 活跃房间数
const activeRooms = new promClient.Gauge({
  name: 'websocket_active_rooms',
  help: '当前活跃的竞拍房间数',
  registers: [register]
});

// 房间用户数
const roomUsers = new promClient.Gauge({
  name: 'websocket_room_users',
  help: '每个房间的用户数',
  labelNames: ['auction_id'],
  registers: [register]
});

// 房间消息数
const roomMessagesTotal = new promClient.Counter({
  name: 'websocket_room_messages_total',
  help: '每个房间的消息总数',
  labelNames: ['auction_id', 'type'],
  registers: [register]
});

// ==================== 竞拍实时指标 ====================
// 实时出价
const liveBids = new promClient.Counter({
  name: 'live_auction_bids_total',
  help: '实时出价总数',
  labelNames: ['auction_id'],
  registers: [register]
});

// 实时出价金额
const liveBidAmount = new promClient.Histogram({
  name: 'live_auction_bid_amount',
  help: '实时出价金额分布',
  labelNames: ['auction_id'],
  buckets: [100, 500, 1000, 5000, 10000, 50000, 100000, 500000],
  registers: [register]
});

// 出价延迟
const bidLatency = new promClient.Histogram({
  name: 'live_auction_bid_latency_seconds',
  help: '出价到广播延迟',
  labelNames: ['auction_id'],
  buckets: [0.01, 0.05, 0.1, 0.2, 0.5, 1, 2],
  registers: [register]
});

// ==================== 连接池指标 ====================
const connectionPoolSize = new promClient.Gauge({
  name: 'websocket_pool_size',
  help: 'WebSocket连接池大小',
  registers: [register]
});

const connectionPoolAvailable = new promClient.Gauge({
  name: 'websocket_pool_available',
  help: '可用连接数',
  registers: [register]
});

// ==================== 监控类 ====================
class WebSocketMonitor {
  constructor() {
    this.connections = new Map();
    this.rooms = new Map();
  }

  /**
   * 记录新连接
   */
  recordConnection(ws, userId) {
    wsConnections.inc();
    wsConnectionsTotal.inc();

    const connectionId = `${userId}_${Date.now()}`;
    this.connections.set(connectionId, {
      ws,
      userId,
      connectedAt: Date.now(),
      rooms: new Set()
    });

    return connectionId;
  }

  /**
   * 记录断开连接
   */
  recordDisconnection(connectionId, reason = 'normal') {
    const connection = this.connections.get(connectionId);
    if (connection) {
      wsConnections.dec();
      wsDisconnectionsTotal.inc({ reason });

      // 从所有房间移除
      connection.rooms.forEach(roomId => {
        this.leaveRoom(connectionId, roomId);
      });

      this.connections.delete(connectionId);
    }
  }

  /**
   * 记录消息
   */
  recordMessage(type, direction, size, duration = null) {
    wsMessagesTotal.inc({ type, direction });
    wsMessageSize.observe({ type, direction }, size);

    if (duration !== null) {
      wsMessageDuration.observe({ type }, duration);
    }
  }

  /**
   * 记录错误
   */
  recordError(type) {
    wsErrorsTotal.inc({ type });
  }

  /**
   * 记录重连
   */
  recordReconnect() {
    wsReconnectsTotal.inc();
  }

  /**
   * 加入房间
   */
  joinRoom(connectionId, auctionId) {
    const connection = this.connections.get(connectionId);
    if (connection) {
      connection.rooms.add(auctionId);

      if (!this.rooms.has(auctionId)) {
        this.rooms.set(auctionId, new Set());
        activeRooms.set(this.rooms.size);
      }

      const roomUsers = this.rooms.get(auctionId);
      roomUsers.add(connectionId);

      roomUsers.set({ auction_id: auctionId }, roomUsers.size);
    }
  }

  /**
   * 离开房间
   */
  leaveRoom(connectionId, auctionId) {
    const connection = this.connections.get(connectionId);
    if (connection) {
      connection.rooms.delete(auctionId);
    }

    const room = this.rooms.get(auctionId);
    if (room) {
      room.delete(connectionId);
      roomUsers.set({ auction_id: auctionId }, room.size);

      if (room.size === 0) {
        this.rooms.delete(auctionId);
        activeRooms.set(this.rooms.size);
      }
    }
  }

  /**
   * 记录房间消息
   */
  recordRoomMessage(auctionId, type) {
    roomMessagesTotal.inc({ auction_id: auctionId, type });
  }

  /**
   * 记录实时出价
   */
  recordLiveBid(auctionId, amount, latency) {
    liveBids.inc({ auction_id: auctionId });
    liveBidAmount.observe({ auction_id: auctionId }, amount);
    bidLatency.observe({ auction_id: auctionId }, latency);
  }

  /**
   * 更新连接池状态
   */
  updatePoolStats(total, available) {
    connectionPoolSize.set(total);
    connectionPoolAvailable.set(available);
  }

  /**
   * 获取统计信息
   */
  getStats() {
    return {
      totalConnections: this.connections.size,
      totalRooms: this.rooms.size,
      connectionsByUser: this.getConnectionsByUser(),
      usersByRoom: this.getUsersByRoom()
    };
  }

  /**
   * 获取用户连接数
   */
  getConnectionsByUser() {
    const stats = {};
    this.connections.forEach(conn => {
      stats[conn.userId] = (stats[conn.userId] || 0) + 1;
    });
    return stats;
  }

  /**
   * 获取房间用户数
   */
  getUsersByRoom() {
    const stats = {};
    this.rooms.forEach((users, auctionId) => {
      stats[auctionId] = users.size;
    });
    return stats;
  }
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

/**
 * 健康检查
 */
function healthCheck() {
  return {
    status: 'healthy',
    timestamp: Date.now(),
    connections: wsConnections.hashMap[''].value || 0,
    rooms: activeRooms.hashMap[''].value || 0
  };
}

module.exports = {
  WebSocketMonitor,
  getMetrics,
  getContentType,
  healthCheck,
  register
};
