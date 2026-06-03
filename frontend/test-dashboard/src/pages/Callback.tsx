import { useEffect, useState } from 'react';
import { startCallback, discoverWS, cancelTest } from '@/api/test';
import { useWSStore } from '@/store/wsStore';
import { usePollReport } from '@/hooks/usePollReport';
import ProgressBar from '@/components/ProgressBar';
import StateMachineTrace, { TraceEntry } from '@/components/StateMachineTrace';
import { Metric } from '@/components/ui/Metric';
import { NumField, TextField } from '@/components/ui/Field';
import { cardStyle, titleStyle, codeStyle, primaryBtn, secondaryBtn, caseChipStyle } from '@/components/ui/styles';

const ALL_CASES = [
  { key: 'normal', label: '正常投递' },
  { key: 'timeout', label: '超时 + Probe 命中' },
  { key: 'duplicate', label: '幂等去重' },
  { key: 'tampered', label: '签名篡改拒绝' },
  { key: 'dlq', label: '重试上限进 DLQ' },
  { key: 'out_of_order', label: '乱序到达保留首条' },
];

interface CallbackForm {
  partner_url: string;
  hmac_secret: string;
  cases: string[];
  max_retry: number;
  timeout_ms: number;
}

interface CaseReport {
  name: string;
  ok: boolean;
  message?: string;
  idempotency_key: string;
  trace?: TraceEntry[];
  http_calls?: number;
  dlq_entered?: boolean;
  idempotent_blocked?: number;
}

interface ScenarioReport {
  cases?: CaseReport[];
  all_ok?: boolean;
  error?: string;
}

const defaultForm: CallbackForm = {
  partner_url: 'http://localhost:18091',
  hmac_secret: 'test-secret-key',
  cases: ALL_CASES.map((c) => c.key),
  max_retry: 3,
  timeout_ms: 1000,
};

export default function Callback() {
  const [form, setForm] = useState<CallbackForm>(defaultForm);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<ScenarioReport | null>(null);
  const [activeCase, setActiveCase] = useState<string | null>(null);
  const { connected, testID, progress, step, connect, disconnect } = useWSStore();
  const poll = usePollReport<ScenarioReport>();

  // 卸载时清理 WS 与全局 store
  useEffect(() => () => disconnect(), [disconnect]);

  const handleStart = async () => {
    setError(null);
    setReport(null);
    setActiveCase(null);
    setRunning(true);
    try {
      const id = await startCallback({
        partner_url: form.partner_url,
        hmac_secret: form.hmac_secret,
        cases: form.cases,
        max_retry: form.max_retry,
        timeout_ms: form.timeout_ms,
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
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>回调可靠投递（场景 H）</h1>

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

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
            gap: 12,
            marginTop: 16,
          }}
        >
          <TextField label="Partner URL" value={form.partner_url} onChange={(v) => setForm((s) => ({ ...s, partner_url: v }))} />
          <TextField label="HMAC Secret" value={form.hmac_secret} onChange={(v) => setForm((s) => ({ ...s, hmac_secret: v }))} type="password" />
          <NumField label="重试上限" value={form.max_retry} min={1} onChange={(v) => setForm((s) => ({ ...s, max_retry: v }))} />
          <NumField label="单次超时(ms)" value={form.timeout_ms} min={100} onChange={(v) => setForm((s) => ({ ...s, timeout_ms: v }))} />
        </div>

        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <button type="button" disabled={running || form.cases.length === 0} onClick={handleStart} style={primaryBtn(running)}>
            {running ? '启动中...' : '启动回调测试'}
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
                <Metric label="HTTP 调用次数" value={String(activeReport.http_calls ?? 0)} />
                <Metric label="进入 DLQ" value={activeReport.dlq_entered ? 'YES' : 'NO'} ok={!activeReport.dlq_entered} />
                <Metric label="幂等阻挡" value={String(activeReport.idempotent_blocked ?? 0)} />
              </div>

              <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 12 }}>
                idem key: <code>{activeReport.idempotency_key}</code>
              </div>

              <StateMachineTrace trace={activeReport.trace ?? []} />

              {activeReport.message && (
                <div style={{ marginTop: 12, color: activeReport.ok ? '#6b7280' : '#ef4444', fontSize: 13 }}>
                  备注：{activeReport.message}
                </div>
              )}
            </div>
          )}
        </section>
      )}
    </div>
  );
}
