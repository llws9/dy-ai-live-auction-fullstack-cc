import { useEffect, useMemo, useState } from 'react';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ReferenceLine,
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

export const theaterPreset: ChaosConfig = {
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
  const startDisabled = isChaosStartDisabled({ running, testID, progress, step });
  const narration = buildNarration({ step, progress, form, report: displayedReport });
  const anchors = useMemo(() => buildCurveAnchors(displayedReport?.buckets ?? []), [displayedReport]);
  const demoMetrics = useMemo(() => buildDemoMetrics(displayedReport), [displayedReport]);

  // 卸载时清理 WS 与全局 store
  useEffect(() => () => disconnect(), [disconnect]);

  const start = async (override?: ChaosConfig) => {
    const config = override ?? form;
    setError(null);
    setReport(null);
    setRunning(true);
    try {
      const id = await startChaos(config);
      const wsURL = await discoverWS(id);
      connect(wsURL, id);
      poll.start(id, setReport);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setRunning(false);
    }
  };

  const startTheaterMode = async () => {
    setForm(theaterPreset);
    await start(theaterPreset);
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

        <div style={{ display: 'flex', gap: 12, marginTop: 12, flexWrap: 'wrap' }}>
          <button type="button" disabled={startDisabled} onClick={() => start()} style={btnP(startDisabled)}>
            {describeChaosStartButton({ mode: 'manual', disabled: startDisabled, running })}
          </button>
          <button type="button" disabled={startDisabled} onClick={startTheaterMode} style={btnP(startDisabled)}>
            {describeChaosStartButton({ mode: 'theater', disabled: startDisabled, running })}
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
            <Metric label="峰值错误率" value={pctFromNumber(demoMetrics.peakErrorRatePct)} bad={(demoMetrics.peakErrorRatePct ?? 0) > 5} />
            <Metric label="损失 QPS" value={demoMetrics.lostQps == null ? '-' : demoMetrics.lostQps.toFixed(1)} bad={(demoMetrics.lostQps ?? 0) > 0} />
            <Metric label="恢复耗时(ms)" value={String(demoMetrics.recoveryMs ?? '-')} />
          </div>
          <ResilienceCurve report={displayedReport} anchors={anchors} narration={narration} demoMetrics={demoMetrics} />
        </section>
      )}
    </div>
  );
}

function pct(v?: number): string {
  if (v === undefined || v === null || Number.isNaN(v)) return '-';
  return `${(v * 100).toFixed(1)}%`;
}

