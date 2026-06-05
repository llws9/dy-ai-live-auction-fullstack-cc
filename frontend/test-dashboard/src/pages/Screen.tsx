import { useEffect, useMemo, useState, type CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import { cancelTest, discoverWS, startUserJourney, type UserJourneyReport } from '@/api/test';
import { usePollReport } from '@/hooks/usePollReport';
import { useWSStore } from '@/store/wsStore';
import { buildDemoTheaterModel, DEMO_USER_JOURNEY_CONFIG, type DemoEvent, type DemoTheaterModel } from './demoTheater';

export default function Screen() {
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<UserJourneyReport | null>(null);
  const { connected, testID, progress, step, history, connect, disconnect } = useWSStore();
  const { start: startPollingReport, cancel: cancelPollingReport } = usePollReport<UserJourneyReport>({ maxAttempts: 120 });

  useEffect(() => {
    return () => {
      disconnect();
      cancelPollingReport();
    };
  }, [disconnect, cancelPollingReport]);

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
      startPollingReport(
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
      cancelPollingReport();
      setStarting(false);
    }
  };

  return (
    <div style={page}>
      <style>{animationCSS}</style>
      <header style={header}>
        <div>
          <div style={eyebrow}>UserJourney Demo Theater</div>
          <h1 style={heroTitle}>{model.heroTitle}</h1>
          <p style={heroSubtitle}>把真实 UserJourney 事件投射成一个自动播放的直播竞拍验收短片。</p>
        </div>
        <div style={headerActions}>
          <span style={badge(model.liveBadge)}>{model.liveBadge}</span>
          <button type="button" onClick={handleStart} disabled={starting} style={primaryBtn(starting)}>
            {starting ? '启动中...' : model.primaryActionLabel}
          </button>
          <button type="button" onClick={handleCancel} disabled={!testID} style={ghostBtn(!testID)}>
            停止演示
          </button>
          <Link to="/test" style={exitBtn}>
            返回控制台
          </Link>
        </div>
      </header>

      <main style={theaterGrid}>
        <section style={phonePanel}>
          <div style={panelHeader}>
            <div>
              <div style={sectionKicker}>H5 直播间同步画面</div>
              <h2 style={sectionTitle}>直播间舞台</h2>
            </div>
            <div style={syncPill}>{model.stage === 'idle' ? '等待剧本' : '事件同步中'}</div>
          </div>
          <LivePhone model={model} />
        </section>

        <aside style={evidencePanel}>
          <div style={panelHeader}>
            <div>
              <div style={sectionKicker}>事件证据流</div>
              <h2 style={sectionTitle}>程序员评委视角</h2>
            </div>
            {model.reportPath && (
              <Link to={model.reportPath} style={reportLink}>
                查看技术报告
              </Link>
            )}
          </div>

          <div style={progressCard}>
            <div style={progressTopLine}>
              <span>{model.progressLabel}</span>
              <span>{model.liveBadge}</span>
            </div>
            <div style={progressTrack}>
              <div style={{ ...progressFill, width: model.progressLabel }} />
            </div>
            <div style={technicalLine}>{model.technicalLine}</div>
          </div>

          {model.failureMessage && (
            <div style={errorBox}>
              {model.failureTitle}: {model.failureMessage}
            </div>
          )}

          <div style={metricGrid}>
            <Metric label="最高价" value={model.currentPrice} />
            <Metric label="领先者" value={model.leaderLabel} />
            <Metric label="出价事件" value={String(model.bidCount)} />
            <Metric label="订单数" value={String(model.orderCount)} />
            <Metric label="库存" value={model.stockLabel} />
            <Metric label="当前事件" value={activeEventLabel(model)} />
          </div>

          <div style={railSection}>
            <h3 style={railTitle}>实时事件</h3>
            <div style={eventList}>
              {model.events.length > 0 ? (
                model.events.map((event, index) => <EventCard key={`${event.step}-${index}`} event={event} active={index === model.events.length - 1} />)
              ) : (
                <div style={emptyState}>等待一键启动后接收 UserJourney 进度流。</div>
              )}
            </div>
          </div>

          <div style={railSection}>
            <h3 style={railTitle}>验收结论</h3>
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
        </aside>
      </main>
    </div>
  );
}

