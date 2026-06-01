import { useEffect, useMemo, useState } from 'react';
import { startAntiSnipe, discoverWS, cancelTest } from '@/api/test';
import { useWSStore } from '@/store/wsStore';
import { usePollReport } from '@/hooks/usePollReport';
import ProgressBar from '@/components/ProgressBar';
import AntiSnipeTimeline, { AntiSnipeTimelineEvent } from '@/components/AntiSnipeTimeline';
import { Metric } from '@/components/ui/Metric';
import { cardStyle, titleStyle, inputStyle, codeStyle, primaryBtn, secondaryBtn, caseChipStyle } from '@/components/ui/styles';

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
  const poll = usePollReport<ScenarioReport>();

  // 卸载时清理 WS 与全局 store
  useEffect(() => () => disconnect(), [disconnect]);

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
      poll.start(id, (r) => {
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
