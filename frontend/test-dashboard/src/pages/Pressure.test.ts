import { describe, expect, it } from 'vitest';
import {
  describePressureStartButton,
  describeWSStatus,
  failureMessageFromMetrics,
  isPressureStartDisabled,
} from './Pressure';

describe('Pressure status text', () => {
  it('shows failure before treating 100 percent progress as completed', () => {
    expect(describeWSStatus(true, 100, 'failed')).toBe('压测失败，实时连接已结束');
  });

  it('extracts backend failure reason from progress metrics', () => {
    expect(failureMessageFromMetrics({ error: 'pressure create auction failed' })).toBe('pressure create auction failed');
  });

  it('keeps the start button disabled while a submitted test has not reached a terminal state', () => {
    expect(isPressureStartDisabled(false, 'test-running', 0, '')).toBe(true);
    expect(describePressureStartButton(false, 'test-running', 0, '')).toBe('压测进行中');
  });

  it('allows starting again after the current pressure test reaches a terminal state', () => {
    expect(isPressureStartDisabled(false, 'test-done', 100, 'done')).toBe(false);
    expect(isPressureStartDisabled(false, 'test-failed', 100, 'failed')).toBe(false);
  });
});
