import { useEffect, useMemo, useState } from 'react';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ReferenceArea,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { startChaos, discoverWS, cancelTest, type ChaosConfig } from '@/api/test';
import { useWSStore } from '@/store/wsStore';
import { usePollReport } from '@/hooks/usePollReport';
import ProgressBar from '@/components/ProgressBar';
import { Metric } from '@/components/ui/Metric';
import { NumField } from '@/components/ui/Field';
import { cardStyle as card, titleStyle as title, inputStyle as input, primaryBtn as btnP, secondaryBtn as btnS } from '@/components/ui/styles';

export interface Bucket {
  ts: string;
  phase: 'baseline' | 'inject' | 'recover';
  ok_count: number;
  fail_count: number;
  avg_latency_ms: number;
}

export interface Report {
  profile?: { type: string; latency_ms?: number; error_rate?: number };
  buckets?: Bucket[];
  baseline_error_rate?: number;
  inject_error_rate?: number;
  recover_error_rate?: number;
  detection_latency_ms?: number;
  recovery_latency_ms?: number;
  all_ok?: boolean;
  error?: string;
}

const defaults: ChaosConfig = {
  fault_type: 'error_rate',
  probe_qps: 20,
  baseline_sec: 3,
  inject_sec: 8,
  recover_sec: 5,
  error_rate: 0.5,
  latency_ms: 0,
  jitter_ms: 0,
};

export default function Chaos() {
  const [form, setForm] = useState<ChaosConfig>(defaults);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<Report | null>(null);
  const { connected, testID, progress, step, history, connect, disconnect } = useWSStore();
  const poll = usePollReport<Report>();
  const liveBuckets = useMemo(() => buildBucketsFromProgressHistory(history), [history]);
  const displayedReport = useMemo(
    () => (report?.buckets?.length ? report : buildLiveResilienceReport(liveBuckets)),
    [report, liveBuckets],
  );
  const faultImplementation = describeFaultImplementation(form.fault_type);

  // 卸载时清理 WS 与全局 store
  useEffect(() => () => disconnect(), [disconnect]);

  const start = async () => {
    setError(null);
    setReport(null);
    setRunning(true);
    try {
      const id = await startChaos(form);
      const wsURL = await discoverWS(id);
      connect(wsURL, id);
      poll.start(id, setReport);
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
    } finally {
      disconnect();
    }
  };

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>故障注入（场景 G - MVP 进程内）</h1>

      <section style={card}>
        <h3 style={title}>故障参数</h3>
        <div style={grid}>
          <Select
            label="故障类型"
            value={form.fault_type}
            options={[
              { v: 'error_rate', l: '错误率注入' },
              { v: 'latency', l: '延迟注入' },
              { v: 'disconnect', l: '强制断连' },
            ]}
            onChange={(v) => setForm({ ...form, fault_type: v as ChaosConfig['fault_type'] })}
          />
          <NumField label="Probe QPS" value={form.probe_qps ?? 20} min={1} onChange={(v) => setForm({ ...form, probe_qps: v })} />
          <NumField label="基线时长(s)" value={form.baseline_sec ?? 3} min={1} onChange={(v) => setForm({ ...form, baseline_sec: v })} />
          <NumField label="注入时长(s)" value={form.inject_sec ?? 8} min={1} onChange={(v) => setForm({ ...form, inject_sec: v })} />
          <NumField label="恢复时长(s)" value={form.recover_sec ?? 5} min={1} onChange={(v) => setForm({ ...form, recover_sec: v })} />
          {form.fault_type === 'error_rate' && (
            <NumField
              label="错误率(0-1)"
              value={form.error_rate ?? 0.5}
              step={0.05}
              min={0}
              onChange={(v) => setForm({ ...form, error_rate: v })}
            />
          )}
          {form.fault_type === 'latency' && (
            <>
              <NumField label="延迟基础(ms)" value={form.latency_ms ?? 200} min={0} onChange={(v) => setForm({ ...form, latency_ms: v })} />
              <NumField label="抖动(ms)" value={form.jitter_ms ?? 0} min={0} onChange={(v) => setForm({ ...form, jitter_ms: v })} />
            </>
          )}
        </div>
        <div style={faultExplanationStyle}>
          <strong>系统实现方式：</strong>{faultImplementation}
        </div>

        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <button type="button" disabled={running} onClick={start} style={btnP(running)}>
            {running ? '启动中...' : '启动'}
          </button>
          <button type="button" disabled={!testID} onClick={handleCancel} style={btnS(!testID)}>
            取消
          </button>
        </div>
        {error && <div style={{ color: '#ef4444', marginTop: 12, fontSize: 13 }}>{error}</div>}
      </section>

      <section style={card}>
        <h3 style={title}>实时进度</h3>
        <div style={{ marginBottom: 8, fontSize: 13, color: '#6b7280' }}>
          test_id: <code>{testID || '-'}</code> · WS: {connected ? '已连接' : '未连接'} · 阶段: {step || '-'}
        </div>
        <ProgressBar value={progress} label={`${progress}%`} />
      </section>

      {displayedReport && displayedReport.buckets && displayedReport.buckets.length > 0 && (
        <section style={card}>
          <h3 style={title}>{report ? '报告' : '实时韧性观测'}</h3>
          <div style={{ ...grid, marginBottom: 16 }}>
            <Metric label="基线错误率" value={pct(displayedReport.baseline_error_rate)} />
            <Metric label="注入错误率" value={pct(displayedReport.inject_error_rate)} bad={(displayedReport.inject_error_rate ?? 0) > 0.05} />
            <Metric label="恢复错误率" value={pct(displayedReport.recover_error_rate)} bad={(displayedReport.recover_error_rate ?? 0) > 0.05} />
            <Metric label="检测延迟(ms)" value={String(displayedReport.detection_latency_ms ?? '-')} />
            <Metric label="恢复延迟(ms)" value={String(displayedReport.recovery_latency_ms ?? '-')} />
            <Metric label="结论" value={displayedReport.all_ok == null ? '-' : displayedReport.all_ok ? 'PASS' : 'FAIL'} ok={displayedReport.all_ok} />
          </div>
          <ResilienceCurve report={displayedReport} />
        </section>
      )}
    </div>
  );
}

