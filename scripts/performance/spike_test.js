import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const responseTimeTrend = new Trend('response_time');

// 峰值测试配置
export const options = {
  // 峰值测试 - 模拟突发流量
  stages: [
    { duration: '1m', target: 50 },     // 1分钟内增加到50用户(基线)
    { duration: '3m', target: 50 },     // 保持50用户3分钟
    { duration: '30s', target: 1000 },  // 30秒内突增至1000用户(峰值)
    { duration: '2m', target: 1000 },   // 保持1000用户2分钟
    { duration: '30s', target: 50 },    // 30秒内降至50用户
    { duration: '2m', target: 50 },     // 保持50用户2分钟
    { duration: '30s', target: 2000 },  // 第二次峰值 - 30秒内突增至2000用户
    { duration: '2m', target: 2000 },   // 保持2000用户2分钟
    { duration: '30s', target: 50 },    // 30秒内降至50用户
    { duration: '2m', target: 50 },     // 保持50用户2分钟
    { duration: '30s', target: 3000 },  // 第三次峰值 - 30秒内突增至3000用户
    { duration: '3m', target: 3000 },   // 保持3000用户3分钟
    { duration: '1m', target: 0 },      // 1分钟内降至0用户
  ],
  thresholds: {
    http_req_duration: ['p(50)<150', 'p(99)<1000'],  // P50 < 150ms, P99 < 1000ms
    errors: ['rate<0.05'],                            // 峰值期间错误率 < 5%
  },
};

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PREFIX = '/api/v1';

// 测试函数
export default function () {
  const scenario = Math.floor(Math.random() * 4); // 0-3 随机场景

  // 登录获取token
  const token = getAuthToken();

  switch (scenario) {
    case 0:
      testAuctionList(token);
      break;
    case 1:
      testAuctionDetail(token);
      break;
    case 2:
      testPlaceBid(token);
      break;
    case 3:
      testProductList(token);
      break;
  }

  sleep(Math.random() * 1 + 0.2); // 0.2-1.2秒随机等待
}

// 获取认证token
function getAuthToken() {
  const userId = __VU;

  // 尝试登录
  const loginPayload = JSON.stringify({
    username: `spike_user_${userId}`,
    password: `Password${userId}!`,
  });

  const loginResponse = http.post(`${BASE_URL}${API_PREFIX}/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (loginResponse.status === 200) {
    const body = JSON.parse(loginResponse.body);
    return body.data?.token;
  }

  // 登录失败则注册
  const registerPayload = JSON.stringify({
    username: `spike_user_${userId}`,
    email: `spike_user_${userId}@test.com`,
    password: `Password${userId}!`,
  });

  const registerResponse = http.post(`${BASE_URL}${API_PREFIX}/auth/register`, registerPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (registerResponse.status === 201) {
    const body = JSON.parse(registerResponse.body);
    return body.data?.token;
  }

  return null;
}

// 测试竞拍列表
function testAuctionList(token) {
  group('竞拍列表(峰值)', () => {
    const response = http.get(`${BASE_URL}${API_PREFIX}/auctions?page=1&limit=20`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    responseTimeTrend.add(response.timings.duration);

    const success = check(response, {
      '竞拍列表状态码正确': (r) => r.status === 200 || r.status === 429, // 429表示限流
      '竞拍列表响应时间可接受': (r) => r.timings.duration < 1000,
    });

    errorRate.add(!success);
  });
}

// 测试竞拍详情
function testAuctionDetail(token) {
  group('竞拍详情(峰值)', () => {
    const auctionId = (__VU % 100) + 1;

    const response = http.get(`${BASE_URL}${API_PREFIX}/auctions/${auctionId}`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    responseTimeTrend.add(response.timings.duration);

    const success = check(response, {
      '竞拍详情状态码正确': (r) => r.status === 200 || r.status === 429,
      '竞拍详情响应时间可接受': (r) => r.timings.duration < 1000,
    });

    errorRate.add(!success);
  });
}

// 测试出价
function testPlaceBid(token) {
  group('并发出价(峰值)', () => {
    const auctionId = (__VU % 100) + 1;
    const bidAmount = 100 + Math.random() * 1000;

    const payload = JSON.stringify({
      auction_id: auctionId,
      amount: bidAmount,
    });

    const response = http.post(
      `${BASE_URL}${API_PREFIX}/auctions/${auctionId}/bid`,
      payload,
      {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      }
    );

    responseTimeTrend.add(response.timings.duration);

    const success = check(response, {
      '出价状态码正确': (r) => [200, 201, 400, 429].includes(r.status), // 400可能表示出价过低
      '出价响应时间可接受': (r) => r.timings.duration < 1000,
    });

    errorRate.add(!success);
  });
}

// 测试商品列表
function testProductList(token) {
  group('商品列表(峰值)', () => {
    const response = http.get(`${BASE_URL}${API_PREFIX}/products?page=1&limit=20`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    responseTimeTrend.add(response.timings.duration);

    const success = check(response, {
      '商品列表状态码正确': (r) => r.status === 200 || r.status === 429,
      '商品列表响应时间可接受': (r) => r.timings.duration < 1000,
    });

    errorRate.add(!success);
  });
}

// 钩子函数
export function handleSummary(data) {
  const peakMetrics = analyzePeakPerformance(data);

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/spike_test_summary.json': JSON.stringify({
      ...data,
      peak_analysis: peakMetrics,
    }, null, 2),
    'reports/spike_test_report.html': htmlReport(data, peakMetrics),
  };
}

// 分析峰值性能
function analyzePeakPerformance(data) {
  const metrics = data.metrics;
  return {
    total_requests: metrics.http_reqs?.values?.count || 0,
    avg_response_time: metrics.http_req_duration?.values?.avg || 0,
    peak_response_time: metrics.http_req_duration?.values?.max || 0,
    p99_response_time: metrics.http_req_duration?.values?.['p(99)'] || 0,
    error_rate: metrics.errors?.values?.rate || 0,
    recovery_time: 'N/A', // 需要从详细数据中计算
  };
}

function textSummary(data, opts) {
  return JSON.stringify(data, null, 2);
}

function htmlReport(data, peakMetrics) {
  return `
<!DOCTYPE html>
<html>
<head>
    <title>峰值测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #f5f5f5; border-radius: 5px; }
        .peak { background: #fff3cd; border-left: 4px solid #ffc107; }
        .pass { color: green; }
        .fail { color: red; }
        h1 { color: #333; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #ff9800; color: white; }
    </style>
</head>
<body>
    <h1>峰值测试报告</h1>
    <div class="metric peak">
        <h2>峰值性能分析</h2>
        <p>总请求数: ${peakMetrics.total_requests}</p>
        <p>平均响应时间: ${peakMetrics.avg_response_time.toFixed(2)}ms</p>
        <p>峰值响应时间: ${peakMetrics.peak_response_time.toFixed(2)}ms</p>
        <p>P99响应时间: ${peakMetrics.p99_response_time.toFixed(2)}ms</p>
        <p>错误率: ${(peakMetrics.error_rate * 100).toFixed(2)}%</p>
    </div>
    <div class="metric">
        <h2>测试说明</h2>
        <p>峰值测试模拟了三次突发流量高峰,测试系统的弹性扩展和恢复能力</p>
        <ul>
            <li>第一次峰值: 50用户突增至1000用户</li>
            <li>第二次峰值: 50用户突增至2000用户</li>
            <li>第三次峰值: 50用户突增至3000用户</li>
        </ul>
    </div>
</body>
</html>
  `;
}
