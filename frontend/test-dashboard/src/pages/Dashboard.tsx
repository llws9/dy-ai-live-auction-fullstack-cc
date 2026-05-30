import { useState } from 'react';
import { startDummy, discoverWS, cancelTest } from '@/api/test';
import { useWSStore } from '@/store/wsStore';
import { useTestStore } from '@/store/testStore';
import ProgressBar from '@/components/ProgressBar';

export default function Dashboard() {
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { connected, testID, progress, step, metrics, history, connect, disconnect } = useWSStore();
  const setCurrent = useTestStore((s) => s.setCurrent);

  const handleStart = async () => {
    setError(null);
    setRunning(true);
    try {
      const id = await startDummy({});
      setCurrent({
        ID: id,
        TestType: 'dummy',
        Status: 'running',
        ConfigJSON: '',
        ResultJSON: '',
        ReplayToken: '',
        ScriptName: '',
        ErrorMsg: '',
        CreatedAt: new Date().toISOString(),
      });
      const wsURL = await discoverWS(id);
      connect(wsURL, id);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
    } finally {
      setRunning(false);
    }
  };

  const handleCancel = async () => {
    if (!testID) return;
    try {
      await cancelTest(testID);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
    } finally {
      disconnect();
    }
  };

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>控制台</h1>

      <section
        style={{
          padding: 16,
          border: '1px solid #e5e7eb',
          borderRadius: 8,
          marginBottom: 16,
        }}
      >
        <h3 style={{ fontSize: 16, marginBottom: 12 }}>Dummy 进度场景（M1 联调）</h3>
        <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
          <button
            type="button"
            disabled={running}
            onClick={handleStart}
            style={{
              padding: '8px 16px',
              background: 'var(--color-primary, #3b82f6)',
              color: '#fff',
              border: 'none',
              borderRadius: 6,
              cursor: running ? 'not-allowed' : 'pointer',
              opacity: running ? 0.6 : 1,
            }}
          >
            {running ? '启动中...' : '启动 Dummy 测试'}
          </button>
          <button
            type="button"
            disabled={!testID}
            onClick={handleCancel}
            style={{
              padding: '8px 16px',
              background: '#fff',
              color: '#1f2937',
              border: '1px solid #d1d5db',
              borderRadius: 6,
              cursor: testID ? 'pointer' : 'not-allowed',
              opacity: testID ? 1 : 0.6,
            }}
          >
            取消
          </button>
        </div>

        {error && (
          <div style={{ color: '#ef4444', marginBottom: 12, fontSize: 13 }}>错误：{error}</div>
        )}

        <div style={{ marginBottom: 8, fontSize: 13, color: '#6b7280' }}>
          test_id: <code>{testID || '-'}</code> · WS: {connected ? '已连接' : '未连接'}
        </div>

        <ProgressBar value={progress} label={step || 'idle'} />
      </section>

      <section
        style={{
          padding: 16,
          border: '1px solid #e5e7eb',
          borderRadius: 8,
          marginBottom: 16,
        }}
      >
        <h3 style={{ fontSize: 16, marginBottom: 12 }}>实时指标</h3>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))',
            gap: 12,
          }}
        >
          <Metric label="QPS" value={fmtNum(metrics.qps)} />
          <Metric label="P99 (ms)" value={fmtNum(metrics.p99_ms)} />
          <Metric
            label="错误率"
            value={metrics.error_rate != null ? `${(Number(metrics.error_rate) * 100).toFixed(2)}%` : '-'}
          />
          <Metric label="累计出价" value={fmtNum(metrics.bids_total)} />
          <Metric label="累计错误" value={fmtNum(metrics.errors_total)} />
          <Metric label="耗时 (ms)" value={fmtNum(metrics.elapsed_ms)} />
        </div>
      </section>

      <section
        style={{
          padding: 16,
          border: '1px solid #e5e7eb',
          borderRadius: 8,
        }}
      >
        <h3 style={{ fontSize: 16, marginBottom: 12 }}>实时消息（{history.length}）</h3>
        <pre
          style={{
            maxHeight: 240,
            overflow: 'auto',
            background: '#0f172a',
            color: '#e2e8f0',
            padding: 12,
            borderRadius: 6,
            fontSize: 12,
            margin: 0,
          }}
        >
          {history.map((m, i) => `[${i}] ${m.step}  ${m.progress}%`).join('\n') ||
            '(暂无)'}
        </pre>
      </section>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div
      style={{
        background: '#f8fafc',
        border: '1px solid #e5e7eb',
        borderRadius: 6,
        padding: '10px 12px',
      }}
    >
      <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 4 }}>{label}</div>
      <div style={{ fontSize: 18, fontFamily: 'monospace', fontWeight: 600 }}>{value}</div>
    </div>
  );
}

function fmtNum(v: unknown): string {
  if (v == null) return '-';
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  return n.toLocaleString();
}
