import { useEffect, useMemo, useState, type CSSProperties } from 'react';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  Legend,
  BarChart,
  Bar,
} from 'recharts';
import { startPressure, discoverWS, cancelTest } from '@/api/test';
import { useWSStore } from '@/store/wsStore';
import ProgressBar from '@/components/ProgressBar';
import { Metric } from '@/components/ui/Metric';
import { NumField } from '@/components/ui/Field';
import { cardStyle, titleStyle, primaryBtn, secondaryBtn } from '@/components/ui/styles';

interface PressureForm {
  concurrent_users: number;
  duration_sec: number;
  scenario: 'hot_auction' | 'throughput';
  fixture_count: number;
  bid_amount: number;
  emit_interval_ms: number;
}

interface BucketSnap {
  upper_ms: number;
  count: number;
}

interface ErrorCodeExplanation {
  code: string;
  count: number;
  title: string;
  detail: string;
  severity: 'info' | 'warning' | 'danger';
}

const defaultForm: PressureForm = {
  concurrent_users: 100,
  duration_sec: 30,
  scenario: 'hot_auction',
  fixture_count: 0,
  bid_amount: 100,
  emit_interval_ms: 1000,
};

export default function Pressure() {
  const [form, setForm] = useState<PressureForm>(defaultForm);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { connected, testID, progress, step, metrics, history, connect, disconnect } = useWSStore();

  // 卸载时清理 WS 与全局 store
  useEffect(() => () => disconnect(), [disconnect]);

  // 时序图数据：从 history 提炼 [{t, qps, p99}]
  const series = useMemo(
    () =>
      history.map((m, i) => ({
        t: i,
        qps: Number(m.metrics?.qps ?? 0),
        p99: Number(m.metrics?.p99_ms ?? 0),
        p95: Number(m.metrics?.p95_ms ?? 0),
        avg: Number(m.metrics?.avg_ms ?? 0),
      })),
    [history],
  );

  // 桶分布数据
  const buckets = useMemo<BucketSnap[]>(() => {
    const raw = (metrics.buckets as BucketSnap[] | undefined) ?? [];
    return raw.map((b) => ({
      upper_ms: b.upper_ms,
      count: Number(b.count) || 0,
    }));
  }, [metrics.buckets]);

  const scenarioInfo = form.scenario === 'throughput'
    ? {
        title: '吞吐压测',
        desc: '自动创建多个拍卖 fixture 分片，worker 分散出价，目标是测有效请求吞吐与延迟。',
      }
    : {
        title: '单拍卖热点冲突',
        desc: '100 人同时盲压同一个拍卖，保留业务冲突，用于观察出价被超越、锁竞争和热点排队。',
      };
  const errorExplanations = useMemo(
    () => explainErrorCodes(metrics.error_codes, Number(metrics.failure ?? 0)),
    [metrics.error_codes, metrics.failure],
  );
  const wsStatus = describeWSStatus(connected, progress, step);
  const failureMessage = failureMessageFromMetrics(metrics);
  const startDisabled = isPressureStartDisabled(running, testID, progress, step);

  const handleStart = async () => {
    if (startDisabled) return;
    setError(null);
    setRunning(true);
    try {
      const id = await startPressure(form);
      const wsURL = await discoverWS(id);
      connect(wsURL, id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setRunning(false);
    }
  };

  const handleCancel = async () => {
    if (!testID) return;
    try {
      await cancelTest(testID);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      disconnect();
    }
  };

  const setField = <K extends keyof PressureForm>(k: K, v: PressureForm[K]) =>
    setForm((s) => ({ ...s, [k]: v }));

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>压力测试（场景 A）</h1>

      <section style={cardStyle}>
        <h3 style={titleStyle}>场景</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 12, marginBottom: 14 }}>
          <button
            type="button"
            onClick={() => setField('scenario', 'hot_auction')}
            style={scenarioButton(form.scenario === 'hot_auction')}
          >
            <strong>单拍卖热点冲突</strong>
            <span>盲压一个拍卖，保留业务失败</span>
          </button>
          <button
            type="button"
            onClick={() => setField('scenario', 'throughput')}
            style={scenarioButton(form.scenario === 'throughput')}
          >
            <strong>吞吐压测</strong>
            <span>多拍卖分片，减少业务冲突</span>
          </button>
        </div>
        <div style={{ marginBottom: 16, color: '#475569', fontSize: 13, lineHeight: 1.6 }}>
          当前：<strong>{scenarioInfo.title}</strong>。{scenarioInfo.desc}
        </div>

        <h3 style={titleStyle}>参数</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12 }}>
          <NumField label="并发用户数" value={form.concurrent_users} min={1}
            onChange={(v) => setField('concurrent_users', v)} />
          <NumField label="持续时间(秒)" value={form.duration_sec} min={1}
            onChange={(v) => setField('duration_sec', v)} />
          <NumField label="出价金额" value={form.bid_amount} min={1}
            onChange={(v) => setField('bid_amount', v)} />
          {form.scenario === 'throughput' && (
            <NumField label="拍卖分片数(0=并发数)" value={form.fixture_count} min={0}
              onChange={(v) => setField('fixture_count', v)} />
          )}
          <NumField label="上报间隔(ms)" value={form.emit_interval_ms} min={100}
            onChange={(v) => setField('emit_interval_ms', v)} />
        </div>
        <div style={{ marginTop: 10, color: '#6b7280', fontSize: 13 }}>
          压测开始前会自动创建有效拍卖 fixture，并为每个压测用户注入合法 JWT。吞吐压测默认按并发用户数创建拍卖分片。
        </div>
        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <button type="button" disabled={startDisabled} onClick={handleStart} style={primaryBtn(startDisabled)}>
            {describePressureStartButton(running, testID, progress, step)}
          </button>
          <button type="button" disabled={!testID} onClick={handleCancel} style={secondaryBtn(!testID)}>
            取消
          </button>
        </div>
        {error && <div style={{ color: '#ef4444', marginTop: 12, fontSize: 13 }}>错误：{error}</div>}
      </section>

      <section style={cardStyle}>
        <h3 style={titleStyle}>实时进度</h3>
        <div style={{ marginBottom: 8, fontSize: 13, color: '#6b7280' }}>
          test_id: <code>{testID || '-'}</code> · WS: {wsStatus} · 步骤: {step || '-'}
        </div>
        <ProgressBar value={progress} label={`${progress}%`} />
        {failureMessage && (
          <div style={{ color: '#ef4444', marginTop: 10, fontSize: 13 }}>
            错误：{failureMessage}
          </div>
        )}
      </section>

      <section style={cardStyle}>
        <h3 style={titleStyle}>核心指标（实时）</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 12 }}>
          <Metric label="QPS" value={fmt(metrics.qps, 1)} />
          <Metric label="Avg (ms)" value={fmt(metrics.avg_ms)} />
          <Metric label="P50 (ms)" value={fmt(metrics.p50_ms)} />
          <Metric label="P95 (ms)" value={fmt(metrics.p95_ms)} />
          <Metric label="P99 (ms)" value={fmt(metrics.p99_ms)} />
          <Metric label="累计请求" value={fmt(metrics.total)} />
          <Metric label="成功" value={fmt(metrics.success)} />
          <Metric label="失败" value={fmt(metrics.failure)} />
          <Metric label="拍卖分片" value={fmt(metrics.fixture_count)} />
        </div>
        <div style={{ marginTop: 10, color: '#6b7280', fontSize: 13 }}>
          场景：{String(metrics.scenario ?? form.scenario)}
        </div>
        <ErrorCodePanel items={errorExplanations} />
      </section>

      <section style={cardStyle}>
        <h3 style={titleStyle}>QPS 与延迟时序</h3>
        <div style={{ width: '100%', height: 280 }}>
          <ResponsiveContainer>
            <LineChart data={series}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="t" tick={{ fontSize: 12 }} />
              <YAxis yAxisId="left" tick={{ fontSize: 12 }} />
              <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 12 }} />
              <Tooltip />
              <Legend />
              <Line yAxisId="left" type="monotone" dataKey="qps" stroke="#3b82f6" name="QPS" dot={false} />
              <Line yAxisId="right" type="monotone" dataKey="p99" stroke="#ef4444" name="P99(ms)" dot={false} />
              <Line yAxisId="right" type="monotone" dataKey="p95" stroke="#f59e0b" name="P95(ms)" dot={false} />
              <Line yAxisId="right" type="monotone" dataKey="avg" stroke="#10b981" name="Avg(ms)" dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section style={cardStyle}>
        <h3 style={titleStyle}>延迟分布（桶式直方图）</h3>
        <div style={{ width: '100%', height: 240 }}>
          <ResponsiveContainer>
            <BarChart data={buckets}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="upper_ms"
                tick={{ fontSize: 12 }}
                tickFormatter={(v: number) => (v < 0 ? '+∞' : `≤${v}ms`)}
              />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip />
              <Bar dataKey="count" fill="#6366f1" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </section>
    </div>
  );
}

