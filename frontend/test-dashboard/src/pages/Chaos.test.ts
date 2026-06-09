import { describe, expect, it } from 'vitest';
import {
  buildBucketsFromProgressHistory,
  buildResilienceSeries,
  describeFaultImplementation,
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
});