function pctFromNumber(v?: number): string {
  if (v === undefined || v === null || Number.isNaN(v)) return '-';
  return `${v.toFixed(1)}%`;
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

export interface CurveAnchors {
  injectIndex?: number;
  slaBreachIndex?: number;
  recoverIndex?: number;
}

export interface DemoMetrics {
  peakErrorRatePct?: number;
  lostQps?: number;
  recoveryMs?: number;
}

export interface ReferenceLineLabels {
  inject: string;
  sla: string;
  recover: string;
}

export interface ReferenceLineLabelProps {
  value: string;
  position: 'insideTopLeft' | 'insideBottomLeft' | 'insideTopRight';
  offset: number;
  fill: string;
}

export interface ChaosLifecycleState {
  running: boolean;
  testID?: string | null;
  progress: number;
  step?: string | null;
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

export function buildNarration({
  step,
  progress,
  form,
  report,
}: {
  step?: string | null;
  progress: number;
  form: Partial<ChaosConfig>;
  report?: Report | null;
}): string {
  if (report && (step === 'done' || step === 'failed' || progress >= 100)) {
    return summarizeResilienceReport(report);
  }
  if (step === 'baseline') return '正在采集基线指标，建立健康水位...';
  if (step === 'inject') {
    const rate = Math.round((form.error_rate ?? theaterPreset.error_rate ?? 0) * 100);
    return `正在注入约 ${rate}% 错误率，观察系统反应...`;
  }
  if (step === 'recover') return '故障已移除，正在观测系统自愈...';
  return '等待启动韧性剧本，系统观测台就绪。';
}

export function buildCurveAnchors(buckets: Bucket[], slaThreshold = 0.05): CurveAnchors {
  const anchors: CurveAnchors = {};
  const injectIndex = buckets.findIndex((bucket) => bucket.phase === 'inject');
  const recoverIndex = buckets.findIndex((bucket) => bucket.phase === 'recover');
  const slaBreachIndex = buckets.findIndex((bucket) => bucketErrorRate(bucket) > slaThreshold);

  if (injectIndex >= 0) anchors.injectIndex = injectIndex;
  if (slaBreachIndex >= 0) anchors.slaBreachIndex = slaBreachIndex;
  if (recoverIndex >= 0) anchors.recoverIndex = recoverIndex;
  return anchors;
}

export function buildDemoMetrics(report?: Report | null): DemoMetrics {
  const buckets = report?.buckets ?? [];
  const injectBuckets = buckets.filter((bucket) => bucket.phase === 'inject');
  const peakErrorRatePct = injectBuckets.length
    ? Number((Math.max(...injectBuckets.map(bucketErrorRate)) * 100).toFixed(1))
    : undefined;
  const baselineQps = averageSuccessQps(buckets, 'baseline');
  const injectQps = averageSuccessQps(buckets, 'inject');
  const lostQps = baselineQps == null || injectQps == null ? undefined : Math.max(0, Number((baselineQps - injectQps).toFixed(1)));

  return {
    peakErrorRatePct,
    lostQps,
    recoveryMs: report?.recovery_latency_ms,
  };
}

export function buildReferenceLineLabels(metrics?: DemoMetrics): ReferenceLineLabels {
  return {
    inject: 'inject',
    sla: `peak ${pctFromNumber(metrics?.peakErrorRatePct)}`,
    recover: metrics?.lostQps == null
      ? `recover ${fmtMs(metrics?.recoveryMs)} / lost QPS -`
      : `recover ${fmtMs(metrics?.recoveryMs)} / -${fmtMetric(metrics.lostQps)}QPS`,
  };
}

export function buildReferenceLineLabelProps(labels: ReferenceLineLabels): Record<keyof ReferenceLineLabels, ReferenceLineLabelProps> {
  return {
    inject: {
      value: labels.inject,
      position: 'insideTopLeft',
      offset: 8,
      fill: theaterStyles.errorColor,
    },
    sla: {
      value: labels.sla,
      position: 'insideBottomLeft',
      offset: 24,
      fill: theaterStyles.warnColor,
    },
    recover: {
      value: labels.recover,
      position: 'insideTopRight',
      offset: 8,
      fill: theaterStyles.primaryColor,
    },
  };
}

export function isChaosStartDisabled({ running, testID, progress, step }: ChaosLifecycleState): boolean {
  if (running) return true;
  if (!testID) return false;
  return !(step === 'done' || step === 'failed' || progress >= 100);
}

export function describeChaosStartButton({
  mode,
  disabled,
  running,
}: {
  mode: 'manual' | 'theater';
  disabled: boolean;
  running: boolean;
}): string {
  if (running) return '启动中...';
  if (disabled) return mode === 'theater' ? '演示进行中...' : '实验进行中...';
  return mode === 'theater' ? './start_theater.sh' : '启动';
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

function bucketErrorRate(bucket: Bucket): number {
  const total = bucket.ok_count + bucket.fail_count;
  return total > 0 ? bucket.fail_count / total : 0;
}

function averageSuccessQps(buckets: Bucket[], phase: Bucket['phase']): number | undefined {
  const phaseBuckets = buckets.filter((bucket) => bucket.phase === phase);
  if (phaseBuckets.length === 0) return undefined;
  return phaseBuckets.reduce((sum, bucket) => sum + bucket.ok_count, 0) / phaseBuckets.length;
}

function isChaosPhase(step: string): step is Bucket['phase'] {
  return step === 'baseline' || step === 'inject' || step === 'recover';
}

function fmtMs(v?: number): string {
  return v == null || Number.isNaN(v) ? '-' : `${v}ms`;
}

function fmtMetric(v?: number): string {
  return v == null || Number.isNaN(v) ? '-' : v.toFixed(1);
}

function ResilienceCurve({
  report,
  anchors,
  narration,
  demoMetrics,
}: {
  report: Report;
  anchors?: CurveAnchors;
  narration?: string;
  demoMetrics?: DemoMetrics;
}) {
  const buckets = report.buckets ?? [];
  const series = useMemo(() => buildResilienceSeries(buckets), [buckets]);
  const spans = useMemo(() => buildPhaseSpans(series), [series]);
  const referenceLabels = useMemo(() => buildReferenceLineLabels(demoMetrics), [demoMetrics]);
  const referenceLabelProps = useMemo(() => buildReferenceLineLabelProps(referenceLabels), [referenceLabels]);

  if (series.length === 0) return null;

  return (
    <div>
      <div style={theaterStyles.curveTitle}>
        系统韧性曲线
      </div>
      {narration && (
        <div style={theaterStyles.narration}>
          <span style={theaterStyles.prompt}>{'>'}</span> {narration}
        </div>
      )}
      {demoMetrics && (
        <div style={theaterStyles.inlineMetrics}>
          <span>peak_error={pctFromNumber(demoMetrics.peakErrorRatePct)}</span>
          <span>lost_qps={demoMetrics.lostQps == null ? '-' : demoMetrics.lostQps.toFixed(1)}</span>
          <span>recover_ms={demoMetrics.recoveryMs ?? '-'}</span>
        </div>
      )}
      <div style={theaterStyles.summary}>
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
            {anchors?.injectIndex != null && (
              <ReferenceLine yAxisId="left" x={anchors.injectIndex} stroke={theaterStyles.errorColor} strokeDasharray="4 4" label={referenceLabelProps.inject} />
            )}
            {anchors?.slaBreachIndex != null && (
              <ReferenceLine yAxisId="left" x={anchors.slaBreachIndex} stroke={theaterStyles.warnColor} strokeDasharray="4 4" label={referenceLabelProps.sla} />
            )}
            {anchors?.recoverIndex != null && (
              <ReferenceLine yAxisId="left" x={anchors.recoverIndex} stroke={theaterStyles.primaryColor} strokeDasharray="4 4" label={referenceLabelProps.recover} />
            )}
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
            <Line yAxisId="left" type="monotone" dataKey="errorRatePct" stroke={theaterStyles.errorColor} strokeWidth={2} name="错误率" dot={false} />
            <Line yAxisId="right" type="monotone" dataKey="avgLatencyMs" stroke={theaterStyles.warnColor} strokeWidth={2} name="平均延迟" dot={false} />
            <Line yAxisId="left" type="monotone" dataKey="successQps" stroke={theaterStyles.primaryColor} strokeWidth={2} name="成功 QPS" dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
      <div style={theaterStyles.legendHint}>
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
  if (phase === 'baseline') return theaterStyles.successColor;
  if (phase === 'inject') return theaterStyles.errorColor;
  return theaterStyles.primaryColor;
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

export const theaterStyles: {
  errorColor: string;
  primaryColor: string;
  successColor: string;
  warnColor: string;
  curveTitle: React.CSSProperties;
  narration: React.CSSProperties;
  prompt: React.CSSProperties;
  inlineMetrics: React.CSSProperties;
  summary: React.CSSProperties;
  legendHint: React.CSSProperties;
} = {
  errorColor: 'var(--color-error-500)',
  primaryColor: 'var(--color-primary-500)',
  successColor: 'var(--color-success-500)',
  warnColor: 'var(--color-warn-500)',
  curveTitle: {
    marginBottom: 10,
    color: 'var(--color-text-1)',
    fontSize: 14,
    fontWeight: 700,
  },
  narration: {
    marginBottom: 10,
    padding: '10px 12px',
    border: '1px solid var(--color-text-1)',
    borderRadius: 'var(--radius-md)',
    background: 'var(--color-text-1)',
    color: 'var(--color-bg-2)',
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
    fontSize: 13,
    lineHeight: 1.6,
  },
  prompt: {
    color: 'var(--color-success-500)',
  },
  inlineMetrics: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 10,
    color: 'var(--color-text-2)',
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
    fontSize: 12,
  },
  summary: {
    marginBottom: 12,
    color: 'var(--color-text-2)',
    fontSize: 13,
    lineHeight: 1.6,
  },
  legendHint: {
    marginTop: 8,
    color: 'var(--color-text-3)',
    fontSize: 12,
  },
};
