import { useMemo, useState } from 'react';
import { startAntiSnipe, discoverWS, cancelTest, getReport } from '@/api/test';
import { useWSStore } from '@/store/wsStore';
import ProgressBar from '@/components/ProgressBar';
import AntiSnipeTimeline, { AntiSnipeTimelineEvent } from '@/components/AntiSnipeTimeline';

const ALL_CASES = [
  { key: 'last_second', label: '末刻出价触发延时' },
  { key: 'delay_cap', label: '延时累计触达上限' },
  { key: 'multi_user_chain', label: '多用户连环触发' },
  { key: 'safe_period', label: '安全期不触发' },
  { key: 'capped_no_extend', label: '已封顶不再延长' },
];

interface AntiSnipeForm {
  bidder_ids: string;
  cases: string[];
}

interface CaseReport {
  name: string;
  ok: boolean;
  message?: string;
  report?: {
    auction_id?: number;
    original_end_time?: string;
    actual_end_time?: string;
    triggered_count?: number;
    bid_count?: number;
    delay_used_ms?: number;
    timeline?: AntiSnipeTimelineEvent[];
  };
}

interface ScenarioReport {
  cases?: CaseReport[];
  all_ok?: boolean;
  error?: string;
}

const defaultForm: AntiSnipeForm = {
  bidder_ids: '1001,1002,1003,1004,1005',
  cases: ALL_CASES.map((c) => c.key),
};

export default function AntiSnipe() {
  const [form, setForm] = useState<AntiSnipeForm>(defaultForm);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<ScenarioReport | null>(null);
  const [activeCase, setActiveCase] = useState<string | null>(null);
  const { connected, testID, progress, step, history, connect, disconnect } = useWSStore();

  const lastWS = useMemo(() => history[history.length - 1], [history]);

  const handleStart = async () => {
    setError(null);
    setReport(null);
    setActiveCase(null);
    setRunning(true);
    try {
      const bidders = form.bidder_ids
        .split(',')
        .map((s) => Number(s.trim()))
        .filter((n) => !Number.isNaN(n) && n > 0);
      const id = await startAntiSnipe({
        cases: form.cases,
        bidder_ids: bidders,
      });
      const wsURL = await discoverWS(id);
      connect(wsURL, id);
      pollReport(id, (r) => {
        setReport(r);
        if (r.cases && r.cases.length > 0) {
          setActiveCase(r.cases[0].name);
        }
      });
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

  const toggleCase = (k: string) => {
    setForm((s) => ({
      ...s,
      cases: s.cases.includes(k) ? s.cases.filter((x) => x !== k) : [...s.cases, k],
    }));
  };

  const activeReport = report?.cases?.find((c) => c.name === activeCase);

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>防狙击延时（场景 F）</h1>

      <section style={cardStyle}>
        <h3 style={titleStyle}>用例选择</h3>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          {ALL_CASES.map((c) => (
            <label key={c.key} style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 13 }}>
              <input type="checkbox" checked={form.cases.includes(c.key)} onChange={() => toggleCase(c.key)} />
              <span>
                <code style={codeStyle}>{c.key}</code> {c.label}
              </span>
            </label>
          ))}
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 12, marginTop: 16 }}>
          <label style={{ display: 'flex', flexDirection: 'column', fontSize: 13 }}>
            <span style={{ color: '#6b7280', marginBottom: 4 }}>出价者 IDs（逗号分隔）</span>
            <input
              type="text"
              value={form.bidder_ids}
              onChange={(e) => setForm((s) => ({ ...s, bidder_ids: e.target.value }))}
              style={inputStyle}
            />
          </label>
        </div>

        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <button type="button" disabled={running || form.cases.length === 0} onClick={handleStart} style={primaryBtn(running)}>
            {running ? '启动中...' : '启动防狙击场景'}
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
          test_id: <code>{testID || '-'}</code> · WS: {connected ? '已连接' : '未连接'} · 当前用例: {step || '-'}
        </div>
        <ProgressBar value={progress} label={`${progress}%`} />
        {lastWS && (
          <div style={{ marginTop: 8, fontSize: 12, color: '#6b7280' }}>
            最近事件 metrics: <code>{JSON.stringify(lastWS.metrics)}</code>
          </div>
        )}
      </section>

      {report?.cases && report.cases.length > 0 && (
        <section style={cardStyle}>
          <h3 style={titleStyle}>用例结果</h3>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
            {report.cases.map((c) => (
              <button
                key={c.name}
                type="button"
                onClick={() => setActiveCase(c.name)}
                style={caseChipStyle(c.ok, activeCase === c.name)}
              >
                {c.ok ? '✓ ' : '✗ '}
                {c.name}
              </button>
            ))}
            <span style={{ marginLeft: 'auto', fontSize: 13, color: report.all_ok ? '#10b981' : '#ef4444' }}>
              {report.all_ok ? '全部通过' : '存在失败'}
            </span>
          </div>

          {activeReport && (
            <div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 12, marginBottom: 16 }}>
                <Metric label="结果" value={activeReport.ok ? 'OK' : 'FAIL'} ok={activeReport.ok} />
                <Metric label="拍卖 ID" value={String(activeReport.report?.auction_id ?? '-')} />
                <Metric label="出价数" value={String(activeReport.report?.bid_count ?? '-')} />
                <Metric label="触发延时数" value={String(activeReport.report?.triggered_count ?? '-')} />
                <Metric
                  label="累计延时"
                  value={`${((activeReport.report?.delay_used_ms ?? 0) / 1000).toFixed(1)}s`}
                />
              </div>

              <AntiSnipeTimeline
                originalEndTime={activeReport.report?.original_end_time}
                actualEndTime={activeReport.report?.actual_end_time}
                events={activeReport.report?.timeline ?? []}
              />

              {activeReport.message && (
                <div style={{ marginTop: 12, color: activeReport.ok ? '#6b7280' : '#ef4444', fontSize: 13 }}>
                  备注：{activeReport.message}
                </div>
              )}
            </div>
          )}
        </section>
      )}

      {report?.error && (
        <section style={cardStyle}>
          <div style={{ color: '#ef4444', fontSize: 13 }}>{report.error}</div>
        </section>
      )}
    </div>
  );
}