function LivePhone({ model }: { model: DemoTheaterModel }) {
  const hasLeader = model.leaderLabel !== '等待领先者';
  const dealHappened = model.orderCount > 0 || model.highlightedEvent === 'order' || model.highlightedEvent === 'fixed_price';
  const skyLamp = model.highlightedEvent === 'sky_lamp';

  return (
    <div style={phoneFrame}>
      <div style={phoneScreen}>
        <div style={phoneTopBar}>
          <span>9:41</span>
          <span>5G</span>
        </div>
        <div style={liveHero}>
          <div style={liveNav}>
            <span style={liveBadge}>LIVE</span>
            <span style={viewerBadge}>2.1w 人在线</span>
          </div>
          <div style={hostBlock}>
            <div style={avatar}>AI</div>
            <div>
              <div style={hostName}>甄选珠宝直播间</div>
              <div style={hostSub}>标准 UserJourney 剧本驱动</div>
            </div>
          </div>
          <div style={productSpotlight}>
            <div style={productImage}>拍品</div>
            <div>
              <div style={productTitle}>天然翡翠手镯</div>
              <div style={productSub}>商家开播 · 实时竞拍 · 一口价库存联动</div>
            </div>
          </div>
          {skyLamp && <div style={lampOverlay} className="demo-lamp-pulse">天灯锁定领先</div>}
          {dealHappened && <div style={dealOverlay} className="demo-deal-pop">成交弹幕</div>}
        </div>

        <div style={priceDock}>
          <div>
            <div style={dockLabel}>当前最高价</div>
            <div style={dockPrice}>{model.currentPrice}</div>
          </div>
          <div style={leaderChip}>{hasLeader ? `${model.leaderLabel} 正在领先` : '等待买家出价'}</div>
        </div>

        <div style={commentStream}>
          <LiveComment text="系统：直播间资产已准备" />
          {hasLeader && <LiveComment text={`${model.leaderLabel}：出价 ${model.currentPrice}`} active />}
          {skyLamp && <LiveComment text="系统：点天灯触发，领先状态高亮" active tone="orange" />}
          {dealHappened && <LiveComment text="系统：订单已生成，库存已扣减" active tone="green" />}
        </div>

        <div style={commerceDock}>
          <div style={stockCard}>
            <span>库存</span>
            <strong>{model.stockLabel === '待验证' ? '待验证' : `库存 ${model.stockLabel}`}</strong>
          </div>
          <div style={bidButton}>立即出价</div>
          <div style={buyButton}>{dealHappened ? '订单已生成' : '一口价抢购'}</div>
        </div>
      </div>
    </div>
  );
}