function ErrorCodePanel({ items }: { items: ErrorCodeExplanation[] }) {
  if (items.length === 0) {
    return (
      <div style={{ marginTop: 12, color: '#64748b', fontSize: 13 }}>
        错误码解释：暂无失败请求。
      </div>
    );
  }

  return (
    <div style={{ marginTop: 14 }}>
      <div style={{ fontSize: 13, fontWeight: 700, color: '#334155', marginBottom: 8 }}>错误码解释</div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 10 }}>
        {items.map((item) => (
          <div key={item.code} style={errorCardStyle(item.severity)}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 10, alignItems: 'center' }}>
              <strong style={{ fontSize: 14 }}>Code {item.code}：{item.title}</strong>
              <span style={{ fontFamily: 'monospace', fontWeight: 700 }}>{item.count.toLocaleString()}</span>
            </div>
            <p style={{ margin: '8px 0 0', color: '#475569', fontSize: 13, lineHeight: 1.55 }}>
              {item.detail}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}

function scenarioButton(active: boolean): CSSProperties {
  return {
    border: `1px solid ${active ? '#2563eb' : '#e5e7eb'}`,
    background: active ? '#eff6ff' : '#fff',
    color: '#0f172a',
    borderRadius: 10,
    padding: '12px 14px',
    textAlign: 'left',
    cursor: 'pointer',
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
    boxShadow: active ? '0 0 0 2px rgba(37,99,235,0.12)' : 'none',
  };
}

