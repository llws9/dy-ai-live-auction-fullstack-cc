import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  cancelTest,
  discoverWS,
  startUserJourney,
  type UserJourneyConfig,
  type UserJourneyReport,
} from '@/api/test';
import { usePollReport } from '@/hooks/usePollReport';
import ProgressBar from '@/components/ProgressBar';
import StepTimeline, { StepEvent } from '@/components/StepTimeline';
import { useWSStore } from '@/store/wsStore';
import { Metric } from '@/components/ui/Metric';
import { NumField } from '@/components/ui/Field';
import { cardStyle, primaryBtn, secondaryBtn, titleStyle } from '@/components/ui/styles';

const defaultConfig: Required<UserJourneyConfig> = {
  include_reminder: true,
  include_sky_lamp: true,
  include_fixed_price: true,
  auction_duration_sec: 30,
  buyer_count: 1,
  keep_evidence: true,
};

export default function UserJourney() {
  const [form, setForm] = useState<Required<UserJourneyConfig>>(defaultConfig);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<UserJourneyReport | null>(null);
  const { connected, testID, progress, step, history, connect, disconnect } = useWSStore();
  const pollReport = usePollReport<UserJourneyReport>({ maxAttempts: 120 });

  useEffect(() => () => disconnect(), [disconnect]);

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

  const reportEvents: StepEvent[] = useMemo(
    () =>
      (report?.steps ?? []).map((s) => ({
        step: s.step,
        ok: s.ok,
        duration_ms: s.duration_ms,
        message: s.message,
        ref_id: s.ref_id,
      })),
    [report],
  );

  const timelineEvents = events.length > 0 ? events : reportEvents;

  const setField = <K extends keyof UserJourneyConfig>(k: K, v: Required<UserJourneyConfig>[K]) =>
    setForm((s) => ({ ...s, [k]: v }));

  const handleStart = async () => {
    setError(null);
    setReport(null);
    setRunning(true);
    try {
      const id = await startUserJourney(form);
      const wsURL = await discoverWS(id);
      connect(wsURL, id);
      pollReport.start(id, setReport, setError);
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
      pollReport.cancel();
    }
  };

  return (
    <div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>用户验收剧本</h1>

      <section style={cardStyle}>
        <h3 style={titleStyle}>一键验收配置</h3>
        <div style={{ color: '#6b7280', fontSize: 13, marginBottom: 12 }}>
          自动造数并以买家视角执行：进直播间、关注、出价、点天灯、一口价购买和最终校验。
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 12 }}>
          <CheckField
            label="验证关注提醒"
            checked={form.include_reminder}
            onChange={(v) => setField('include_reminder', v)}
          />
          <CheckField
            label="验证点天灯"
            checked={form.include_sky_lamp}
            onChange={(v) => setField('include_sky_lamp', v)}
          />
          <CheckField
            label="验证一口价"
            checked={form.include_fixed_price}
            onChange={(v) => setField('include_fixed_price', v)}
          />
          <CheckField
            label="保留验收证据"
            checked={form.keep_evidence}
            onChange={(v) => setField('keep_evidence', v)}
          />
          <NumField
            label="竞拍时长(秒)"
            value={form.auction_duration_sec}
            min={5}
            onChange={(v) => setField('auction_duration_sec', v)}
          />
          <NumField
            label="买家数(P0=1)"
            value={form.buyer_count}
            min={1}
            onChange={(v) => setField('buyer_count', v)}
          />
        </div>
        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <button type="button" disabled={running} onClick={handleStart} style={primaryBtn(running)}>
            {running ? '启动中...' : '启动用户验收'}
          </button>
          <button type="button" disabled={!testID} onClick={handleCancel} style={secondaryBtn(!testID)}>
            取消
          </button>
          {testID && (
            <Link to={`/test/report/${testID}`} style={{ alignSelf: 'center', fontSize: 13 }}>
              查看报告
            </Link>
          )}
        </div>
        {error && <div style={{ color: '#ef4444', marginTop: 12, fontSize: 13 }}>错误：{error}</div>}
      </section>

      <section style={cardStyle}>
        <h3 style={titleStyle}>实时进度</h3>
        <div style={{ marginBottom: 8, fontSize: 13, color: '#6b7280' }}>
          test_id: <code>{testID || report?.test_run_id || '-'}</code> · WS: {connected ? '已连接' : '未连接'} · 当前步骤:{' '}
          {step || '-'}
        </div>
        <ProgressBar value={progress} label={`${progress}%`} />
      </section>

      <section style={cardStyle}>
        <h3 style={titleStyle}>步骤时间线</h3>
        <StepTimeline events={timelineEvents} />
      </section>

      {report && (
        <section style={cardStyle}>
          <h3 style={titleStyle}>证据报告</h3>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 12 }}>
            <Metric label="整体成功" value={report.all_ok ? 'OK' : 'FAIL'} ok={report.all_ok} />
            <Metric label="直播间 ID" value={fmt(report.live_stream_id)} />
            <Metric label="竞拍 ID" value={fmt(report.auction_id)} />
            <Metric label="一口价商品" value={fmt(report.fixed_price_item_id)} />
            <Metric label="订单 ID" value={fmt(report.order_id)} />
            <Metric label="余额变化" value={`${report.balance_before ?? '-'} → ${report.balance_after ?? '-'}`} />
            <Metric label="库存变化" value={`${report.stock_before ?? '-'} → ${report.stock_after ?? '-'}`} />
          </div>
          {report.warnings && report.warnings.length > 0 && (
            <div style={{ marginTop: 12, color: '#92400e', fontSize: 13 }}>
              warning: {report.warnings.join('；')}
            </div>
          )}
          {report.error && <div style={{ marginTop: 12, color: '#ef4444', fontSize: 13 }}>{report.error}</div>}
        </section>
      )}
    </div>
  );
}

function CheckField({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: '#374151' }}>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      {label}
    </label>
  );
}

function fmt(v?: number): string {
  return v == null || Number.isNaN(v) ? '-' : String(v);
}
