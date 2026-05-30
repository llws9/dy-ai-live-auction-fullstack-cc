import { useMemo, useState } from 'react';
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

interface PressureForm {
  concurrent_users: number;
  duration_sec: number;
  target_auction_id: number;
  bid_amount: number;
  emit_interval_ms: number;
}

interface BucketSnap {
  upper_ms: number;
  count: number;
}

const defaultForm: PressureForm = {
  concurrent_users: 100,
  duration_sec: 30,
  target_auction_id: 1,
  bid_amount: 100,
  emit_interval_ms: 1000,
};

export default function Pressure() {
  const [form, setForm] = useState<PressureForm>(defaultForm);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { connected, testID, progress, step, metrics, history, connect, disconnect } = useWSStore();

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

  const handleStart = async () => {
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
        <h3 style={titleStyle}>参数</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12 }}>
          <NumField label="并发用户数" value={form.concurrent_users} min={1}
            onChange={(v) => setField('concurrent_users', v)} />
          <NumField label="持续时间(秒)" value={form.duration_sec} min={1}
            onChange={(v) => setField('duration_sec', v)} />
          <NumField label="目标拍卖 ID" value={form.target_auction_id} min={1}
            onChange={(v) => setField('target_auction_id', v)} />
          <NumField label="出价金额" value={form.bid_amount} min={1}
            onChange={(v) => setField('bid_amount', v)} />
          <NumField label="上报间隔(ms)" value={form.emit_interval_ms} min={100}
            onChange={(v) => setField('emit_interval_ms', v)} />
        </div>
        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <button type="button" disabled={running} onClick={handleStart} style={primaryBtn(running)}>
            {running ? '启动中...' : '启动压测'}
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
          test_id: <code>{testID || '-'}</code> · WS: {connected ? '已连接' : '未连接'} · 步骤: {step || '-'}
        </div>
        <ProgressBar value={progress} label={`${progress}%`} />
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
        </div>
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

function NumField({
  label,
  value,
  min,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  onChange: (v: number) => void;
}) {
  return (
    <label style={{ display: 'flex', flexDirection: 'column', fontSize: 13 }}>
      <span style={{ color: '#6b7280', marginBottom: 4 }}>{label}</span>
      <input
        type="number"
        value={value}
        min={min}
        onChange={(e) => onChange(Number(e.target.value) || min)}
        style={{
          padding: '6px 10px',
          border: '1px solid #d1d5db',
          borderRadius: 6,
          fontSize: 14,
        }}
      />
    </label>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ background: '#f8fafc', border: '1px solid #e5e7eb', borderRadius: 6, padding: '10px 12px' }}>
      <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 4 }}>{label}</div>
      <div style={{ fontSize: 18, fontFamily: 'monospace', fontWeight: 600 }}>{value}</div>
    </div>
  );
}

function fmt(v: unknown, digits = 0): string {
  if (v == null) return '-';
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  return digits > 0 ? n.toFixed(digits) : n.toLocaleString();
}

const cardStyle: React.CSSProperties = {
  padding: 16,
  border: '1px solid #e5e7eb',
  borderRadius: 8,
  marginBottom: 16,
};

const titleStyle: React.CSSProperties = { fontSize: 16, marginBottom: 12 };

const primaryBtn = (disabled: boolean): React.CSSProperties => ({
  padding: '8px 16px',
  background: 'var(--color-primary, #3b82f6)',
  color: '#fff',
  border: 'none',
  borderRadius: 6,
  cursor: disabled ? 'not-allowed' : 'pointer',
  opacity: disabled ? 0.6 : 1,
});

const secondaryBtn = (disabled: boolean): React.CSSProperties => ({
  padding: '8px 16px',
  background: '#fff',
  color: '#1f2937',
  border: '1px solid #d1d5db',
  borderRadius: 6,
  cursor: disabled ? 'not-allowed' : 'pointer',
  opacity: disabled ? 0.6 : 1,
});
