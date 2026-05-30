import axios from 'axios';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

const http = axios.create({
  baseURL: API_BASE,
  timeout: 10000,
});

// 测试结果数据结构（与后端 model.TestResult 字段对应；gorm 默认字段名首字母大写）
export interface TestResult {
  ID: string;
  TestType: string;
  Status: 'running' | 'completed' | 'failed' | 'cancelled';
  ConfigJSON: string;
  ResultJSON: string;
  ReplayToken: string;
  ScriptName: string;
  ErrorMsg: string;
  CreatedAt: string;
  CompletedAt?: string | null;
}

export interface HistoryResp {
  total: number;
  items: TestResult[];
}

// 启动 dummy 任务
export async function startDummy(config: Record<string, unknown> = {}): Promise<string> {
  const r = await http.post<{ test_id: string }>('/test/dummy', config);
  return r.data.test_id;
}

// 启动压测任务（场景 A）
export interface PressureConfig {
  concurrent_users: number;
  duration_sec: number;
  target_auction_id: number;
  bid_amount: number;
  emit_interval_ms?: number;
}
export async function startPressure(config: PressureConfig): Promise<string> {
  const r = await http.post<{ test_id: string }>('/test/pressure', config);
  return r.data.test_id;
}

// 启动 E2E 全链路测试（场景 E）
export interface E2EConfig {
  seller_id?: number;
  bidder_ids?: number[];
  subscriber_id?: number;
  start_price?: number;
  increment?: number;
  duration?: number;
  poll_interval?: number; // ns（go time.Duration），可缺省
  poll_timeout?: number;
}
export async function startE2E(config: E2EConfig): Promise<string> {
  const r = await http.post<{ test_id: string }>('/test/e2e', config);
  return r.data.test_id;
}

// 启动防狙击测试（场景 F）
export interface AntiSnipeConfig {
  cases?: string[]; // 为空 → 跑全部 5 个
  bidder_ids?: number[];
}
export async function startAntiSnipe(config: AntiSnipeConfig = {}): Promise<string> {
  const r = await http.post<{ test_id: string }>('/test/antisnipe', config);
  return r.data.test_id;
}

// 启动外部回调可靠投递测试（场景 H）
export interface CallbackConfig {
  partner_url?: string;
  hmac_secret?: string;
  cases?: string[];
  max_retry?: number;
  timeout_ms?: number;
}
export async function startCallback(config: CallbackConfig = {}): Promise<string> {
  const r = await http.post<{ test_id: string }>('/test/callback', config);
  return r.data.test_id;
}

// 启动场景 G 故障注入测试
export interface ChaosConfig {
  probe_url?: string;
  probe_qps?: number;
  baseline_sec?: number;
  inject_sec?: number;
  recover_sec?: number;
  fault_type: 'latency' | 'error_rate' | 'disconnect';
  latency_ms?: number;
  jitter_ms?: number;
  error_rate?: number;
}
export async function startChaos(config: ChaosConfig): Promise<string> {
  const r = await http.post<{ test_id: string }>('/test/chaos', config);
  return r.data.test_id;
}

// 启动 M7.1 剧本：路径 :name = quickstart/antisnipe/reliability/chaos/fullshow
export async function startScript(name: string, extra: Record<string, unknown> = {}): Promise<string> {
  const r = await http.post<{ test_id: string }>(`/test/script/${name}`, extra);
  return r.data.test_id;
}

// M7.3 A/B 对比：同一 type 的两份 cfg 同时跑
export interface CompareReq {
  type: string;
  left: Record<string, unknown>;
  right: Record<string, unknown>;
}
export interface CompareResp {
  type: string;
  left_id: string;
  right_id: string;
}
export async function postCompare(req: CompareReq): Promise<CompareResp> {
  const r = await http.post<CompareResp>('/test/compare', req);
  return r.data;
}

export async function getStatus(id: string): Promise<TestResult> {
  const r = await http.get<TestResult>(`/test/status/${id}`);
  return r.data;
}

export async function getHistory(params: {
  test_type?: string;
  status?: string;
  page?: number;
  page_size?: number;
}): Promise<HistoryResp> {
  const r = await http.get<HistoryResp>('/test/history', { params });
  return r.data;
}

export async function getReport(id: string): Promise<TestResult> {
  const r = await http.get<TestResult>(`/test/report/${id}`);
  return r.data;
}

export async function cancelTest(id: string): Promise<void> {
  await http.post(`/test/cancel/${id}`);
}

// 获取 WS 真实地址（gateway 返回 endpoint discovery）
export async function discoverWS(testID: string): Promise<string> {
  const r = await http.get<{ data: { ws_url: string } }>(
    `${import.meta.env.VITE_WS_BASE || '/ws'}/test/progress`,
    { params: { test_id: testID } },
  );
  return r.data.data.ws_url;
}
