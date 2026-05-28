import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const wsConnectLatency = new Trend('ws_connect_latency');
const wsMessageLatency = new Trend('ws_message_latency');
const wsConnections = new Counter('ws_connections');
const wsMessages = new Counter('ws_messages');

// WebSocket连接测试场景配置
export const options = {
  scenarios: {
    // 并发WebSocket连接场景
    concurrent_ws_connections: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 500 },   // 2分钟内增加到500个连接
        { duration: '3m', target: 500 },   // 保持500个连接3分钟
        { duration: '2m', target: 1000 },  // 2分钟内增加到1000个连接
        { duration: '5m', target: 1000 },  // 保持1000个连接5分钟
        { duration: '2m', target: 0 },     // 2分钟内降至0
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    ws_connect_latency: ['p(99)<500'],  // WebSocket连接延迟 P99 < 500ms
    ws_message_latency: ['p(99)<200'],  // 消息延迟 P99 < 200ms
    errors: ['rate<0.05'],              // 错误率 < 5%
  },
};

// 基础URL配置
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const WS_URL = BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://');
const API_PREFIX = '/api/v1';

// 测试函数 - WebSocket连接场景
export default function () {
  // 获取认证token
  const token = getAuthToken();

  if (!token) {
    errorRate.add(1);
    console.error('无法获取认证token');
    return;
  }

  // 随机选择一个竞拍房间
  const auctionId = Math.floor(Math.random() * 100) + 1;

  // 测试WebSocket连接
  testWebSocketConnection(auctionId, token);
}

// 获取认证token
function getAuthToken() {
  const userId = __VU;

  // 尝试登录
  const loginPayload = JSON.stringify({
    username: `ws_user_${userId}`,
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
    username: `ws_user_${userId}`,
    email: `ws_user_${userId}@test.com`,
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

// WebSocket连接测试
function testWebSocketConnection(auctionId, token) {
  const wsUrl = `${WS_URL}/ws/auction/${auctionId}?token=${token}`;

  const startTime = new Date();

  try {
    const ws = new WebSocket(wsUrl);

    ws.on('open', () => {
      const connectDuration = new Date() - startTime;
      wsConnectLatency.add(connectDuration);
      wsConnections.add(1);

      console.log(`WebSocket连接已建立 - 竞拍ID: ${auctionId}, 延迟: ${connectDuration}ms`);

      // 发送心跳消息
      ws.send(JSON.stringify({
        type: 'ping',
        timestamp: Date.now(),
      }));

      // 定期发送测试消息
      const messageInterval = setInterval(() => {
        const msgStartTime = new Date();
        ws.send(JSON.stringify({
          type: 'ping',
          timestamp: msgStartTime.getTime(),
        }));

        // 等待响应
        ws.on('message', (message) => {
          const msgLatency = new Date() - msgStartTime;
          wsMessageLatency.add(msgLatency);
          wsMessages.add(1);
        });
      }, 1000); // 每秒发送一条消息

      // 保持连接10秒
      setTimeout(() => {
        clearInterval(messageInterval);
        ws.close();
      }, 10000);
    });

    ws.on('message', (message) => {
      console.log(`收到消息: ${message}`);
    });

    ws.on('error', (error) => {
      console.error(`WebSocket错误: ${error}`);
      errorRate.add(1);
    });

    ws.on('close', () => {
      console.log(`WebSocket连接已关闭 - 竞拍ID: ${auctionId}`);
    });
  } catch (error) {
    console.error(`WebSocket连接失败: ${error}`);
    errorRate.add(1);
  }
}

// 钩子函数
export function handleSummary(data) {
  const wsAnalysis = analyzeWebSocketPerformance(data);

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'reports/scenario_websocket.json': JSON.stringify({
      ...data,
      ws_analysis: wsAnalysis,
    }, null, 2),
    'reports/scenario_websocket.html': htmlReport(data, wsAnalysis),
  };
}

// 分析WebSocket性能
function analyzeWebSocketPerformance(data) {
  const metrics = data.metrics;
  return {
    total_connections: metrics.ws_connections?.values?.count || 0,
    total_messages: metrics.ws_messages?.values?.count || 0,
    avg_connect_latency: metrics.ws_connect_latency?.values?.avg || 0,
    p99_connect_latency: metrics.ws_connect_latency?.values?.['p(99)'] || 0,
    avg_message_latency: metrics.ws_message_latency?.values?.avg || 0,
    p99_message_latency: metrics.ws_message_latency?.values?.['p(99)'] || 0,
    error_rate: (metrics.errors?.values?.rate || 0) * 100,
  };
}

function textSummary(data, opts) {
  return JSON.stringify(data, null, 2);
}

function htmlReport(data, wsAnalysis) {
  return `
<!DOCTYPE html>
<html>
<head>
    <title>WebSocket连接测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 20px 0; padding: 15px; background: #f3e5f5; border-radius: 5px; }
        h1 { color: #7b1fa2; }
        h2 { color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #9c27b0; color: white; }
    </style>
</head>
<body>
    <h1>WebSocket连接测试报告</h1>
    <div class="metric">
        <h2>测试目标</h2>
        <p>并发连接: 1000个</p>
        <p>消息推送: 10000条/s</p>
    </div>
    <div class="metric">
        <h2>测试结果</h2>
        <p>总连接数: ${wsAnalysis.total_connections}</p>
        <p>总消息数: ${wsAnalysis.total_messages}</p>
        <p>平均连接延迟: ${wsAnalysis.avg_connect_latency.toFixed(2)}ms</p>
        <p>P99连接延迟: ${wsAnalysis.p99_connect_latency.toFixed(2)}ms</p>
        <p>平均消息延迟: ${wsAnalysis.avg_message_latency.toFixed(2)}ms</p>
        <p>P99消息延迟: ${wsAnalysis.p99_message_latency.toFixed(2)}ms</p>
        <p>错误率: ${wsAnalysis.error_rate.toFixed(2)}%</p>
    </div>
</body>
</html>
  `;
}
