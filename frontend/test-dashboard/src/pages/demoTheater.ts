import type { DemoSnapshot, UserJourneyConfig, UserJourneyReport } from '@/api/test';
import type { ProgressMsg } from '@/store/wsStore';

export const DEMO_USER_JOURNEY_CONFIG: Required<UserJourneyConfig> = {
  include_reminder: true,
  include_sky_lamp: true,
  include_fixed_price: true,
  auction_duration_sec: 30,
  buyer_count: 1,
  keep_evidence: true,
};

export type DemoStage = 'idle' | 'starting' | 'running' | 'success' | 'failed';
export type ConclusionStatus = 'pending' | 'passed' | 'failed';

export interface DemoEvent {
  step: string;
  title: string;
  description: string;
  tone: 'neutral' | 'blue' | 'orange' | 'green' | 'red';
}

export interface DemoConclusion {
  title: string;
  description: string;
  status: ConclusionStatus;
}

export interface DemoTheaterModel {
  stage: DemoStage;
  heroTitle: string;
  primaryActionLabel: string;
  liveBadge: 'READY' | 'STARTING' | 'LIVE' | 'DONE' | 'FAILED';
  currentPrice: string;
  leaderLabel: string;
  bidCount: number;
  orderCount: number;
  stockLabel: string;
  highlightedEvent: DemoSnapshot['highlighted_event'] | 'idle';
  events: DemoEvent[];
  conclusions: DemoConclusion[];
  progressLabel: string;
  technicalLine: string;
  reportPath: string | null;
  failureTitle: string | null;
  failureMessage: string | null;
}

export interface BuildDemoTheaterModelInput {
  connected: boolean;
  testID: string | null;
  progress: number;
  step: string;
  history: ProgressMsg[];
  report: UserJourneyReport | null;
  error: string | null;
  starting: boolean;
}

const STEP_EVENTS: Record<string, DemoEvent> = {
  prepare: {
    step: 'prepare',
    title: '演示资产已创建',
    description: '商家、商品、直播间、竞拍规则和买家资金已准备完成',
    tone: 'blue',
  },
  enter_live: {
    step: 'enter_live',
    title: '买家进入直播间',
    description: '直播间切换为可交互状态',
    tone: 'blue',
  },
  reminder: {
    step: 'reminder',
    title: '关注提醒已验证',
    description: '买家关注直播间，提醒状态完成回读',
    tone: 'neutral',
  },
  auction_bid: {
    step: 'auction_bid',
    title: '实时出价发生',
    description: '当前价刷新，领先者进入竞拍态',
    tone: 'blue',
  },
  sky_lamp: {
    step: 'sky_lamp',
    title: '点天灯触发',
    description: '高权重竞价反馈出现，领先状态被锁定展示',
    tone: 'orange',
  },
  fixed_price_purchase: {
    step: 'fixed_price_purchase',
    title: '一口价成交',
    description: '订单生成，库存开始扣减',
    tone: 'green',
  },
  verify: {
    step: 'verify',
    title: '闭环校验通过',
    description: '订单、库存、余额和竞拍结果完成一致性校验',
    tone: 'green',
  },
  cleanup: {
    step: 'cleanup',
    title: '证据已保留',
    description: '演示报告可用于技术下钻',
    tone: 'neutral',
  },
};

const CONCLUSIONS: DemoConclusion[] = [
  { title: '业务闭环成立', description: '进房、竞拍、成交、订单链路通过', status: 'pending' },
  { title: '并发结果唯一', description: '赢家唯一，订单唯一，无重复成交', status: 'pending' },
  { title: '资产状态一致', description: '库存、余额、订单状态对齐', status: 'pending' },
];

export function buildDemoTheaterModel(input: BuildDemoTheaterModelInput): DemoTheaterModel {
  const snapshot = latestSnapshot(input);
  const reportID = input.report?.test_run_id || input.testID;
  const stage = resolveStage(input);
  const events = input.history.map((message) => STEP_EVENTS[message.step]).filter((event): event is DemoEvent => Boolean(event));

  return {
    stage,
    heroTitle: 'AI 直播竞拍全链路验收',
    primaryActionLabel: stage === 'failed' || stage === 'success' ? '重新演示' : '开始演示',
    liveBadge: resolveLiveBadge(stage),
    currentPrice: snapshot?.current_price ? `¥${snapshot.current_price}` : '待启动',
    leaderLabel: snapshot?.leader_label || '等待领先者',
    bidCount: snapshot?.bid_count ?? 0,
    orderCount: snapshot?.order_count ?? 0,
    stockLabel: formatStock(snapshot, input.report),
    highlightedEvent: snapshot?.highlighted_event || 'idle',
    events,
    conclusions: CONCLUSIONS.map((item) => ({
      ...item,
      status: stage === 'success' ? 'passed' : stage === 'failed' ? 'failed' : 'pending',
    })),
    progressLabel: `${Math.max(0, Math.min(100, input.progress))}%`,
    technicalLine: technicalLine(input),
    reportPath: reportID ? `/test/report/${reportID}` : null,
    failureTitle: stage === 'failed' ? `${stepBusinessName(input.step)}阶段失败` : null,
    failureMessage: input.error || input.report?.error || null,
  };
}

function latestSnapshot(input: BuildDemoTheaterModelInput): DemoSnapshot | undefined {
  const fromHistory = [...input.history]
    .reverse()
    .map((message) => message.metrics?.demo_snapshot)
    .find(Boolean) as DemoSnapshot | undefined;

  return input.report?.demo_snapshot || fromHistory;
}

function resolveStage(input: BuildDemoTheaterModelInput): DemoStage {
  if (input.error || input.report?.all_ok === false) return 'failed';
  if (input.report?.all_ok === true) return 'success';
  if (input.starting) return 'starting';
  if (input.testID || input.connected || input.progress > 0) return 'running';
  return 'idle';
}

function resolveLiveBadge(stage: DemoStage): DemoTheaterModel['liveBadge'] {
  if (stage === 'idle') return 'READY';
  if (stage === 'starting') return 'STARTING';
  if (stage === 'success') return 'DONE';
  if (stage === 'failed') return 'FAILED';
  return 'LIVE';
}

function formatStock(snapshot: DemoSnapshot | undefined, report: UserJourneyReport | null): string {
  const before = snapshot?.stock_before ?? report?.stock_before;
  const after = snapshot?.stock_after ?? report?.stock_after;
  if (before == null && after == null) return '待验证';
  return `${before ?? '-'} → ${after ?? '-'}`;
}

function stepBusinessName(step: string): string {
  const names: Record<string, string> = {
    prepare: '演示准备',
    enter_live: '进直播间',
    reminder: '关注提醒',
    auction_bid: '出价',
    sky_lamp: '点天灯',
    fixed_price_purchase: '一口价购买',
    verify: '汇总校验',
    cleanup: '证据清理',
  };
  return names[step] || '演示';
}

function technicalLine(input: BuildDemoTheaterModelInput): string {
  if (!input.testID && !input.report?.test_run_id) return '等待一键启动 UserJourney 标准剧本';

  const ws = input.connected ? 'WS 已连接' : 'WS 未连接';
  return `test_id=${input.testID || input.report?.test_run_id || '-'} · ${ws} · step=${input.step || '-'}`;
}
