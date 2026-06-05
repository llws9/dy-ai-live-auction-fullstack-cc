import { describe, expect, it } from 'vitest';
import type { UserJourneyReport } from '@/api/test';
import type { ProgressMsg } from '@/store/wsStore';
import { buildDemoTheaterModel, DEMO_USER_JOURNEY_CONFIG } from './demoTheater';

describe('demoTheater', () => {
  it('uses standard user journey config', () => {
    expect(DEMO_USER_JOURNEY_CONFIG).toEqual({
      include_reminder: true,
      include_sky_lamp: true,
      include_fixed_price: true,
      auction_duration_sec: 30,
      buyer_count: 1,
      keep_evidence: true,
    });
  });

  it('builds idle model', () => {
    const model = buildDemoTheaterModel(baseInput());
    expect(model.stage).toBe('idle');
    expect(model.currentPrice).toBe('待启动');
    expect(model.technicalLine).toBe('等待一键启动 UserJourney 标准剧本');
  });

  it('maps sky lamp progress into live story', () => {
    const model = buildDemoTheaterModel({
      ...baseInput(),
      connected: true,
      testID: 'tj_live',
      progress: 62,
      step: 'sky_lamp',
      history: [
        progress('tj_live', 50, 'auction_bid', {
          demo_snapshot: {
            current_price: '110.00',
            leader_label: '买家 2001',
            bid_count: 1,
            stock_before: 1,
            stock_after: 1,
            highlighted_event: 'bid',
          },
        }),
        progress('tj_live', 62, 'sky_lamp', {
          demo_snapshot: {
            current_price: '110.00',
            leader_label: '买家 2001',
            bid_count: 1,
            highlighted_event: 'sky_lamp',
          },
        }),
      ],
    });
    expect(model.stage).toBe('running');
    expect(model.liveBadge).toBe('LIVE');
    expect(model.currentPrice).toBe('¥110.00');
    expect(model.leaderLabel).toBe('买家 2001');
    expect(model.events[model.events.length - 1]?.title).toBe('点天灯触发');
  });

  it('ignores malformed demo snapshot metrics', () => {
    const model = buildDemoTheaterModel({
      ...baseInput(),
      connected: true,
      testID: 'tj_bad_snapshot',
      progress: 50,
      step: 'auction_bid',
      history: [
        progress('tj_bad_snapshot', 50, 'auction_bid', {
          demo_snapshot: {
            current_price: 110,
            leader_label: ['买家 2001'],
            bid_count: '1',
          },
        }),
      ],
    });

    expect(model.currentPrice).toBe('待启动');
    expect(model.leaderLabel).toBe('等待领先者');
    expect(model.bidCount).toBe(0);
  });

  it('shows success conclusions from report', () => {
    const report: UserJourneyReport = {
      test_run_id: 'tj_done',
      all_ok: true,
      order_id: 501,
      stock_before: 1,
      stock_after: 0,
      demo_snapshot: {
        current_price: '110.00',
        leader_label: '买家 2001',
        bid_count: 1,
        order_count: 1,
        stock_before: 1,
        stock_after: 0,
        highlighted_event: 'verify',
      },
    };
    const model = buildDemoTheaterModel({ ...baseInput(), testID: 'tj_done', progress: 100, step: 'verify', report });
    expect(model.stage).toBe('success');
    expect(model.conclusions.every((item) => item.status === 'passed')).toBe(true);
    expect(model.reportPath).toBe('/test/report/tj_done');
    expect(model.stockLabel).toBe('1 → 0');
  });

  it('shows business-stage failure', () => {
    const model = buildDemoTheaterModel({
      ...baseInput(),
      testID: 'tj_fail',
      progress: 62,
      step: 'sky_lamp',
      report: { test_run_id: 'tj_fail', all_ok: false, error: 'sky_lamp failed: upstream timeout' },
    });
    expect(model.stage).toBe('failed');
    expect(model.failureTitle).toBe('点天灯阶段失败');
    expect(model.failureMessage).toContain('upstream timeout');
    expect(model.reportPath).toBe('/test/report/tj_fail');
  });
});

function baseInput() {
  return {
    connected: false,
    testID: null,
    progress: 0,
    step: '',
    history: [],
    report: null,
    error: null,
    starting: false,
  };
}

function progress(testID: string, value: number, step: string, metrics: Record<string, unknown>): ProgressMsg {
  return { test_id: testID, progress: value, step, metrics, ts: Date.now() };
}