function pct(v?: number): string {
  if (v === undefined || v === null || Number.isNaN(v)) return '-';
  return `${(v * 100).toFixed(1)}%`;
}

interface ResiliencePoint {
  index: number;
  phase: Bucket['phase'];
  errorRatePct: number;
  avgLatencyMs: number;
  successQps: number;
  total: number;
}

interface PhaseSpan {
  phase: Bucket['phase'];
  start: number;
  end: number;
}

export function buildResilienceSeries(buckets: Bucket[]): ResiliencePoint[] {
  return buckets.map((bucket, index) => {
    const total = bucket.ok_count + bucket.fail_count;
    return {
      index,
      phase: bucket.phase,
      errorRatePct: total > 0 ? Number(((bucket.fail_count / total) * 100).toFixed(1)) : 0,
      avgLatencyMs: bucket.avg_latency_ms,
      successQps: bucket.ok_count,
      total,
    };
  });
}

export function summarizeResilienceReport(report: Report): string {
  return `故障注入后错误率从 ${pct(report.baseline_error_rate)} 上升到 ${pct(report.inject_error_rate)}，恢复阶段回落到 ${pct(report.recover_error_rate)}，检测延迟 ${fmtMs(report.detection_latency_ms)}，恢复延迟 ${fmtMs(report.recovery_latency_ms)}。`;
}

export function buildBucketsFromProgressHistory<T extends { step: string; metrics?: Record<string, unknown>; ts: number }>(
  messages: T[],
): Bucket[] {
  return messages.flatMap((message) => {
    if (!isChaosPhase(message.step)) return [];
    const metrics = message.metrics ?? {};
    return [{
      ts: new Date(message.ts).toISOString(),
      phase: message.step,
      ok_count: Number(metrics.ok ?? 0),
      fail_count: Number(metrics.fail ?? 0),
      avg_latency_ms: Number(metrics.avg_latency_ms ?? 0),
    }];
  });
}

export function describeFaultImplementation(faultType: string): string {
  switch (faultType) {
    case 'error_rate':
      return 'test-service 在 ChaosTransport 中按概率短路探测请求并返回 503，用来模拟下游间歇性错误；业务服务不被真实打挂，实验可快速恢复。';
    case 'latency':
      return 'test-service 在请求转发前执行 sleep，并可叠加 jitter，用来模拟慢下游、网络抖动或连接池排队导致的尾延迟抬升。';
    case 'disconnect':
      return 'test-service 在传输层主动制造连接中断，模拟上游连接被重置、网关断连或依赖不可达。';
    default:
      return '当前故障类型通过 test-service 进程内 ChaosTransport 注入，验证故障配置、实时观测和恢复判定闭环。';
  }
}

function buildLiveResilienceReport(buckets: Bucket[]): Report | null {
  if (buckets.length === 0) return null;
  return {
    buckets,
    baseline_error_rate: phaseErrorRate(buckets, 'baseline'),
    inject_error_rate: phaseErrorRate(buckets, 'inject'),
    recover_error_rate: phaseErrorRate(buckets, 'recover'),
  };
}

function phaseErrorRate(buckets: Bucket[], phase: Bucket['phase']): number | undefined {
  const totals = buckets
    .filter((bucket) => bucket.phase === phase)
    .reduce(
      (acc, bucket) => ({
        ok: acc.ok + bucket.ok_count,
        fail: acc.fail + bucket.fail_count,
      }),
      { ok: 0, fail: 0 },
    );
  const total = totals.ok + totals.fail;
  return total > 0 ? totals.fail / total : undefined;
}