function errorCardStyle(severity: ErrorCodeExplanation['severity']): CSSProperties {
  const palette = {
    info: { border: '#bfdbfe', background: '#eff6ff' },
    warning: { border: '#fde68a', background: '#fffbeb' },
    danger: { border: '#fecaca', background: '#fef2f2' },
  }[severity];
  return {
    border: `1px solid ${palette.border}`,
    background: palette.background,
    borderRadius: 10,
    padding: '10px 12px',
  };
}

function explainErrorCodes(v: unknown, failureTotal: number): ErrorCodeExplanation[] {
  if (!v || typeof v !== 'object') return [];
  const entries = Object.entries(v as Record<string, unknown>);
  return entries
    .map(([code, rawCount]) => {
      const count = Number(rawCount) || 0;
      const percent = failureTotal > 0 ? `，约占失败请求 ${(count / failureTotal * 100).toFixed(1)}%` : '';
      const base = explainErrorCode(code);
      return {
        code,
        count,
        title: base.title,
        detail: `${base.detail}${percent}。`,
        severity: base.severity,
      };
    })
    .sort((a, b) => b.count - a.count);
}

function explainErrorCode(code: string): Omit<ErrorCodeExplanation, 'code' | 'count'> {
  switch (code) {
    case '0':
      return {
        title: '客户端未收到 HTTP 状态',
        detail: '请求在客户端侧失败，常见原因是真实请求超时、连接中断或服务未及时返回。正常压测结束导致的 context cancel 已被剔除，不应计入该错误码',
        severity: 'warning',
      };
    case '400':
      return {
        title: '业务规则拒绝',
        detail: '请求已到达出价接口，但业务规则不接受。压测里最常见是同一拍卖分片内并发乱序，导致出价金额不足或已被其他用户超越',
        severity: 'info',
      };
    case '401':
      return {
        title: '认证失败',
        detail: 'JWT 缺失、无效或过期。压测场景出现该错误通常说明 JWT_SECRET、签名逻辑或网关认证链路不一致',
        severity: 'danger',
      };
    case '403':
      return {
        title: '权限不足',
        detail: '身份已识别但没有权限执行操作。压测出价出现该错误通常要检查用户角色、接口权限或 fixture 创建身份',
        severity: 'danger',
      };
    case '429':
      return {
        title: '网关限流',
        detail: '请求被 gateway 令牌桶限流拦截，说明压测流量超过当前限流阈值，不代表业务逻辑失败',
        severity: 'warning',
      };
    case '500':
      return {
        title: '服务端内部错误',
        detail: '请求进入服务端后触发系统级异常，需要检查 gateway/auction 日志，重点看 DB、Redis、分布式锁和事务错误',
        severity: 'danger',
      };
    case '502':
    case '503':
    case '504':
      return {
        title: '上游服务不可用或超时',
        detail: '网关无法正常从后端服务获得响应，通常与服务未启动、连接池耗尽、上游超时或本地进程异常有关',
        severity: 'danger',
      };
    default:
      return {
        title: '未分类错误',
        detail: '当前平台尚未内置该错误码解释，需要查看对应 HTTP 响应体和服务日志补充映射',
        severity: 'warning',
      };
  }
}

export function describeWSStatus(connected: boolean, progress: number, step: string): string {
  if (step === 'failed') return '压测失败，实时连接已结束';
  if (progress >= 100 || step === 'done') return '压测已完成，实时连接已结束';
  return connected ? '已连接' : '等待连接';
}

export function failureMessageFromMetrics(metrics: Record<string, unknown>): string | null {
  const error = metrics.error;
  return typeof error === 'string' && error.trim() ? error : null;
}

export function isPressureStartDisabled(running: boolean, testID: string | null, progress: number, step: string): boolean {
  if (running) return true;
  if (!testID) return false;
  return !isPressureTerminal(progress, step);
}

export function describePressureStartButton(running: boolean, testID: string | null, progress: number, step: string): string {
  if (running) return '启动中...';
  if (isPressureStartDisabled(running, testID, progress, step)) return '压测进行中';
  return '启动压测';
}

function isPressureTerminal(progress: number, step: string): boolean {
  return step === 'failed' || step === 'done' || progress >= 100;
}

function fmt(v: unknown, digits = 0): string {
  if (v == null) return '-';
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  return digits > 0 ? n.toFixed(digits) : n.toLocaleString();
}
