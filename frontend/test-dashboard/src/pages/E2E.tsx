import { useEffect, useMemo, useState } from 'react';
import { startE2E, discoverWS, cancelTest } from '@/api/test';
import { useWSStore } from '@/store/wsStore';
import { usePollReport } from '@/hooks/usePollReport';
import ProgressBar from '@/components/ProgressBar';
import StepTimeline, { StepEvent } from '@/components/StepTimeline';
import { Metric } from '@/components/ui/Metric';
import { NumField, TextField } from '@/components/ui/Field';
import { cardStyle, titleStyle, primaryBtn, secondaryBtn } from '@/components/ui/styles';

interface E2EForm {
  seller_id: number;
  bidder_ids: string; // 逗号分隔
  subscriber_id: number;
  start_price: number;
  increment: number;
  duration: number;
}

interface E2EReport {
  test_id?: string;
  auction_id?: number;
  product_id?: number;
  winner_id?: number;
  order_id?: number;
  steps?: StepEvent[];
  all_ok?: boolean;
  error?: string;
}

const defaultForm: E2EForm = {
  seller_id: 9001,
  bidder_ids: '2001',
  subscriber_id: 0,
  start_price: 100,
  increment: 10,
  duration: 30,
};

export default function E2E() {
  const [form, setForm] = useState<E2EForm>(defaultForm);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<E2EReport | null>(null);
  const { connected, testID, progress, step, history, connect, disconnect } = useWSStore();
  const pollReport = usePollReport<E2EReport>({ maxAttempts: 60 });

  // 卸载时清理 WS 与全局 store
  useEffect(() => () => disconnect(), [disconnect]);

  // 把 WS 历史转成 StepEvent[]
  const events: StepEvent[] = useMemo(
    () =>
      history.map((m) => ({
        step: m.step,
        ok: m.metrics?.ok as boolean | undefined,
        duration_ms: m.metrics?.duration_ms as number | undefined,
        message: m.metrics?.message as string | undefined,
        ref_id: m.metrics?.ref_id as number | undefined,
        ts: m.ts,
      })),
    [history],
  );

  const setField = <K extends keyof E2EForm>(k: K, v: E2EForm[K]) =>
    setForm((s) => ({ ...s, [k]: v }));

  const handleStart = async () => {
    setError(null);
    setReport(null);
    setRunning(true);
    try {
      const bidders = form.bidder_ids
        .split(',')
        .map((s) => Number(s.trim()))
        .filter((n) => !Number.isNaN(n) && n > 0);
      const id = await startE2E({
        seller_id: form.seller_id,
        bidder_ids: bidders,
        subscriber_id: form.subscriber_id,
        start_price: form.start_price,
        increment: form.increment,
        duration: form.duration,
      });
      const wsURL = await discoverWS(id);
      connect(wsURL, id);
      pollReport.start(id, setReport);
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

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>E2E 全链路（场景 E）</h1>

      <section style={cardStyle}>
        <h3 style={titleStyle}>参数</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12 }}>
          <NumField label="卖家 ID" value={form.seller_id} min={1}
            onChange={(v) => setField('seller_id', v)} />
          <TextField label="出价者 IDs（逗号分隔）" value={form.bidder_ids}
            onChange={(v) => setField('bidder_ids', v)} />
          <NumField label="订阅者 ID" value={form.subscriber_id} min={1}
            onChange={(v) => setField('subscriber_id', v)} />
          <NumField label="起拍价" value={form.start_price} min={0}
            onChange={(v) => setField('start_price', v)} />
          <NumField label="加价幅度" value={form.increment} min={1}
            onChange={(v) => setField('increment', v)} />
          <NumField label="持续时间(秒)" value={form.duration} min={1}
            onChange={(v) => setField('duration', v)} />
        </div>
        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <button type="button" disabled={running} onClick={handleStart} style={primaryBtn(running)}>
            {running ? '启动中...' : '启动 E2E'}
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
          test_id: <code>{testID || '-'}</code> · WS: {connected ? '已连接' : '未连接'} · 当前步骤: {step || '-'}
        </div>
        <ProgressBar value={progress} label={`${progress}%`} />
      </section>

      <section style={cardStyle}>
        <h3 style={titleStyle}>步骤时间轴</h3>
        <StepTimeline events={events} />
      </section>

      {report && (
        <section style={cardStyle}>
          <h3 style={titleStyle}>最终报告</h3>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 12 }}>
            <Metric label="整体成功" value={report.all_ok ? 'OK' : 'FAIL'} ok={report.all_ok} />
            <Metric label="商品 ID" value={String(report.product_id ?? '-')} />
            <Metric label="拍卖 ID" value={String(report.auction_id ?? '-')} />
            <Metric label="中标用户" value={String(report.winner_id ?? '-')} />
            <Metric label="订单 ID" value={String(report.order_id ?? '-')} />
          </div>
          {report.error && (
            <div style={{ marginTop: 12, color: '#ef4444', fontSize: 13 }}>{report.error}</div>
          )}
        </section>
      )}
    </div>
  );
}
