import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const productLatency = new Trend('product_latency');
const productCounter = new Counter('product_requests');

// 商品管理测试场景配置
export const options = {
  scenarios: {
    // 商品列表查询场景 - 500/s
    product_list_query: {
      executor: 'constant-arrival-rate',
      rate: 500,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 100,
      maxVUs: 500,
      exec: 'testProductList',
    },
    // 商品创建场景 - 100/s
    product_creation: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 50,
      maxVUs: 200,
      exec: 'testProductCreation',
    },
  },
  thresholds: {
    http_req_duration: ['p(50)<100', 'p(99)<200'],
    errors: ['rate<0.01'],
    product_latency: ['p(99)<200'],
  },
};

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PREFIX = '/api/v1';

// 全局token缓存
let adminToken = null;

// 初始化
export function setup() {
  // 获取管理员token
  const token = getAdminToken();
  return { token: token };
}

// 商品列表查询测试
export function testProductList(data) {
  const token = data.token || getAdminToken();

  if (!token) {
    errorRate.add(1);
    return;
  }

  const page = Math.floor(Math.random() * 10) + 1;
  const limit = 20;

  const startTime = new Date();

  const response = http.get(`${BASE_URL}${API_PREFIX}/products?page=${page}&limit=${limit}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  const duration = new Date() - startTime;
  productLatency.add(duration);
  productCounter.add(1);

  const success = check(response, {
    '商品列表状态码为200': (r) => r.status === 200,
    '商品列表响应时间 < 200ms': (r) => r.timings.duration < 200,
    '商品列表有数据': (r) => {
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

// 商品创建测试
export function testProductCreation(data) {
  const token = data.token || getAdminToken();

  if (!token) {
    errorRate.add(1);
    return;
  }

  const productId = Date.now();
  const productData = {
    name: `测试商品_${productId}`,
    description: `这是测试商品描述_${productId}`,
    price: 100 + Math.random() * 1000,
    stock: Math.floor(Math.random() * 1000),
    category_id: Math.floor(Math.random() * 10) + 1,
    images: ['https://example.com/image.jpg'],
  };

  const payload = JSON.stringify(productData);

  const startTime = new Date();

  const response = http.post(`${BASE_URL}${API_PREFIX}/products`, payload, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  });

  const duration = new Date() - startTime;
  productLatency.add(duration);
  productCounter.add(1);

  const success = check(response, {
    '商品创建状态码为201': (r) => r.status === 201 || r.status === 200,
    '商品创建响应时间 < 300ms': (r) => r.timings.duration < 300,
    '商品创建返回ID': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && body.data.id;
      } catch (e) {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

// 获取管理员token
function getAdminToken() {
  if (adminToken) {
    return adminToken;
  }

  // 尝试登录管理员账号
  const loginPayload = JSON.stringify({
    username: 'admin',
    password: 'AdminPassword123!',
  });

  const loginResponse = http.post(`${BASE_URL}${API_PREFIX}/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (loginResponse.status === 200) {
    try {
      const body = JSON.parse(loginResponse.body);
      adminToken = body.data?.token;
      return adminToken;
    } catch (e) {
      // ignore
    }
  }

  // 如果登录失败,使用普通用户
  const userId = __VU;
  const userLoginPayload = JSON.stringify({
    username: `product_user_${userId}`,
    password: `Password${userId}!`,
  });

  const userLoginResponse = http.post(`${BASE_URL}${API_PREFIX}/auth/login`, userLoginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (userLoginResponse.status === 200) {
    try {
      const body = JSON.parse(userLoginResponse.body);
      return body.data?.token;
    } catch (e) {
      // ignore
    }
  }

  // 注册新用户
  const registerPayload = JSON.stringify({
    username: `product_user_${userId}`,
    email: `product_user_${userId}@test.com`,
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
  const productAnalysis = analyzeProductPerformance(data);

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/scenario_product_management.json': JSON.stringify({
      ...data,
      product_analysis: productAnalysis,
    }, null, 2),
    'reports/scenario_product_management.html': htmlReport(data, productAnalysis),
  };
}

// 分析商品管理性能
function analyzeProductPerformance(data) {
  const metrics = data.metrics;
  return {
    total_requests: metrics.product_requests?.values?.count || 0,
    avg_latency: metrics.product_latency?.values?.avg || 0,
    p99_latency: metrics.product_latency?.values?.['p(99)'] || 0,
    error_rate: (metrics.errors?.values?.rate || 0) * 100,
  };
}

function textSummary(data, opts) {
  return JSON.stringify(data, null, 2);
}

function htmlReport(data, productAnalysis) {
  return `
<!DOCTYPE html>
<html>
<head>
    <title>商品管理测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #e0f2f1; border-radius: 5px; }
        h1 { color: #00796b; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #009688; color: white; }
    </style>
</head>
<body>
    <h1>商品管理测试报告</h1>
    <div class="metric">
        <h2>测试目标</h2>
        <p>商品列表查询: 500/s</p>
        <p>商品创建: 100/s</p>
    </div>
    <div class="metric">
        <h2>测试结果</h2>
        <p>总请求数: ${productAnalysis.total_requests}</p>
        <p>平均延迟: ${productAnalysis.avg_latency.toFixed(2)}ms</p>
        <p>P99延迟: ${productAnalysis.p99_latency.toFixed(2)}ms</p>
        <p>错误率: ${productAnalysis.error_rate.toFixed(2)}%</p>
    </div>
</body>
</html>
  `;
}