function pollReport(testID: string, setReport: (r: ScenarioReport) => void) {
  let n = 0;
  const max = 120;
  const tick = async () => {
    n += 1;
    try {
      const t = await getReport(testID);
      if (t.Status === 'completed' || t.Status === 'failed' || t.Status === 'cancelled') {
        try {
          const r = JSON.parse(t.ResultJSON || '{}') as ScenarioReport;
          setReport(r);
        } catch {
          setReport({ error: t.ErrorMsg || 'parse error' });
        }
        return;
      }
    } catch {
      /* ignore */
    }
    if (n < max) setTimeout(tick, 1000);
  };
  setTimeout(tick, 1000);
}

function Metric({ label, value, ok }: { label: string; value: string; ok?: boolean }) {
  return (
    <div style={{ background: '#f8fafc', border: '1px solid #e5e7eb', borderRadius: 6, padding: '10px 12px' }}>
      <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 4 }}>{label}</div>
      <div
        style={{
          fontSize: 18,
          fontFamily: 'monospace',
          fontWeight: 600,
          color: ok === undefined ? '#1f2937' : ok ? '#10b981' : '#ef4444',
        }}
      >
        {value}
      </div>
    </div>
  );
}

const cardStyle: React.CSSProperties = {
  padding: 16,
  border: '1px solid #e5e7eb',
  borderRadius: 8,
  marginBottom: 16,
};
const titleStyle: React.CSSProperties = { fontSize: 16, marginBottom: 12 };
const inputStyle: React.CSSProperties = {
  padding: '6px 10px',
  border: '1px solid #d1d5db',
  borderRadius: 6,
  fontSize: 14,
};
const codeStyle: React.CSSProperties = {
  background: '#f1f5f9',
  padding: '1px 6px',
  borderRadius: 3,
  fontSize: 12,
};
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
const caseChipStyle = (ok: boolean, active: boolean): React.CSSProperties => ({
  padding: '6px 12px',
  borderRadius: 16,
  border: `1px solid ${ok ? '#10b981' : '#ef4444'}`,
  background: active ? (ok ? '#10b981' : '#ef4444') : '#fff',
  color: active ? '#fff' : ok ? '#10b981' : '#ef4444',
  fontSize: 13,
  cursor: 'pointer',
});