function LiveComment({ text, active = false, tone = 'blue' }: { text: string; active?: boolean; tone?: 'blue' | 'orange' | 'green' }) {
  return (
    <div style={{ ...commentBubble, ...(active ? activeComment(tone) : null) }} className={active ? 'demo-comment-float' : undefined}>
      {text}
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

function EventCard({ event, active }: { event: DemoEvent; active: boolean }) {
  return (
    <div style={{ ...eventCard, borderColor: eventTone[event.tone], ...(active ? activeEventCard : null) }}>
      <div style={{ ...eventBullet, background: eventTone[event.tone] }} />
      <div>
        <div style={eventTitle}>{event.title}</div>
        <div style={eventDesc}>{event.description}</div>
      </div>
    </div>
  );
}

function activeEventLabel(model: DemoTheaterModel): string {
  return model.events[model.events.length - 1]?.title || '等待启动';
}

const eventTone: Record<DemoEvent['tone'], string> = {
  neutral: '#94a3b8',
  blue: '#38bdf8',
  orange: '#f59e0b',
  green: '#22c55e',
  red: '#ef4444',
};

const animationCSS = `
@keyframes demoCommentFloat {
  0% { transform: translateX(18px); opacity: 0; }
  100% { transform: translateX(0); opacity: 1; }
}
@keyframes demoLampPulse {
  0%, 100% { box-shadow: 0 0 20px rgba(251,191,36,0.45); transform: scale(1); }
  50% { box-shadow: 0 0 48px rgba(251,191,36,0.9); transform: scale(1.04); }
}
@keyframes demoDealPop {
  0% { transform: translateY(18px) scale(0.92); opacity: 0; }
  100% { transform: translateY(0) scale(1); opacity: 1; }
}
.demo-comment-float { animation: demoCommentFloat 420ms ease-out; }
.demo-lamp-pulse { animation: demoLampPulse 1.2s ease-in-out infinite; }
.demo-deal-pop { animation: demoDealPop 420ms ease-out; }
`;

const page: CSSProperties = {
  minHeight: '100vh',
  width: '100vw',
  background:
    'radial-gradient(circle at 12% 8%, rgba(59,130,246,0.25), transparent 30%), radial-gradient(circle at 78% 22%, rgba(251,191,36,0.18), transparent 28%), #07111f',
  color: '#e2e8f0',
  padding: 28,
  display: 'flex',
  flexDirection: 'column',
  gap: 22,
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
  fontSize: 42,
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
  gap: 10,
  flexWrap: 'wrap',
  justifyContent: 'flex-end',
};
const exitBtn: CSSProperties = {
  padding: '10px 14px',
  border: '1px solid rgba(148,163,184,0.35)',
  borderRadius: 999,
  color: '#cbd5e1',
  textDecoration: 'none',
  fontSize: 14,
  background: 'rgba(15,23,42,0.72)',
};
const theaterGrid: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'minmax(430px, 0.95fr) minmax(520px, 1.05fr)',
  gap: 24,
  alignItems: 'stretch',
};
const phonePanel: CSSProperties = {
  background: 'linear-gradient(145deg, rgba(15,23,42,0.92), rgba(14,116,144,0.2))',
  border: '1px solid rgba(125,211,252,0.25)',
  borderRadius: 30,
  padding: 22,
  display: 'flex',
  flexDirection: 'column',
  gap: 18,
};
const evidencePanel: CSSProperties = {
  background: 'rgba(15,23,42,0.82)',
  border: '1px solid rgba(148,163,184,0.18)',
  borderRadius: 30,
  padding: 22,
  display: 'flex',
  flexDirection: 'column',
  gap: 18,
};
const panelHeader: CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  gap: 16,
};
const sectionKicker: CSSProperties = {
  color: '#93c5fd',
  fontSize: 12,
  fontWeight: 900,
  letterSpacing: 2,
};
const sectionTitle: CSSProperties = {
  margin: '4px 0 0',
  color: '#f8fafc',
  fontSize: 24,
};
const syncPill: CSSProperties = {
  border: '1px solid rgba(56,189,248,0.35)',
  borderRadius: 999,
  padding: '8px 12px',
  color: '#a5f3fc',
  background: 'rgba(8,47,73,0.5)',
  fontSize: 13,
  fontWeight: 800,
};
const primaryBtn = (disabled: boolean): CSSProperties => ({
  border: 0,
  borderRadius: 999,
  padding: '11px 18px',
  color: '#03111f',
  background: disabled ? '#64748b' : 'linear-gradient(135deg, #facc15, #fb923c)',
  fontWeight: 900,
  cursor: disabled ? 'not-allowed' : 'pointer',
});
const ghostBtn = (disabled: boolean): CSSProperties => ({
  border: '1px solid rgba(148,163,184,0.28)',
  borderRadius: 999,
  padding: '10px 15px',
  color: disabled ? '#64748b' : '#cbd5e1',
  background: 'rgba(15,23,42,0.55)',
  cursor: disabled ? 'not-allowed' : 'pointer',
});
const phoneFrame: CSSProperties = {
  alignSelf: 'center',
  width: 390,
  minHeight: 720,
  borderRadius: 42,
  padding: 12,
  background: 'linear-gradient(145deg, #020617, #334155)',
  boxShadow: '0 30px 80px rgba(0,0,0,0.48)',
};
const phoneScreen: CSSProperties = {
  minHeight: 696,
  borderRadius: 34,
  overflow: 'hidden',
  background: 'linear-gradient(180deg, #111827 0%, #020617 70%)',
  display: 'flex',
  flexDirection: 'column',
};
const phoneTopBar: CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  padding: '13px 22px 8px',
  color: '#e5e7eb',
  fontSize: 12,
  fontWeight: 800,
};
const liveHero: CSSProperties = {
  position: 'relative',
  minHeight: 300,
  padding: 18,
  display: 'flex',
  flexDirection: 'column',
  justifyContent: 'space-between',
  background:
    'linear-gradient(180deg, rgba(15,23,42,0.1), rgba(2,6,23,0.82)), linear-gradient(135deg, #6d28d9, #0f766e 48%, #92400e)',
};
const liveNav: CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
};
const liveBadge: CSSProperties = {
  borderRadius: 999,
  padding: '6px 10px',
  background: '#ef4444',
  color: '#fff',
  fontSize: 12,
  fontWeight: 900,
};
const viewerBadge: CSSProperties = {
  borderRadius: 999,
  padding: '6px 10px',
  background: 'rgba(2,6,23,0.58)',
  color: '#e2e8f0',
  fontSize: 12,
};
const hostBlock: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 10,
};
const avatar: CSSProperties = {
  width: 42,
  height: 42,
  borderRadius: 999,
  display: 'grid',
  placeItems: 'center',
  background: 'linear-gradient(135deg, #facc15, #fb7185)',
  color: '#111827',
  fontWeight: 900,
};
const hostName: CSSProperties = {
  color: '#fff',
  fontWeight: 900,
};
const hostSub: CSSProperties = {
  color: '#cbd5e1',
  fontSize: 12,
  marginTop: 2,
};
const productSpotlight: CSSProperties = {
  display: 'flex',
  gap: 12,
  alignItems: 'center',
  padding: 12,
  borderRadius: 18,
  background: 'rgba(2,6,23,0.62)',
  border: '1px solid rgba(255,255,255,0.16)',
};
const productImage: CSSProperties = {
  width: 72,
  height: 72,
  borderRadius: 16,
  display: 'grid',
  placeItems: 'center',
  background: 'linear-gradient(135deg, #fef3c7, #a7f3d0)',
  color: '#064e3b',
  fontWeight: 900,
};
const productTitle: CSSProperties = {
  color: '#fff',
  fontSize: 18,
  fontWeight: 900,
};
const productSub: CSSProperties = {
  color: '#cbd5e1',
  fontSize: 12,
  lineHeight: 1.5,
  marginTop: 5,
};
const lampOverlay: CSSProperties = {
  position: 'absolute',
  right: 18,
  top: 92,
  borderRadius: 18,
  padding: '12px 14px',
  background: 'linear-gradient(135deg, #f59e0b, #fde68a)',
  color: '#422006',
  fontWeight: 900,
};
const dealOverlay: CSSProperties = {
  position: 'absolute',
  left: 18,
  bottom: 96,
  borderRadius: 18,
  padding: '12px 14px',
  background: 'linear-gradient(135deg, #22c55e, #bbf7d0)',
  color: '#052e16',
  fontWeight: 900,
};
const priceDock: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 12,
  padding: 16,
  background: '#0f172a',
  borderBottom: '1px solid rgba(148,163,184,0.16)',
};
const dockLabel: CSSProperties = {
  color: '#94a3b8',
  fontSize: 12,
};
const dockPrice: CSSProperties = {
  marginTop: 3,
  color: '#fef3c7',
  fontSize: 34,
  fontWeight: 900,
  letterSpacing: -1,
};
const leaderChip: CSSProperties = {
  borderRadius: 999,
  padding: '8px 10px',
  background: 'rgba(56,189,248,0.12)',
  color: '#a5f3fc',
  fontSize: 12,
  fontWeight: 800,
};
const commentStream: CSSProperties = {
  flex: 1,
  display: 'flex',
  flexDirection: 'column',
  gap: 8,
  padding: 14,
};
const commentBubble: CSSProperties = {
  width: 'fit-content',
  maxWidth: '92%',
  borderRadius: 999,
  padding: '8px 12px',
  background: 'rgba(30,41,59,0.78)',
  color: '#e2e8f0',
  fontSize: 13,
};
const activeComment = (tone: 'blue' | 'orange' | 'green'): CSSProperties => ({
  background:
    tone === 'green'
      ? 'rgba(34,197,94,0.2)'
      : tone === 'orange'
        ? 'rgba(245,158,11,0.22)'
        : 'rgba(56,189,248,0.18)',
  color: tone === 'green' ? '#bbf7d0' : tone === 'orange' ? '#fde68a' : '#a5f3fc',
  border: `1px solid ${tone === 'green' ? 'rgba(34,197,94,0.42)' : tone === 'orange' ? 'rgba(245,158,11,0.42)' : 'rgba(56,189,248,0.35)'}`,
});
const commerceDock: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: '1fr 1fr',
  gap: 10,
  padding: 14,
  background: '#020617',
};
const stockCard: CSSProperties = {
  gridColumn: '1 / -1',
  display: 'flex',
  justifyContent: 'space-between',
  color: '#cbd5e1',
  padding: '10px 12px',
  borderRadius: 14,
  background: 'rgba(15,23,42,0.92)',
};
const bidButton: CSSProperties = {
  textAlign: 'center',
  borderRadius: 16,
  padding: '13px 10px',
  background: 'linear-gradient(135deg, #38bdf8, #2563eb)',
  color: '#fff',
  fontWeight: 900,
};
const buyButton: CSSProperties = {
  textAlign: 'center',
  borderRadius: 16,
  padding: '13px 10px',
  background: 'linear-gradient(135deg, #f97316, #facc15)',
  color: '#1c1917',
  fontWeight: 900,
};
const progressCard: CSSProperties = {
  padding: 14,
  borderRadius: 18,
  background: 'rgba(2,6,23,0.42)',
  border: '1px solid rgba(148,163,184,0.16)',
};
const progressTopLine: CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  color: '#e2e8f0',
  fontWeight: 900,
  marginBottom: 10,
};
const progressTrack: CSSProperties = {
  height: 8,
  borderRadius: 999,
  background: 'rgba(148,163,184,0.2)',
  overflow: 'hidden',
};
const progressFill: CSSProperties = {
  height: '100%',
  borderRadius: 999,
  background: 'linear-gradient(90deg, #38bdf8, #facc15)',
};
const technicalLine: CSSProperties = {
  marginTop: 12,
  color: '#94a3b8',
  fontSize: 12,
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
  wordBreak: 'break-all',
};
const errorBox: CSSProperties = {
  padding: 12,
  border: '1px solid rgba(248,113,113,0.35)',
  borderRadius: 14,
  color: '#fecaca',
  background: 'rgba(127,29,29,0.32)',
};
const metricGrid: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
  gap: 10,
};
const metricBox: CSSProperties = {
  background: 'rgba(2,6,23,0.42)',
  border: '1px solid rgba(148,163,184,0.16)',
  borderRadius: 16,
  padding: 13,
};
const metricLabel: CSSProperties = {
  color: '#94a3b8',
  fontSize: 12,
};
const metricValue: CSSProperties = {
  color: '#f8fafc',
  fontSize: 18,
  fontWeight: 900,
  marginTop: 6,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
};
const railSection: CSSProperties = {
  display: 'grid',
  gap: 12,
};
const railTitle: CSSProperties = {
  margin: 0,
  color: '#f8fafc',
  fontSize: 18,
};
const eventList: CSSProperties = {
  display: 'grid',
  gap: 10,
  maxHeight: 260,
  overflow: 'auto',
};
const eventCard: CSSProperties = {
  display: 'flex',
  gap: 12,
  alignItems: 'flex-start',
  padding: 12,
  border: '1px solid',
  borderRadius: 16,
  background: 'rgba(2,6,23,0.42)',
};
const activeEventCard: CSSProperties = {
  background: 'rgba(15,23,42,0.78)',
  boxShadow: '0 0 0 1px rgba(255,255,255,0.06), 0 14px 40px rgba(0,0,0,0.22)',
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
  padding: 18,
};
const conclusionList: CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 12,
};
const conclusionRow: CSSProperties = {
  display: 'flex',
  gap: 12,
  alignItems: 'flex-start',
};
const conclusionDot = (status: string): CSSProperties => ({
  width: 12,
  height: 12,
  borderRadius: 999,
  marginTop: 4,
  background: status === 'passed' ? '#22c55e' : status === 'failed' ? '#ef4444' : '#64748b',
});
const conclusionTitle: CSSProperties = {
  color: '#f8fafc',
  fontSize: 16,
  fontWeight: 900,
};
const conclusionDesc: CSSProperties = {
  color: '#94a3b8',
  marginTop: 3,
  lineHeight: 1.5,
  fontSize: 13,
};
const reportLink: CSSProperties = {
  color: '#67e8f9',
  textDecoration: 'none',
  fontWeight: 800,
};
const badge = (value: string): CSSProperties => ({
  borderRadius: 999,
  padding: '9px 13px',
  color: value === 'FAILED' ? '#fecaca' : value === 'DONE' ? '#bbf7d0' : '#cffafe',
  border: '1px solid rgba(148,163,184,0.28)',
  background: 'rgba(15,23,42,0.72)',
  fontWeight: 900,
  letterSpacing: 1,
});
