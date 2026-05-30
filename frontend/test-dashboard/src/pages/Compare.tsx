import { useEffect, useState } from 'react';
import { postCompare, getReport, type TestResult } from '@/api/test';
import ProgressBar from '@/components/ProgressBar';

const PRESETS: Record<string, { left: Record<string, unknown>; right: Record<string, unknown>; desc: string }> = {
  pressure: {
    desc: '左：50 并发 / 右：200 并发',
    left: { concurrent_users: 50, duration_sec: 10, target_auction_id: 9001, bid_amount: 100, emit_interval_ms: 200 },
    right: { concurrent_users: 200, duration_sec: 10, target_auction_id: 9001, bid_amount: 100, emit_interval_ms: 200 },
  },
  chaos: {
    desc: '左：错误率 0.2 / 右：错误率 0.7',
    left: { fault_type: 'error_rate', error_rate: 0.2, baseline_sec: 3, inject_sec: 6, recover_sec: 4, probe_qps: 20 },
    right: { fault_type: 'error_rate', error_rate: 0.7, baseline_sec: 3, inject_sec: 6, recover_sec: 4, probe_qps: 20 },
  },
  antisnipe: {
    desc: '两侧默认全用例（演示对比页可正常并行）',
    left: { cases: ['within_window'] },
    right: { cases: ['outside_window'] },
  },
};

type Side = 'left' | 'right';

export default function Compare() {
  const [type, setType] = useState<'pressure' | 'chaos' | 'antisnipe'>('pressure');
  const [leftRaw, setLeftRaw] = useState<string>(JSON.stringify(PRESETS.pressure.left, null, 2));
  const [rightRaw, setRightRaw] = useState<string>(JSON.stringify(PRESETS.pressure.right, null, 2));
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [leftID, setLeftID] = useState<string | null>(null);
  const [rightID, setRightID] = useState<string | null>(null);
  const [leftRes, setLeftRes] = useState<TestResult | null>(null);
  const [rightRes, setRightRes] = useState<TestResult | null>(null);

  const applyPreset = (k: typeof type) => {
    setType(k);
    setLeftRaw(JSON.stringify(PRESETS[k].left, null, 2));
    setRightRaw(JSON.stringify(PRESETS[k].right, null, 2));
  };

  const start = async () => {
    setError(null);
    setLeftRes(null);
    setRightRes(null);
    setRunning(true);
    try {
      const left = JSON.parse(leftRaw);
      const right = JSON.parse(rightRaw);
      const r = await postCompare({ type, left, right });
      setLeftID(r.left_id);
      setRightID(r.right_id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setRunning(false);
    }
  };

  // 轮询两边结果
  useEffect(() => {
    if (!leftID && !rightID) return;
    let stopped = false;
    const tick = async () => {
      if (stopped) return;
      const tasks: Array<Promise<void>> = [];
      if (leftID) {
        tasks.push(
          getReport(leftID).then(setLeftRes).catch(() => {
            /* ignore */
          }),
        );
      }
      if (rightID) {
        tasks.push(
          getReport(rightID).then(setRightRes).catch(() => {
            /* ignore */
          }),
        );
      }
      await Promise.all(tasks);
    };
    tick();
    const handle = setInterval(tick, 1000);
    return () => {
      stopped = true;
      clearInterval(handle);
    };
  }, [leftID, rightID]);

  const isDone = (r: TestResult | null) =>
    !!r && (r.Status === 'completed' || r.Status === 'failed' || r.Status === 'cancelled');
  const allDone = isDone(leftRes) && isDone(rightRes);

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>A/B 对比模式</h1>

      <section style={card}>
        <h3 style={title}>预设</h3>
        <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
          {(['pressure', 'chaos', 'antisnipe'] as const).map((k) => (
            <button key={k} type="button" onClick={() => applyPreset(k)} style={chipBtn(type === k)}>
              {k}
            </button>
          ))}
          <span style={{ marginLeft: 12, fontSize: 13, color: '#6b7280' }}>{PRESETS[type].desc}</span>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <CfgEditor side="left" raw={leftRaw} setRaw={setLeftRaw} />
          <CfgEditor side="right" raw={rightRaw} setRaw={setRightRaw} />
        </div>
        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <button type="button" disabled={running} onClick={start} style={btnP(running)}>
            {running ? '提交中...' : '同时启动 A/B'}
          </button>
          {allDone && (
            <span style={{ fontSize: 13, color: '#10b981', alignSelf: 'center' }}>两侧均已完成</span>
          )}
        </div>
        {error && <div style={{ color: '#ef4444', marginTop: 12, fontSize: 13 }}>{error}</div>}
      </section>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        <SidePanel side="left" id={leftID} res={leftRes} />
        <SidePanel side="right" id={rightID} res={rightRes} />
      </div>
    </div>
  );
}

function CfgEditor({ side, raw, setRaw }: { side: Side; raw: string; setRaw: (v: string) => void }) {
  return (
    <div>
      <div style={{ fontSize: 13, color: '#6b7280', marginBottom: 4 }}>{side.toUpperCase()} cfg</div>
      <textarea
        value={raw}
        onChange={(e) => setRaw(e.target.value)}
        rows={10}
        spellCheck={false}
        style={{
          width: '100%',
          fontFamily: 'monospace',
          fontSize: 12,
          padding: 8,
          border: '1px solid #d1d5db',
          borderRadius: 6,
        }}
      />
    </div>
  );
}

function SidePanel({ side, id, res }: { side: Side; id: string | null; res: TestResult | null }) {
  const status = res?.Status ?? (id ? 'running' : '-');
  const progress = res?.Status === 'completed' ? 100 : res?.Status === 'failed' ? 100 : 0;
  let summary: Record<string, unknown> = {};
  if (res?.ResultJSON) {
    try {
      summary = JSON.parse(res.ResultJSON);
    } catch {
      /* ignore */
    }
  }
  return (
    <section style={{ ...card, borderColor: side === 'left' ? '#3b82f6' : '#f59e0b' }}>
      <h3 style={title}>{side.toUpperCase()}</h3>
      <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 8 }}>
        test_id: <code>{id ?? '-'}</code> · 状态: <strong>{status}</strong>
      </div>
      <ProgressBar value={progress} label={`${status}`} />
      <pre
        style={{
          background: '#0f172a',
          color: '#e2e8f0',
          fontSize: 11,
          padding: 8,
          borderRadius: 6,
          marginTop: 12,
          maxHeight: 300,
          overflow: 'auto',
        }}
      >
        {Object.keys(summary).length > 0 ? JSON.stringify(summary, null, 2) : '等待结果...'}
      </pre>
    </section>
  );
}

const card: React.CSSProperties = { padding: 16, border: '1px solid #e5e7eb', borderRadius: 8, marginBottom: 16 };
const title: React.CSSProperties = { fontSize: 16, marginBottom: 12 };
const btnP = (d: boolean): React.CSSProperties => ({
  padding: '8px 16px',
  background: 'var(--color-primary, #3b82f6)',
  color: '#fff',
  border: 'none',
  borderRadius: 6,
  cursor: d ? 'not-allowed' : 'pointer',
  opacity: d ? 0.6 : 1,
});
const chipBtn = (active: boolean): React.CSSProperties => ({
  padding: '4px 12px',
  borderRadius: 16,
  border: '1px solid',
  borderColor: active ? '#3b82f6' : '#d1d5db',
  background: active ? '#3b82f6' : '#fff',
  color: active ? '#fff' : '#1f2937',
  cursor: 'pointer',
  fontSize: 13,
});
