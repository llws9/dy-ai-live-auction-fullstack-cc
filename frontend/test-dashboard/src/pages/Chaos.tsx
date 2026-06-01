import { useEffect, useState } from 'react';
import { startChaos, discoverWS, cancelTest, type ChaosConfig } from '@/api/test';
import { useWSStore } from '@/store/wsStore';
import { usePollReport } from '@/hooks/usePollReport';
import ProgressBar from '@/components/ProgressBar';
import { Metric } from '@/components/ui/Metric';
import { NumField } from '@/components/ui/Field';
import { cardStyle as card, titleStyle as title, inputStyle as input, primaryBtn as btnP, secondaryBtn as btnS } from '@/components/ui/styles';

interface Bucket {
  ts: string;
  phase: 'baseline' | 'inject' | 'recover';
  ok_count: number;
  fail_count: number;
  avg_latency_ms: number;
}

interface Report {
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
  const { connected, testID, progress, step, connect, disconnect } = useWSStore();
  const poll = usePollReport<Report>();

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

      {report && report.buckets && report.buckets.length > 0 && (
        <section style={card}>
          <h3 style={title}>报告</h3>
          <div style={{ ...grid, marginBottom: 16 }}>
            <Metric label="基线错误率" value={pct(report.baseline_error_rate)} />
            <Metric label="注入错误率" value={pct(report.inject_error_rate)} bad={(report.inject_error_rate ?? 0) > 0.05} />
            <Metric label="恢复错误率" value={pct(report.recover_error_rate)} bad={(report.recover_error_rate ?? 0) > 0.05} />
            <Metric label="检测延迟(ms)" value={String(report.detection_latency_ms ?? '-')} />
            <Metric label="恢复延迟(ms)" value={String(report.recovery_latency_ms ?? '-')} />
            <Metric label="结论" value={report.all_ok ? 'PASS' : 'FAIL'} ok={report.all_ok} />
          </div>
          <ErrorChart buckets={report.buckets} />
        </section>
      )}
    </div>
  );
}

function pct(v?: number): string {
  if (v === undefined || v === null || Number.isNaN(v)) return '-';
  return `${(v * 100).toFixed(1)}%`;
}

function ErrorChart({ buckets }: { buckets: Bucket[] }) {
  const W = 720;
  const H = 200;
  const PAD = 32;
  const max = Math.max(1, ...buckets.map((b) => b.ok_count + b.fail_count));
  const dx = (W - PAD * 2) / Math.max(1, buckets.length - 1);
  const xy = (i: number, c: number) => [PAD + i * dx, H - PAD - (c / max) * (H - PAD * 2)] as const;
  const phaseColor = (p: Bucket['phase']) =>
    p === 'baseline' ? '#10b981' : p === 'inject' ? '#ef4444' : '#3b82f6';

  return (
    <div style={{ overflowX: 'auto' }}>
      <svg width={W} height={H} style={{ background: '#fafafa', borderRadius: 6 }}>
        <line x1={PAD} y1={H - PAD} x2={W - PAD} y2={H - PAD} stroke="#d1d5db" />
        <line x1={PAD} y1={PAD} x2={PAD} y2={H - PAD} stroke="#d1d5db" />
        {buckets.map((b, i) => {
          const total = b.ok_count + b.fail_count;
          const [x, yTop] = xy(i, total);
          const [, yFail] = xy(i, b.fail_count);
          const w = Math.max(2, dx * 0.6);
          return (
            <g key={i}>
              <rect x={x - w / 2} y={yTop} width={w} height={H - PAD - yTop} fill={phaseColor(b.phase)} opacity={0.25} />
              <rect x={x - w / 2} y={yFail} width={w} height={H - PAD - yFail} fill="#ef4444" opacity={0.85} />
            </g>
          );
        })}
        <text x={PAD} y={PAD - 8} fontSize="11" fill="#6b7280">
          柱：每秒请求数；红：失败数；绿/红/蓝底：阶段
        </text>
      </svg>
    </div>
  );
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
