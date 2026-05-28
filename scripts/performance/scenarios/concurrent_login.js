import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const authLatency = new Trend('auth_latency');
const authCounter = new Counter('auth_requests');

// 并发登录测试场景配置
export const options = {
  stages: [
    { duration: '1m', target: 50 },    // 1分钟内增加到50用户
    { duration: '2m', target: 100 },   // 2分钟内增加到100用户
    { duration: '3m', target: 100 },   // 保持100用户3分钟(100/s并发登录)
    { duration: '1m', target: 200 },   // 1分钟内增加到200用户
    { duration: '2m', target: 200 },   // 保持200用户2分钟
    { duration: '1m', target: 0 },     // 1分钟内降至0用户
  ],
  thresholds: {
    http_req_duration: ['p(50)<100', 'p(99)<200'],
    errors: ['rate<0.01'],
    auth_latency: ['p(99)<200'],
  },
};

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PREFIX = '/api/v1';

// 测试函数 - 并发登录场景
export default function () {
  const userId = __VU;

  // 场景1: 并发登录 (100/s)
  testConcurrentLogin(userId);

  sleep(Math.random() * 0.5 + 0.5); // 0.5-1秒随机等待
}

// 并发登录测试
function testConcurrentLogin(userId) {
  const username = `concurrent_user_${userId}`;
  const password = `Password${userId}!`;

  // 首先尝试登录
  const loginResponse = attemptLogin(username, password);

  if (!loginResponse.success) {
    // 如果登录失败,尝试注册
    attemptRegister(userId);
  }

  authCounter.add(1);
}

// 尝试登录
function attemptLogin(username, password) {
  const payload = JSON.stringify({
    username: username,
    password: password,
  });

  const startTime = new Date();

  const response = http.post(`${BASE_URL}${API_PREFIX}/auth/login`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const duration = new Date() - startTime;
  authLatency.add(duration);

  const success = check(response, {
    '登录成功': (r) => r.status === 200,
    '登录响应时间 < 200ms': (r) => r.timings.duration < 200,
    '登录返回有效token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && body.data.token;
      } catch (e) {
        return false;
      }
    },
  });

  errorRate.add(!success);

  return {
    success: success,
    status: response.status,
    response: response,
  };
}

// 尝试注册
function attemptRegister(userId) {
  const username = `concurrent_user_${userId}`;
  const email = `concurrent_user_${userId}@test.com`;
  const password = `Password${userId}!`;

  const payload = JSON.stringify({
    username: username,
    email: email,
    password: password,
  });

  const response = http.post(`${BASE_URL}${API_PREFIX}/auth/register`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const success = check(response, {
    '注册成功': (r) => r.status === 201,
    '注册响应时间 < 300ms': (r) => r.timings.duration < 300,
  });

  errorRate.add(!success);

  return {
    success: success,
    status: response.status,
    response: response,
  };
}

// 钩子函数
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/scenario_concurrent_login.json': JSON.stringify(data, null, 2),
    'reports/scenario_concurrent_login.html': htmlReport(data),
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
    <title>并发登录测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #e3f2fd; border-radius: 5px; }
        h1 { color: #1976d2; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #2196f3; color: white; }
    </style>
</head>
<body>
    <h1>并发登录测试报告</h1>
    <div class="metric">
        <h2>测试目标</h2>
        <p>并发登录: 100/s</p>
        <p>并发注册: 50/s</p>
    </div>
    <div class="metric">
        <h2>测试结果</h2>
        <p>总认证请求数: ${data.metrics.auth_requests?.values?.count || 0}</p>
        <p>平均认证延迟: ${data.metrics.auth_latency?.values?.avg.toFixed(2) || 0}ms</p>
        <p>P99认证延迟: ${data.metrics.auth_latency?.values?.['p(99)'].toFixed(2) || 0}ms</p>
        <p>错误率: ${(data.metrics.errors?.values?.rate * 100 || 0).toFixed(2)}%</p>
    </div>
</body>
</html>
  `;
}
