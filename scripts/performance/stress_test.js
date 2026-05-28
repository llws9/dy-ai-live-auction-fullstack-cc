import { check, group, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { Rate, Trend, Counter } from 'k6/metrics';
import { WebSocket } from 'k6/experimental/websockets';

// 自定义指标
const errorRate = new Rate('errors');
const bidLatency = new Trend('bid_latency');
const auctionListLatency = new Trend('auction_list_latency');
const auctionDetailLatency = new Trend('auction_detail_latency');
const bidCounter = new Counter('bid_count');

// 压力测试配置
export const options = {
  // 压力测试配置 - 逐步增加到系统极限
  stages: [
    { duration: '5m', target: 100 },    // 5分钟内增加到100用户
    { duration: '5m', target: 500 },    // 5分钟内增加到500用户
    { duration: '5m', target: 1000 },   // 5分钟内增加到1000用户
    { duration: '5m', target: 2000 },   // 5分钟内增加到2000用户
    { duration: '10m', target: 3000 },  // 10分钟内增加到3000用户(系统极限)
    { duration: '5m', target: 3000 },   // 保持3000用户5分钟
    { duration: '5m', target: 0 },      // 5分钟内降至0用户
  ],
  thresholds: {
    http_req_duration: ['p(50)<100', 'p(99)<500'],  // P50 < 100ms, P99 < 500ms
    errors: ['rate<0.01'],                          // 错误率 < 1%
    bid_latency: ['p(99)<500'],
    bid_count: ['count>10000'],                     // 总出价数 > 10000
  },
};

// 测试数据
const testData = new SharedArray('auction data', function () {
  const auctions = [];
  for (let i = 1; i <= 100; i++) {
    auctions.push({
      id: i,
      title: `竞拍商品 ${i}`,
      currentPrice: 100 + Math.random() * 1000,
    });
  }
  return auctions;
});

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PREFIX = '/api/v1';

// 测试函数
export default function () {
  const auctionId = (__VU % 100) + 1;
  const userId = __VU;

  // 登录获取token
  const token = loginUser(userId);

  if (!token) {
    errorRate.add(1);
    return;
  }

  // 场景1: 查询竞拍列表 (高并发)
  queryAuctionList(token);

  // 场景2: 查询竞拍详情 (高并发)
  queryAuctionDetail(auctionId, token);

  // 场景3: 并发出价 (极高并发)
  placeBid(auctionId, token, userId);

  // 场景4: WebSocket连接测试
  testWebSocketConnection(auctionId, token);

  sleep(Math.random() * 2 + 0.5); // 0.5-2.5秒随机等待
}

// 用户登录
function loginUser(userId) {
  const payload = JSON.stringify({
    username: `user_${userId}`,
    password: `Password${userId}!`,
  });

  const response = http.post(`${BASE_URL}${API_PREFIX}/auth/login`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (response.status === 200) {
    const body = JSON.parse(response.body);
    return body.data?.token;
  }

  // 如果登录失败,尝试注册
  const regPayload = JSON.stringify({
    username: `user_${userId}`,
    email: `user_${userId}@test.com`,
    password: `Password${userId}!`,
  });

  const regResponse = http.post(`${BASE_URL}${API_PREFIX}/auth/register`, regPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (regResponse.status === 201) {
    const body = JSON.parse(regResponse.body);
    return body.data?.token;
  }

  return null;
}

// 查询竞拍列表
function queryAuctionList(token) {
  group('竞拍列表查询', () => {
    const startTime = new Date();

    const response = http.get(`${BASE_URL}${API_PREFIX}/auctions?page=1&limit=20`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    const duration = new Date() - startTime;
    auctionListLatency.add(duration);

    const success = check(response, {
      '竞拍列表状态码为200': (r) => r.status === 200,
      '竞拍列表响应时间 < 100ms': (r) => r.timings.duration < 100,
      '竞拍列表有数据': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.items && body.data.items.length > 0;
        } catch (e) {
          return false;
        }
      },
    });

    errorRate.add(!success);
  });
}

// 查询竞拍详情
function queryAuctionDetail(auctionId, token) {
  group('竞拍详情查询', () => {
    const startTime = new Date();

    const response = http.get(`${BASE_URL}${API_PREFIX}/auctions/${auctionId}`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    const duration = new Date() - startTime;
    auctionDetailLatency.add(duration);

    const success = check(response, {
      '竞拍详情状态码为200': (r) => r.status === 200,
      '竞拍详情响应时间 < 200ms': (r) => r.timings.duration < 200,
      '竞拍详情完整': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.id && body.data.title;
        } catch (e) {
          return false;
        }
      },
    });

    errorRate.add(!success);
  });
}

// 并发出价
function placeBid(auctionId, token, userId) {
  group('竞拍出价', () => {
    const bidAmount = 100 + Math.random() * 1000;

    const payload = JSON.stringify({
      auction_id: auctionId,
      amount: bidAmount,
    });

    const startTime = new Date();

    const response = http.post(`${BASE_URL}${API_PREFIX}/auctions/${auctionId}/bid`, payload, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    });

    const duration = new Date() - startTime;
    bidLatency.add(duration);
    bidCounter.add(1);

    const success = check(response, {
      '出价状态码为200或201': (r) => r.status === 200 || r.status === 201,
      '出价响应时间 < 500ms': (r) => r.timings.duration < 500,
      '出价成功': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success === true || body.code === 0;
        } catch (e) {
          return false;
        }
      },
    });

    errorRate.add(!success);
  });
}

// WebSocket连接测试
function testWebSocketConnection(auctionId, token) {
  group('WebSocket连接', () => {
    try {
      const wsUrl = BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://');
      const ws = new WebSocket(`${wsUrl}/ws/auction/${auctionId}?token=${token}`);

      ws.on('open', () => {
        console.log(`WebSocket连接已建立 - 竞拍ID: ${auctionId}`);

        // 发送测试消息
        ws.send(JSON.stringify({
          type: 'ping',
          timestamp: Date.now(),
        }));
      });

      ws.on('message', (message) => {
        console.log(`收到WebSocket消息: ${message}`);
      });

      ws.on('error', (error) => {
        console.error(`WebSocket错误: ${error}`);
        errorRate.add(1);
      });

      // 保持连接5秒
      setTimeout(() => {
        ws.close();
      }, 5000);
    } catch (error) {
      console.error(`WebSocket连接失败: ${error}`);
      errorRate.add(1);
    }
  });
}

// 钩子函数
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/stress_test_summary.json': JSON.stringify(data, null, 2),
    'reports/stress_test_report.html': htmlReport(data),
  };
}

function textSummary(data, opts) {
  return JSON.stringify(data, null, 2);
}

function htmlReport(data) {
  return `
<!DOCTYPE html>
<html>
<head>
    <title>压力测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #f5f5f5; border-radius: 5px; }
        .pass { color: green; }
        .fail { color: red; }
        h1 { color: #333; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f44336; color: white; }
    </style>
</head>
<body>
    <h1>压力测试报告</h1>
    <div class="metric">
        <h2>测试概览</h2>
        <p>总请求数: ${data.metrics.http_reqs.values.count}</p>
        <p>平均响应时间: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms</p>
        <p>P50响应时间: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms</p>
        <p>P99响应时间: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms</p>
        <p>错误率: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%</p>
        <p>总出价数: ${data.metrics.bid_count?.values?.count || 0}</p>
    </div>
</body>
</html>
  `;
}
