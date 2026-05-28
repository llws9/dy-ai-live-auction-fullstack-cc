import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const registrationLatency = new Trend('registration_latency');
const registrationCounter = new Counter('registration_count');

// 并发注册测试场景配置
export const options = {
  stages: [
    { duration: '1m', target: 25 },    // 1分钟内增加到25用户
    { duration: '2m', target: 50 },    // 2分钟内增加到50用户
    { duration: '3m', target: 50 },    // 保持50用户3分钟(50/s并发注册)
    { duration: '1m', target: 100 },   // 1分钟内增加到100用户
    { duration: '2m', target: 100 },   // 保持100用户2分钟
    { duration: '1m', target: 0 },     // 1分钟内降至0用户
  ],
  thresholds: {
    http_req_duration: ['p(50)<150', 'p(99)<300'],
    errors: ['rate<0.05'],  // 注册错误率可以稍高
    registration_latency: ['p(99)<300'],
  },
};

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PREFIX = '/api/v1';

// 测试函数 - 并发注册场景
export default function () {
  const timestamp = Date.now();
  const vuId = __VU;

  // 生成唯一的用户名和邮箱
  const username = `user_${timestamp}_${vuId}`;
  const email = `user_${timestamp}_${vuId}@test.com`;
  const password = `Password${vuId}!`;

  // 执行注册
  testUserRegistration(username, email, password);

  sleep(Math.random() * 0.5 + 0.5); // 0.5-1秒随机等待
}

// 用户注册测试
function testUserRegistration(username, email, password) {
  const payload = JSON.stringify({
    username: username,
    email: email,
    password: password,
  });

  const startTime = new Date();

  const response = http.post(`${BASE_URL}${API_PREFIX}/auth/register`, payload, {
    headers: {
      'Content-Type': 'application/json',
    },
  });

  const duration = new Date() - startTime;
  registrationLatency.add(duration);
  registrationCounter.add(1);

  const success = check(response, {
    '注册状态码为201': (r) => r.status === 201,
    '注册响应时间 < 300ms': (r) => r.timings.duration < 300,
    '注册返回有效数据': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && (body.data.token || body.data.user_id);
      } catch (e) {
        return false;
      }
    },
    '用户名唯一': (r) => {
      // 如果用户名已存在,返回409状态码
      return r.status !== 409;
    },
  });

  // 注册失败不一定是错误(可能是用户名重复)
  if (!success && response.status !== 409) {
    errorRate.add(1);
  }

  return {
    success: success,
    status: response.status,
    response: response,
  };
}

// 钩子函数
export function handleSummary(data) {
  const registrationAnalysis = analyzeRegistrationPerformance(data);

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/scenario_concurrent_registration.json': JSON.stringify({
      ...data,
      registration_analysis: registrationAnalysis,
    }, null, 2),
    'reports/scenario_concurrent_registration.html': htmlReport(data, registrationAnalysis),
  };
}

// 分析注册性能
function analyzeRegistrationPerformance(data) {
  const metrics = data.metrics;
  return {
    total_registrations: metrics.registration_count?.values?.count || 0,
    avg_latency: metrics.registration_latency?.values?.avg || 0,
    p99_latency: metrics.registration_latency?.values?.['p(99)'] || 0,
    error_rate: (metrics.errors?.values?.rate || 0) * 100,
  };
}

function textSummary(data, opts) {
  return JSON.stringify(data, null, 2);
}

function htmlReport(data, registrationAnalysis) {
  return `
<!DOCTYPE html>
<html>
<head>
    <title>并发注册测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #fce4ec; border-radius: 5px; }
        h1 { color: #c2185b; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #e91e63; color: white; }
    </style>
</head>
<body>
    <h1>并发注册测试报告</h1>
    <div class="metric">
        <h2>测试目标</h2>
        <p>并发注册: 50/s</p>
    </div>
    <div class="metric">
        <h2>测试结果</h2>
        <p>总注册数: ${registrationAnalysis.total_registrations}</p>
        <p>平均注册延迟: ${registrationAnalysis.avg_latency.toFixed(2)}ms</p>
        <p>P99注册延迟: ${registrationAnalysis.p99_latency.toFixed(2)}ms</p>
        <p>错误率: ${registrationAnalysis.error_rate.toFixed(2)}%</p>
    </div>
</body>
</html>
  `;
}