function isChaosPhase(step: string): step is Bucket['phase'] {
  return step === 'baseline' || step === 'inject' || step === 'recover';
}

function fmtMs(v?: number): string {
  return v == null || Number.isNaN(v) ? '-' : `${v}ms`;
}

function ResilienceCurve({ report }: { report: Report }) {
  const buckets = report.buckets ?? [];
  const series = useMemo(() => buildResilienceSeries(buckets), [buckets]);
  const spans = useMemo(() => buildPhaseSpans(series), [series]);

  if (series.length === 0) return null;

  return (
    <div>
      <div style={{ marginBottom: 10, color: '#0f172a', fontSize: 14, fontWeight: 700 }}>
        系统韧性曲线
      </div>
      <div style={{ marginBottom: 12, color: '#475569', fontSize: 13, lineHeight: 1.6 }}>
        {summarizeResilienceReport(report)}
      </div>
      <div style={{ width: '100%', height: 300 }}>
        <ResponsiveContainer>
          <LineChart data={series} margin={{ top: 12, right: 24, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" />
            {spans.map((span) => (
              <ReferenceArea
                key={`${span.phase}-${span.start}`}
                x1={span.start - 0.5}
                x2={span.end + 0.5}
                yAxisId="left"
                fill={phaseColor(span.phase)}
                fillOpacity={0.08}
                strokeOpacity={0}
              />
            ))}
            <XAxis dataKey="index" tickFormatter={(v: number) => String(v + 1)} tick={{ fontSize: 12 }} />
            <YAxis yAxisId="left" tick={{ fontSize: 12 }} />
            <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 12 }} />
            <Tooltip
              formatter={(value, name) => {
                const n = Number(value ?? 0);
                const label = String(name);
                if (label === '错误率') return [`${n.toFixed(1)}%`, label];
                if (label === '平均延迟') return [`${n.toFixed(0)}ms`, label];
                return [n.toLocaleString(), label];
              }}
              labelFormatter={(_, payload) => {
                const point = payload?.[0]?.payload as ResiliencePoint | undefined;
                return point ? `第 ${point.index + 1} 秒 · ${phaseLabel(point.phase)}` : '';
              }}
            />
            <Legend />
            <Line yAxisId="left" type="monotone" dataKey="errorRatePct" stroke="#ef4444" strokeWidth={2} name="错误率" dot={false} />
            <Line yAxisId="right" type="monotone" dataKey="avgLatencyMs" stroke="#8b5cf6" strokeWidth={2} name="平均延迟" dot={false} />
            <Line yAxisId="left" type="monotone" dataKey="successQps" stroke="#2563eb" strokeWidth={2} name="成功 QPS" dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
      <div style={{ marginTop: 8, color: '#64748b', fontSize: 12 }}>
        背景：绿色 baseline / 红色 inject / 蓝色 recover；红线看故障冲击，紫线看延迟抬升，蓝线看有效吞吐恢复。
      </div>
    </div>
  );
}

function buildPhaseSpans(series: ResiliencePoint[]): PhaseSpan[] {
  if (series.length === 0) return [];
  const spans: PhaseSpan[] = [];
  let start = series[0].index;
  let phase = series[0].phase;
  for (const point of series.slice(1)) {
    if (point.phase !== phase) {
      spans.push({ phase, start, end: point.index - 1 });
      start = point.index;
      phase = point.phase;
    }
  }
  spans.push({ phase, start, end: series[series.length - 1].index });
  return spans;
}

function phaseColor(phase: Bucket['phase']): string {
  if (phase === 'baseline') return '#10b981';
  if (phase === 'inject') return '#ef4444';
  return '#3b82f6';
}

function phaseLabel(phase: Bucket['phase']): string {
  if (phase === 'baseline') return '基线';
  if (phase === 'inject') return '故障注入';
  return '恢复';
}

function Select({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: { v: string; l: string }[];
  onChange: (v: string) => void;
}) {
  return (
    <label style={{ display: 'flex', flexDirection: 'column', fontSize: 13 }}>
      <span style={{ color: '#6b7280', marginBottom: 4 }}>{label}</span>
      <select value={value} onChange={(e) => onChange(e.target.value)} style={input}>
        {options.map((o) => (
          <option key={o.v} value={o.v}>
            {o.l}
          </option>
        ))}
      </select>
    </label>
  );
}

const grid: React.CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))',
  gap: 12,
};

const faultExplanationStyle: React.CSSProperties = {
  marginTop: 12,
  padding: '10px 12px',
  border: '1px solid #bfdbfe',
  borderRadius: 8,
  background: '#eff6ff',
  color: '#334155',
  fontSize: 13,
  lineHeight: 1.6,
};
