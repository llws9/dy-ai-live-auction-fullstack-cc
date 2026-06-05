import { useEffect, useMemo, useState, type CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import { cancelTest, discoverWS, startUserJourney, type UserJourneyReport } from '@/api/test';
import { usePollReport } from '@/hooks/usePollReport';
import { useWSStore } from '@/store/wsStore';
import { buildDemoTheaterModel, DEMO_USER_JOURNEY_CONFIG, type DemoEvent } from './demoTheater';

export default function Screen() {
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<UserJourneyReport | null>(null);
  const { connected, testID, progress, step, history, connect, disconnect } = useWSStore();
  const pollReport = usePollReport<UserJourneyReport>({ maxAttempts: 120 });

  useEffect(() => {
    return () => {
      disconnect();
      pollReport.cancel();
    };
  }, [disconnect, pollReport]);

  const model = useMemo(
    () =>
      buildDemoTheaterModel({
        connected,
        testID,
        progress,
        step,
        history,
        report,
        error,
        starting,
      }),
    [connected, error, history, progress, report, starting, step, testID],
  );

  const handleStart = async () => {
    setStarting(true);
    setError(null);
    setReport(null);
    try {
      const id = await startUserJourney(DEMO_USER_JOURNEY_CONFIG);
      const wsURL = await discoverWS(id);
      connect(wsURL, id);
      pollReport.start(
        id,
        (nextReport) => {
          setReport(nextReport);
          setStarting(false);
        },
        (message) => {
          setError(message);
          setStarting(false);
        },
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setStarting(false);
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
      setStarting(false);
    }
  };

  return (
    <div style={page}>
      <header style={header}>
        <div>
          <div style={eyebrow}>UserJourney Demo Theater</div>
          <h1 style={heroTitle}>{model.heroTitle}</h1>
          <p style={heroSubtitle}>一键跑通进房、出价、点天灯、成交、库存扣减和最终证据报告。</p>
        </div>
        <div style={headerActions}>
          <span style={badge(model.liveBadge)}>{model.liveBadge}</span>
          <Link to="/test" style={exitBtn}>
            返回控制台
          </Link>
        </div>
      </header>

      <section style={heroGrid}>
        <div style={stageCard}>
          <div style={stageHeader}>
            <span style={sectionKicker}>本场挑战</span>
            <button type="button" onClick={handleStart} disabled={starting} style={primaryBtn(starting)}>
              {starting ? '启动中...' : model.primaryActionLabel}
            </button>
          </div>
          <div style={challengeText}>让 AI 直播竞拍在一个标准剧本内完成业务闭环，并把关键事实投到评委大屏。</div>
          <div style={roleGrid}>
            <Role title="参演角色" value="商家开播" detail="创建直播间、商品和竞拍规则" />
            <Role
              title="买家出价"
              value={model.stage === 'idle' ? '等待买家' : '竞价中'}
              detail={`当前领先：${model.leaderLabel}`}
            />
            <Role title="点天灯" value={model.highlightedEvent === 'sky_lamp' ? '已触发' : '待触发'} detail="高权重竞价反馈进入演示主线" />
          </div>
          <div style={actionRow}>
            <button type="button" onClick={handleCancel} disabled={!testID} style={ghostBtn(!testID)}>
              停止演示
            </button>
            {model.reportPath && (
              <Link to={model.reportPath} style={reportLink}>
                查看技术报告
              </Link>
            )}
          </div>
          {model.failureMessage && (
            <div style={errorBox}>
              {model.failureTitle}: {model.failureMessage}
            </div>
          )}
        </div>

        <div style={priceCard}>
          <div style={sectionKicker}>当前最高价</div>
          <div style={priceText}>{model.currentPrice}</div>
          <div style={leaderText}>{model.leaderLabel}</div>
          <div style={metricGrid}>
            <Metric label="出价事件" value={String(model.bidCount)} />
            <Metric label="订单数" value={String(model.orderCount)} />
            <Metric label="库存变化" value={model.stockLabel} />
            <Metric label="进度" value={model.progressLabel} />
          </div>
          <div style={progressTrack}>
            <div style={{ ...progressFill, width: model.progressLabel }} />
          </div>
          <div style={technicalLine}>{model.technicalLine}</div>
        </div>
      </section>

      <section style={contentGrid}>
        <div style={panel}>
          <div style={panelTitle}>直播事件</div>
          <div style={eventList}>
            {model.events.length > 0 ? (
              model.events.map((event, index) => <EventCard key={`${event.step}-${index}`} event={event} />)
            ) : (
              <div style={emptyState}>等待一键启动后接收 UserJourney 进度流。</div>
            )}
          </div>
        </div>

        <div style={panel}>
          <div style={panelTitle}>验收结论</div>
          <div style={conclusionList}>
            {model.conclusions.map((item) => (
              <div key={item.title} style={conclusionRow}>
                <span style={conclusionDot(item.status)} />
                <div>
                  <div style={conclusionTitle}>{item.title}</div>
                  <div style={conclusionDesc}>{item.description}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}

function Role({ title, value, detail }: { title: string; value: string; detail: string }) {
  return (
    <div style={roleCard}>
      <div style={roleTitle}>{title}</div>
      <div style={roleValue}>{value}</div>
      <div style={roleDetail}>{detail}</div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div style={metricBox}>
      <div style={metricLabel}>{label}</div>
      <div style={metricValue}>{value}</div>
    </div>
  );
}

function EventCard({ event }: { event: DemoEvent }) {
  return (
    <div style={{ ...eventCard, borderColor: eventTone[event.tone] }}>
      <div style={{ ...eventBullet, background: eventTone[event.tone] }} />
      <div>
        <div style={eventTitle}>{event.title}</div>
        <div style={eventDesc}>{event.description}</div>
      </div>
    </div>
  );
}

const eventTone: Record<DemoEvent['tone'], string> = {
  neutral: '#94a3b8',
  blue: '#38bdf8',
  orange: '#f59e0b',
  green: '#22c55e',
  red: '#ef4444',
};

const page: CSSProperties = {
  minHeight: '100vh',
  width: '100vw',
  background:
    'radial-gradient(circle at 12% 8%, rgba(56,189,248,0.25), transparent 28%), radial-gradient(circle at 88% 20%, rgba(245,158,11,0.18), transparent 26%), #07111f',
  color: '#e2e8f0',
  padding: 32,
  display: 'flex',
  flexDirection: 'column',
  gap: 24,
  boxSizing: 'border-box',
};
const header: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 24,
};
const eyebrow: CSSProperties = {
  color: '#38bdf8',
  fontSize: 13,
  letterSpacing: 3,
  textTransform: 'uppercase',
  fontWeight: 800,
};
const heroTitle: CSSProperties = {
  margin: '6px 0',
  color: '#f8fafc',
  fontSize: 48,
  lineHeight: 1.05,
};
const heroSubtitle: CSSProperties = {
  margin: 0,
  color: '#94a3b8',
  fontSize: 16,
};
const headerActions: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 12,
};
const exitBtn: CSSProperties = {
  padding: '10px 16px',
  border: '1px solid rgba(148,163,184,0.35)',
  borderRadius: 999,
  color: '#cbd5e1',
  textDecoration: 'none',
  fontSize: 14,
  background: 'rgba(15,23,42,0.72)',
};
const heroGrid: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: '1.25fr 0.75fr',
  gap: 24,
};
const stageCard: CSSProperties = {
  background: 'linear-gradient(135deg, rgba(15,23,42,0.92), rgba(14,116,144,0.18))',
  border: '1px solid rgba(125,211,252,0.28)',
  borderRadius: 28,
  padding: 28,
  boxShadow: '0 28px 80px rgba(0,0,0,0.32)',
};
const priceCard: CSSProperties = {
  background: 'linear-gradient(180deg, rgba(251,191,36,0.16), rgba(15,23,42,0.9))',
  border: '1px solid rgba(251,191,36,0.28)',
  borderRadius: 28,
  padding: 28,
};
const stageHeader: CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  gap: 16,
};
const sectionKicker: CSSProperties = {
  color: '#93c5fd',
  fontSize: 13,
  fontWeight: 800,
  letterSpacing: 2,
};
const challengeText: CSSProperties = {
  marginTop: 20,
  fontSize: 28,
  lineHeight: 1.25,
  color: '#f8fafc',
  maxWidth: 760,
};
const roleGrid: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
  gap: 14,
  marginTop: 24,
};
const roleCard: CSSProperties = {
  background: 'rgba(15,23,42,0.68)',
  border: '1px solid rgba(148,163,184,0.18)',
  borderRadius: 18,
  padding: 18,
};
const roleTitle: CSSProperties = {
  color: '#94a3b8',
  fontSize: 13,
  marginBottom: 8,
};
const roleValue: CSSProperties = {
  color: '#f8fafc',
  fontSize: 22,
  fontWeight: 800,
};
const roleDetail: CSSProperties = {
  color: '#94a3b8',
  fontSize: 12,
  lineHeight: 1.6,
  marginTop: 8,
};
const actionRow: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 12,
  marginTop: 22,
};
const primaryBtn = (disabled: boolean): CSSProperties => ({
  border: 0,
  borderRadius: 999,
  padding: '13px 22px',
  color: '#03111f',
  background: disabled ? '#64748b' : 'linear-gradient(135deg, #facc15, #fb923c)',
  fontWeight: 900,
  cursor: disabled ? 'not-allowed' : 'pointer',
});
const ghostBtn = (disabled: boolean): CSSProperties => ({
  border: '1px solid rgba(148,163,184,0.28)',
  borderRadius: 999,
  padding: '11px 18px',
  color: disabled ? '#64748b' : '#cbd5e1',
  background: 'rgba(15,23,42,0.55)',
  cursor: disabled ? 'not-allowed' : 'pointer',
});
const reportLink: CSSProperties = {
  color: '#67e8f9',
  textDecoration: 'none',
  fontWeight: 800,
};
const errorBox: CSSProperties = {
  marginTop: 16,
  padding: 12,
  border: '1px solid rgba(248,113,113,0.35)',
  borderRadius: 14,
  color: '#fecaca',
  background: 'rgba(127,29,29,0.32)',
};
const priceText: CSSProperties = {
  marginTop: 18,
  fontSize: 62,
  fontWeight: 900,
  color: '#fef3c7',
  letterSpacing: -2,
};
const leaderText: CSSProperties = {
  color: '#fde68a',
  fontSize: 20,
  fontWeight: 800,
};
const metricGrid: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
  gap: 12,
  marginTop: 24,
};
const metricBox: CSSProperties = {
  background: 'rgba(15,23,42,0.55)',
  border: '1px solid rgba(251,191,36,0.16)',
  borderRadius: 16,
  padding: 14,
};
const metricLabel: CSSProperties = {
  color: '#94a3b8',
  fontSize: 12,
};
const metricValue: CSSProperties = {
  color: '#f8fafc',
  fontSize: 22,
  fontWeight: 900,
  marginTop: 6,
};
const progressTrack: CSSProperties = {
  height: 8,
  borderRadius: 999,
  background: 'rgba(148,163,184,0.2)',
  overflow: 'hidden',
  marginTop: 22,
};
const progressFill: CSSProperties = {
  height: '100%',
  borderRadius: 999,
  background: 'linear-gradient(90deg, #38bdf8, #facc15)',
};
const technicalLine: CSSProperties = {
  marginTop: 14,
  color: '#94a3b8',
  fontSize: 12,
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
};
const contentGrid: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: '1fr 1fr',
  gap: 24,
};
const panel: CSSProperties = {
  background: 'rgba(15,23,42,0.78)',
  border: '1px solid rgba(148,163,184,0.18)',
  borderRadius: 24,
  padding: 24,
};
const panelTitle: CSSProperties = {
  color: '#f8fafc',
  fontSize: 22,
  fontWeight: 900,
  marginBottom: 16,
};
const eventList: CSSProperties = {
  display: 'grid',
  gap: 12,
};
const eventCard: CSSProperties = {
  display: 'flex',
  gap: 12,
  alignItems: 'flex-start',
  padding: 14,
  border: '1px solid',
  borderRadius: 16,
  background: 'rgba(2,6,23,0.42)',
};
const eventBullet: CSSProperties = {
  width: 10,
  height: 10,
  borderRadius: 999,
  marginTop: 6,
  flex: '0 0 auto',
};
const eventTitle: CSSProperties = {
  color: '#f8fafc',
  fontWeight: 900,
};
const eventDesc: CSSProperties = {
  color: '#94a3b8',
  fontSize: 13,
  lineHeight: 1.6,
  marginTop: 4,
};
const emptyState: CSSProperties = {
  color: '#94a3b8',
  border: '1px dashed rgba(148,163,184,0.28)',
  borderRadius: 16,
  padding: 24,
};
const conclusionList: CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 16,
};
const conclusionRow: CSSProperties = {
  display: 'flex',
  gap: 14,
  alignItems: 'flex-start',
};
const conclusionDot = (status: string): CSSProperties => ({
  width: 14,
  height: 14,
  borderRadius: 999,
  marginTop: 4,
  background: status === 'passed' ? '#22c55e' : status === 'failed' ? '#ef4444' : '#64748b',
});
const conclusionTitle: CSSProperties = {
  color: '#f8fafc',
  fontSize: 18,
  fontWeight: 900,
};
const conclusionDesc: CSSProperties = {
  color: '#94a3b8',
  marginTop: 4,
  lineHeight: 1.5,
};
const badge = (value: string): CSSProperties => ({
  borderRadius: 999,
  padding: '9px 14px',
  color: value === 'FAILED' ? '#fecaca' : value === 'DONE' ? '#bbf7d0' : '#cffafe',
  border: '1px solid rgba(148,163,184,0.28)',
  background: 'rgba(15,23,42,0.72)',
  fontWeight: 900,
  letterSpacing: 1,
});
