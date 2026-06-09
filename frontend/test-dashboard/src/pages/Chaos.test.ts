import { describe, expect, it } from 'vitest';
import {
  buildCurveAnchors,
  buildBucketsFromProgressHistory,
  buildDemoMetrics,
  buildNarration,
  buildReferenceLineLabels,
  buildReferenceLineLabelProps,
  buildResilienceSeries,
  describeFaultImplementation,
  describeChaosStartButton,
  isChaosStartDisabled,
  theaterStyles,
  summarizeResilienceReport,
  type Bucket,
  type Report,
} from './Chaos';

describe('Chaos resilience presentation', () => {
  it('derives judge-facing resilience curves from chaos buckets', () => {
    const buckets: Bucket[] = [
      { ts: 't0', phase: 'baseline', ok_count: 20, fail_count: 0, avg_latency_ms: 12 },
      { ts: 't1', phase: 'inject', ok_count: 8, fail_count: 12, avg_latency_ms: 88 },
      { ts: 't2', phase: 'recover', ok_count: 20, fail_count: 0, avg_latency_ms: 14 },
    ];

    expect(buildResilienceSeries(buckets)).toEqual([
      { index: 0, phase: 'baseline', errorRatePct: 0, avgLatencyMs: 12, successQps: 20, total: 20 },
      { index: 1, phase: 'inject', errorRatePct: 60, avgLatencyMs: 88, successQps: 8, total: 20 },
      { index: 2, phase: 'recover', errorRatePct: 0, avgLatencyMs: 14, successQps: 20, total: 20 },
    ]);
  });

  it('summarizes the resilience evidence in one sentence', () => {
    const report: Report = {
      baseline_error_rate: 0,
      inject_error_rate: 0.523,
      recover_error_rate: 0,
      detection_latency_ms: 1000,
      recovery_latency_ms: 1000,
    };

    expect(summarizeResilienceReport(report)).toBe(
      '故障注入后错误率从 0.0% 上升到 52.3%，恢复阶段回落到 0.0%，检测延迟 1000ms，恢复延迟 1000ms。',
    );
  });

  it('builds live resilience buckets from websocket progress history', () => {
    expect(
      buildBucketsFromProgressHistory([
        { test_id: 't', progress: 10, step: 'baseline', metrics: { ok: 10, fail: 0, avg_latency_ms: 3 }, ts: 1710000000000 },
        { test_id: 't', progress: 55, step: 'inject', metrics: { ok: 4, fail: 6, avg_latency_ms: 20 }, ts: 1710000001000 },
        { test_id: 't', progress: 80, step: 'unknown', metrics: { ok: 10, fail: 0, avg_latency_ms: 4 }, ts: 1710000002000 },
      ]),
    ).toEqual([
      { ts: '2024-03-09T16:00:00.000Z', phase: 'baseline', ok_count: 10, fail_count: 0, avg_latency_ms: 3 },
      { ts: '2024-03-09T16:00:01.000Z', phase: 'inject', ok_count: 4, fail_count: 6, avg_latency_ms: 20 },
    ]);
  });

  it('explains how the selected chaos fault is implemented', () => {
    expect(describeFaultImplementation('error_rate')).toContain('按概率短路');
    expect(describeFaultImplementation('latency')).toContain('sleep');
    expect(describeFaultImplementation('disconnect')).toContain('连接中断');
  });

  it('builds theater narration from live step and final report evidence', () => {
    const form = { fault_type: 'error_rate' as const, error_rate: 0.5 };
    const report: Report = {
      baseline_error_rate: 0,
      inject_error_rate: 0.5,
      recover_error_rate: 0,
      recovery_latency_ms: 1200,
    };

    expect(buildNarration({ step: 'baseline', progress: 12, form })).toContain('正在采集基线指标');
    expect(buildNarration({ step: 'inject', progress: 45, form })).toContain('约 50% 错误率');
    expect(buildNarration({ step: 'recover', progress: 80, form })).toContain('正在观测系统自愈');
    expect(buildNarration({ step: 'done', progress: 100, form, report })).toBe(summarizeResilienceReport(report));
    expect(buildNarration({ step: 'done', progress: 100, form, report: { buckets: [], baseline_error_rate: 0, inject_error_rate: 0.5, recover_error_rate: 0 } })).toContain('故障注入后错误率');
  });

  it('marks inject, sla breach, and recover anchors on the resilience curve', () => {
    const buckets: Bucket[] = [
      { ts: 't0', phase: 'baseline', ok_count: 20, fail_count: 0, avg_latency_ms: 12 },
      { ts: 't1', phase: 'inject', ok_count: 19, fail_count: 1, avg_latency_ms: 20 },
      { ts: 't2', phase: 'inject', ok_count: 14, fail_count: 6, avg_latency_ms: 88 },
      { ts: 't3', phase: 'recover', ok_count: 20, fail_count: 0, avg_latency_ms: 14 },
    ];

    expect(buildCurveAnchors(buckets)).toEqual({
      injectIndex: 1,
      slaBreachIndex: 2,
      recoverIndex: 3,
    });
  });

  it('omits the sla breach anchor when error rate never crosses the threshold', () => {
    const buckets: Bucket[] = [
      { ts: 't0', phase: 'baseline', ok_count: 20, fail_count: 0, avg_latency_ms: 12 },
      { ts: 't1', phase: 'inject', ok_count: 20, fail_count: 0, avg_latency_ms: 20 },
      { ts: 't2', phase: 'recover', ok_count: 20, fail_count: 0, avg_latency_ms: 14 },
    ];

    expect(buildCurveAnchors(buckets)).toEqual({
      injectIndex: 1,
      recoverIndex: 2,
    });
  });

  it('derives theater metric cards from buckets and report latency', () => {
    const report: Report = {
      recovery_latency_ms: 1400,
      buckets: [
        { ts: 't0', phase: 'baseline', ok_count: 20, fail_count: 0, avg_latency_ms: 12 },
        { ts: 't1', phase: 'baseline', ok_count: 22, fail_count: 0, avg_latency_ms: 12 },
        { ts: 't2', phase: 'inject', ok_count: 8, fail_count: 12, avg_latency_ms: 88 },
        { ts: 't3', phase: 'inject', ok_count: 10, fail_count: 10, avg_latency_ms: 80 },
      ],
    };

    expect(buildDemoMetrics(report)).toEqual({
      peakErrorRatePct: 60,
      lostQps: 12,
      recoveryMs: 1400,
    });
  });

  it('does not report negative lost qps for a stronger inject phase', () => {
    expect(
      buildDemoMetrics({
        buckets: [
          { ts: 't0', phase: 'baseline', ok_count: 5, fail_count: 0, avg_latency_ms: 12 },
          { ts: 't1', phase: 'inject', ok_count: 8, fail_count: 0, avg_latency_ms: 88 },
        ],
      }),
    ).toMatchObject({ lostQps: 0 });
  });

  it('keeps chaos start disabled for the full non-terminal lifecycle', () => {
    expect(isChaosStartDisabled({ running: true, testID: undefined, progress: 0, step: '' })).toBe(true);
    expect(isChaosStartDisabled({ running: false, testID: 't1', progress: 60, step: 'inject' })).toBe(true);
    expect(isChaosStartDisabled({ running: false, testID: 't1', progress: 100, step: 'done' })).toBe(false);
    expect(isChaosStartDisabled({ running: false, testID: 't1', progress: 80, step: 'failed' })).toBe(false);
  });

  it('describes the theater start button in terminal-style copy', () => {
    expect(describeChaosStartButton({ mode: 'theater', disabled: false, running: false })).toBe('./start_theater.sh');
    expect(describeChaosStartButton({ mode: 'manual', disabled: false, running: false })).toBe('启动');
    expect(describeChaosStartButton({ mode: 'theater', disabled: true, running: false })).toBe('演示进行中...');
  });

  it('attaches inline metrics to ReferenceLine labels', () => {
    expect(
      buildReferenceLineLabels({
        peakErrorRatePct: 60,
        lostQps: 12,
        recoveryMs: 1400,
      }),
    ).toEqual({
      inject: 'inject',
      sla: 'peak 60.0%',
      recover: 'recover 1400ms / -12.0QPS',
    });
    expect(buildReferenceLineLabels({ recoveryMs: 1400 }).recover).toBe('recover 1400ms / lost QPS -');
  });



  it('builds compact non-overlapping ReferenceLine label props', () => {
    const labels = buildReferenceLineLabels({ peakErrorRatePct: 60, lostQps: 10.8, recoveryMs: 0 });
    expect(labels).toEqual({
      inject: 'inject',
      sla: 'peak 60.0%',
      recover: 'recover 0ms / -10.8QPS',
    });

    expect(buildReferenceLineLabelProps(labels)).toEqual({
      inject: { value: 'inject', position: 'insideTopLeft', offset: 8, fill: 'var(--color-error-500)' },
      sla: { value: 'peak 60.0%', position: 'insideBottomLeft', offset: 24, fill: 'var(--color-warn-500)' },
      recover: { value: 'recover 0ms / -10.8QPS', position: 'insideTopRight', offset: 8, fill: 'var(--color-primary-500)' },
    });
  });

  it('uses test-dashboard css variables for new theater styles and colors', () => {
    expect(JSON.stringify(theaterStyles)).not.toMatch(/#[0-9a-fA-F]{3,8}/);
    expect(theaterStyles.errorColor).toBe('var(--color-error-500)');
    expect(theaterStyles.primaryColor).toBe('var(--color-primary-500)');
    expect(theaterStyles.narration.borderRadius).toBe('var(--radius-md)');
  });
});
