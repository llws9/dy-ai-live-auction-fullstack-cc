import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const auctionQueryLatency = new Trend('auction_query_latency');
const auctionDetailLatency = new Trend('auction_detail_latency');
const queryCounter = new Counter('query_count');

// 竞拍查询测试场景配置
export const options = {
  scenarios: {
    // 竞拍列表查询 - 500/s
    auction_list_query: {
      executor: 'constant-arrival-rate',
      rate: 500,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 100,
      maxVUs: 500,
      exec: 'testAuctionListQuery',
    },
    // 竞拍详情查询 - 800/s
    auction_detail_query: {
      executor: 'constant-arrival-rate',
      rate: 800,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 200,
      maxVUs: 800,
      exec: 'testAuctionDetailQuery',
    },
  },
  thresholds: {
    http_req_duration: ['p(50)<100', 'p(99)<200'],
    errors: ['rate<0.01'],
    auction_query_latency: ['p(99)<200'],
    auction_detail_latency: ['p(99)<200'],
  },
};

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PREFIX = '/api/v1';

// 测试函数
export default function () {
  // 使用不同的测试函数
}

// 竞拍列表查询测试
export function testAuctionListQuery() {
  const token = getAuthToken();

  if (!token) {
    errorRate.add(1);
    return;
  }

  const page = Math.floor(Math.random() * 10) + 1;
  const status = ['active', 'pending', 'completed'][Math.floor(Math.random() * 3)];

  const startTime = new Date();

  const response = http.get(
    `${BASE_URL}${API_PREFIX}/auctions?page=${page}&limit=20&status=${status}`,
    {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    }
  );

  const duration = new Date() - startTime;
  auctionQueryLatency.add(duration);
  queryCounter.add(1);

  const success = check(response, {
    '竞拍列表查询成功': (r) => r.status === 200,
    '竞拍列表响应时间 < 200ms': (r) => r.timings.duration < 200,
    '竞拍列表有数据': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && body.data.items;
      } catch (e) {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

// 竞拍详情查询测试
export function testAuctionDetailQuery() {
  const token = getAuthToken();

  if (!token) {
    errorRate.add(1);
    return;
  }

  // 随机选择一个竞拍ID
  const auctionId = Math.floor(Math.random() * 100) + 1;

  const startTime = new Date();

  const response = http.get(
    `${BASE_URL}${API_PREFIX}/auctions/${auctionId}`,
    {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    }
  );

  const duration = new Date() - startTime;
  auctionDetailLatency.add(duration);
  queryCounter.add(1);

  const success = check(response, {
    '竞拍详情查询成功': (r) => r.status === 200,
    '竞拍详情响应时间 < 200ms': (r) => r.timings.duration < 200,
    '竞拍详情数据完整': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && body.data.id && body.data.title && body.data.current_price;
      } catch (e) {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

// 获取认证token
function getAuthToken() {
  const userId = __VU;

  // 使用缓存token
  const cacheKey = `token_${userId}`;
  const cachedToken = __ENV[cacheKey];
  if (cachedToken) {
    return cachedToken;
  }

  // 尝试登录
  const loginPayload = JSON.stringify({
    username: `query_user_${userId}`,
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
    username: `query_user_${userId}`,
    email: `query_user_${userId}@test.com`,
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

// 钩子函数
export function handleSummary(data) {
  const queryAnalysis = analyzeQueryPerformance(data);

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/scenario_auction_query.json': JSON.stringify({
      ...data,
      query_analysis: queryAnalysis,
    }, null, 2),
    'reports/scenario_auction_query.html': htmlReport(data, queryAnalysis),
  };
}

// 分析查询性能
function analyzeQueryPerformance(data) {
  const metrics = data.metrics;
  return {
    total_queries: metrics.query_count?.values?.count || 0,
    avg_query_latency: metrics.auction_query_latency?.values?.avg || 0,
    p99_query_latency: metrics.auction_query_latency?.values?.['p(99)'] || 0,
    avg_detail_latency: metrics.auction_detail_latency?.values?.avg || 0,
    p99_detail_latency: metrics.auction_detail_latency?.values?.['p(99)'] || 0,
    error_rate: (metrics.errors?.values?.rate || 0) * 100,
  };
}

function textSummary(data, opts) {
  return JSON.stringify(data, null, 2);
}

function htmlReport(data, queryAnalysis) {
  return `
<!DOCTYPE html>
<html>
<head>
    <title>竞拍查询测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #e8eaf6; border-radius: 5px; }
        h1 { color: #3f51b5; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #3f51b5; color: white; }
    </style>
</head>
<body>
    <h1>竞拍查询测试报告</h1>
    <div class="metric">
        <h2>测试目标</h2>
        <p>竞拍列表查询: 500/s</p>
        <p>竞拍详情查询: 800/s</p>
    </div>
    <div class="metric">
        <h2>测试结果</h2>
        <p>总查询数: ${queryAnalysis.total_queries}</p>
        <p>平均列表查询延迟: ${queryAnalysis.avg_query_latency.toFixed(2)}ms</p>
        <p>P99列表查询延迟: ${queryAnalysis.p99_query_latency.toFixed(2)}ms</p>
        <p>平均详情查询延迟: ${queryAnalysis.avg_detail_latency.toFixed(2)}ms</p>
        <p>P99详情查询延迟: ${queryAnalysis.p99_detail_latency.toFixed(2)}ms</p>
        <p>错误率: ${queryAnalysis.error_rate.toFixed(2)}%</p>
    </div>
</body>
</html>
  `;
}
