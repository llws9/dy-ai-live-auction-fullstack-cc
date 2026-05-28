import { check, group, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const loginLatency = new Trend('login_latency');
const registerLatency = new Trend('register_latency');

// 测试配置
export const options = {
  // 负载测试配置
  stages: [
    { duration: '2m', target: 100 },   // 2分钟内增加到100用户
    { duration: '5m', target: 100 },   // 保持100用户5分钟
    { duration: '2m', target: 200 },   // 2分钟内增加到200用户
    { duration: '5m', target: 200 },   // 保持200用户5分钟
    { duration: '2m', target: 300 },   // 2分钟内增加到300用户
    { duration: '5m', target: 300 },   // 保持300用户5分钟
    { duration: '3m', target: 0 },     // 3分钟内降至0用户
  ],
  thresholds: {
    http_req_duration: ['p(50)<100', 'p(99)<200'],  // P50 < 100ms, P99 < 200ms
    errors: ['rate<0.01'],                           // 错误率 < 1%
    login_latency: ['p(99)<200'],
    register_latency: ['p(99)<300'],
  },
};

// 测试数据
const testData = new SharedArray('test users', function () {
  const users = [];
  for (let i = 0; i < 1000; i++) {
    users.push({
      username: `testuser_${i}`,
      email: `test_${i}@example.com`,
      password: `Password${i}!`,
    });
  }
  return users;
});

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PREFIX = '/api/v1';

// 测试函数
export default function () {
  const userId = __VU % testData.length;
  const user = testData[userId];

  // 场景1: 用户注册
  if (__VU % 3 === 0) {
    testUserRegistration(user);
  }

  // 场景2: 用户登录
  testUserLogin(user);

  sleep(1);
}

// 用户注册测试
function testUserRegistration(user) {
  group('用户注册', () => {
    const payload = JSON.stringify({
      username: user.username,
      email: user.email,
      password: user.password,
    });

    const startTime = new Date();

    const response = http.post(`${BASE_URL}${API_PREFIX}/auth/register`, payload, {
      headers: {
        'Content-Type': 'application/json',
      },
    });

    const duration = new Date() - startTime;
    registerLatency.add(duration);

    const success = check(response, {
      '注册状态码为201': (r) => r.status === 201,
      '注册响应时间 < 300ms': (r) => r.timings.duration < 300,
      '注册返回token': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.token;
        } catch (e) {
          return false;
        }
      },
    });

    errorRate.add(!success);
  });
}

// 用户登录测试
function testUserLogin(user) {
  group('用户登录', () => {
    const payload = JSON.stringify({
      username: user.username,
      password: user.password,
    });

    const startTime = new Date();

    const response = http.post(`${BASE_URL}${API_PREFIX}/auth/login`, payload, {
      headers: {
        'Content-Type': 'application/json',
      },
    });

    const duration = new Date() - startTime;
    loginLatency.add(duration);

    const success = check(response, {
      '登录状态码为200': (r) => r.status === 200,
      '登录响应时间 < 200ms': (r) => r.timings.duration < 200,
      '登录返回token': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.token;
        } catch (e) {
          return false;
        }
      },
    });

    errorRate.add(!success);

    if (success) {
      const body = JSON.parse(response.body);
      return body.data.token;
    }

    return null;
  });
}

// 钩子函数
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/load_test_summary.json': JSON.stringify(data, null, 2),
    'reports/load_test_report.html': htmlReport(data),
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
    <title>负载测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #f5f5f5; border-radius: 5px; }
        .pass { color: green; }
        .fail { color: red; }
        h1 { color: #333; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
    </style>
</head>
<body>
    <h1>负载测试报告</h1>
    <div class="metric">
        <h2>测试概览</h2>
        <p>总请求数: ${data.metrics.http_reqs.values.count}</p>
        <p>平均响应时间: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms</p>
        <p>P50响应时间: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms</p>
        <p>P99响应时间: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms</p>
        <p>错误率: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%</p>
    </div>
</body>
</html>
  `;
}
