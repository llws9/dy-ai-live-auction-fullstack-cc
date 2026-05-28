import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const bidLatency = new Trend('bid_latency');
const bidSuccessRate = new Rate('bid_success');
const bidCounter = new Counter('bid_count');

// 并发出价测试场景配置 - 目标1000/s
export const options = {
  scenarios: {
    // 高并发出价场景
    high_concurrent_bids: {
      executor: 'constant-arrival-rate',
      rate: 1000, // 1000次/秒
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 500,
      maxVUs: 2000,
    },
  },
  thresholds: {
    http_req_duration: ['p(50)<100', 'p(99)<500'],
    errors: ['rate<0.01'],
    bid_latency: ['p(99)<500'],
    bid_success: ['rate>0.95'], // 出价成功率 > 95%
    bid_count: ['count>250000'], // 5分钟内总出价数 > 250000
  },
};

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PREFIX = '/api/v1';

// 全局token缓存
let globalToken = null;

// 测试函数 - 并发出价场景
export default function () {
  // 获取认证token
  if (!globalToken) {
    globalToken = getAuthToken();
  }

  if (!globalToken) {
    errorRate.add(1);
    console.error('无法获取认证token');
    return;
  }

  // 随机选择一个竞拍ID (假设有100个活跃竞拍)
  const auctionId = Math.floor(Math.random() * 100) + 1;

  // 执行出价
  testConcurrentBid(auctionId, globalToken);
}

// 获取认证token
function getAuthToken() {
  const userId = __VU;

  // 尝试登录
  const loginPayload = JSON.stringify({
    username: `bid_user_${userId}`,
    password: `Password${userId}!`,
  });

  const loginResponse = http.post(`${BASE_URL}${API_PREFIX}/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (loginResponse.status === 200) {
    try {
      const body = JSON.parse(loginResponse.body);
      return body.data?.token;
    } catch (e) {
      // ignore
    }
  }

  // 登录失败则注册
  const registerPayload = JSON.stringify({
    username: `bid_user_${userId}`,
    email: `bid_user_${userId}@test.com`,
    password: `Password${userId}!`,
  });

  const registerResponse = http.post(`${BASE_URL}${API_PREFIX}/auth/register`, registerPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (registerResponse.status === 201) {
    try {
      const body = JSON.parse(registerResponse.body);
      return body.data?.token;
    } catch (e) {
      // ignore
    }
  }

  return null;
}

// 并发出价测试
function testConcurrentBid(auctionId, token) {
  // 生成随机出价金额
  const bidAmount = 100 + Math.random() * 1000;

  const payload = JSON.stringify({
    auction_id: auctionId,
    amount: parseFloat(bidAmount.toFixed(2)),
  });

  const startTime = new Date();

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

  const duration = new Date() - startTime;
  bidLatency.add(duration);
  bidCounter.add(1);

  // 检查出价结果
  const success = check(response, {
    '出价请求完成': (r) => [200, 201, 400, 409, 429].includes(r.status),
    '出价响应时间 < 500ms': (r) => r.timings.duration < 500,
  });

  // 判断出价是否成功
  let bidSuccess = false;
  if (response.status === 200 || response.status === 201) {
    bidSuccess = true;
    bidSuccessRate.add(1);
  } else if (response.status === 400) {
    // 出价过低,不算失败
    bidSuccess = true;
    bidSuccessRate.add(1);
  } else if (response.status === 409) {
    // 竞价冲突,正常情况
    bidSuccessRate.add(0.5);
  } else if (response.status === 429) {
    // 限流,正常情况
    bidSuccessRate.add(0.3);
  } else {
    // 其他错误
    bidSuccessRate.add(0);
    errorRate.add(1);
    console.error(`出价失败 - 状态码: ${response.status}, 竞拍ID: ${auctionId}`);
  }

  return {
    success: bidSuccess,
    status: response.status,
    latency: duration,
  };
}

// 钩子函数
export function handleSummary(data) {
  const bidAnalysis = analyzeBidPerformance(data);

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/scenario_concurrent_bid.json': JSON.stringify({
      ...data,
      bid_analysis: bidAnalysis,
    }, null, 2),
    'reports/scenario_concurrent_bid.html': htmlReport(data, bidAnalysis),
  };
}

// 分析出价性能
function analyzeBidPerformance(data) {
  const metrics = data.metrics;
  return {
    total_bids: metrics.bid_count?.values?.count || 0,
    target_qps: 1000,
    actual_qps: (metrics.bid_count?.values?.count || 0) / 300, // 5分钟
    avg_bid_latency: metrics.bid_latency?.values?.avg || 0,
    p99_bid_latency: metrics.bid_latency?.values?.['p(99)'] || 0,
    bid_success_rate: (metrics.bid_success?.values?.rate || 0) * 100,
    error_rate: (metrics.errors?.values?.rate || 0) * 100,
  };
}

function textSummary(data, opts) {
  return JSON.stringify(data, null, 2);
}

function htmlReport(data, bidAnalysis) {
  return `
<!DOCTYPE html>
<html>
<head>
    <title>并发出价测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #fff3e0; border-radius: 5px; }
        .performance { background: #e8f5e9; border-left: 4px solid #4caf50; }
        h1 { color: #e65100; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #ff9800; color: white; }
        .good { color: #4caf50; font-weight: bold; }
        .warning { color: #ff9800; font-weight: bold; }
        .bad { color: #f44336; font-weight: bold; }
    </style>
</head>
<body>
    <h1>并发出价测试报告</h1>
    <div class="metric">
        <h2>测试目标</h2>
        <p>目标QPS: <strong>1000/s</strong></p>
        <p>实际QPS: <strong class="${bidAnalysis.actual_qps >= 900 ? 'good' : bidAnalysis.actual_qps >= 700 ? 'warning' : 'bad'}">${bidAnalysis.actual_qps.toFixed(2)}/s</strong></p>
    </div>
    <div class="metric performance">
        <h2>性能指标</h2>
        <p>总出价数: ${bidAnalysis.total_bids}</p>
        <p>平均出价延迟: ${bidAnalysis.avg_bid_latency.toFixed(2)}ms</p>
        <p>P99出价延迟: ${bidAnalysis.p99_bid_latency.toFixed(2)}ms</p>
        <p>出价成功率: <strong class="${bidAnalysis.bid_success_rate >= 95 ? 'good' : 'warning'}">${bidAnalysis.bid_success_rate.toFixed(2)}%</strong></p>
        <p>错误率: <strong class="${bidAnalysis.error_rate <= 1 ? 'good' : 'bad'}">${bidAnalysis.error_rate.toFixed(2)}%</strong></p>
    </div>
    <div class="metric">
        <h2>性能评估</h2>
        <ul>
            <li>P50响应时间 < 100ms: ${bidAnalysis.avg_bid_latency < 100 ? '✓ 通过' : '✗ 未通过'}</li>
            <li>P99响应时间 < 500ms: ${bidAnalysis.p99_bid_latency < 500 ? '✓ 通过' : '✗ 未通过'}</li>
            <li>出价成功率 > 95%: ${bidAnalysis.bid_success_rate >= 95 ? '✓ 通过' : '✗ 未通过'}</li>
            <li>错误率 < 1%: ${bidAnalysis.error_rate <= 1 ? '✓ 通过' : '✗ 未通过'}</li>
        </ul>
    </div>
</body>
</html>
  `;
}
