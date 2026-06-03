import { useMemo } from 'react';

export interface AntiSnipeTimelineEvent {
  at: string; // RFC3339
  user_id: number;
  bid_ok: boolean;
  delay_used_sec: number;
  end_time: string; // 当前 EndTime 快照
  triggered: boolean; // 本次出价是否触发延时
}

export interface AntiSnipeTimelineProps {
  originalEndTime?: string;
  actualEndTime?: string;
  events: AntiSnipeTimelineEvent[];
  height?: number;
}

/**
 * 横向时间轴：
 * - 左端为最早出价时间，右端为 max(actualEndTime, originalEndTime, 最后事件)
 * - 蓝色刻度线：原计划截拍点
 * - 红色刻度线：实际截拍点
 * - 圆点：每次出价（绿=成功 + 触发延时，灰=成功未触发，红=失败）
 * - 顶部进度条显示已用延时占比
 */
export default function AntiSnipeTimeline({
  originalEndTime,
  actualEndTime,
  events,
  height = 80,
}: AntiSnipeTimelineProps) {
  const layout = useMemo(() => buildLayout(events, originalEndTime, actualEndTime), [
    events,
    originalEndTime,
    actualEndTime,
  ]);

  if (!layout) {
    return (
      <div style={{ color: '#9ca3af', fontSize: 13 }}>
        暂无时间轴数据，启动后将实时绘制。
      </div>
    );
  }

  const { startMs, endMs, eventPoints, originalPct, actualPct } = layout;
  const totalMs = endMs - startMs;

  return (
    <div>
      <div style={{ position: 'relative', height, background: '#f8fafc', border: '1px solid #e5e7eb', borderRadius: 6 }}>
        {/* 主轴 */}
        <div style={{ position: 'absolute', left: 16, right: 16, top: '50%', height: 2, background: '#cbd5e1' }} />

        {/* 原计划截拍点 */}
        {originalPct !== null && (
          <Marker pct={originalPct} color="#3b82f6" label="原截拍" />
        )}
        {/* 实际截拍点 */}
        {actualPct !== null && (
          <Marker pct={actualPct} color="#ef4444" label="实际截拍" top />
        )}

        {/* 出价点 */}
        {eventPoints.map((p, idx) => (
          <div
            key={idx}
            title={`#${idx + 1} user=${p.user_id} ok=${p.bid_ok} triggered=${p.triggered} delay=${p.delay_used_sec}s`}
            style={{
              position: 'absolute',
              left: `calc(${p.pct}% + 16px - ${(p.pct / 100) * 32}px)`,
              top: '50%',
              transform: 'translate(-50%, -50%)',
              width: 10,
              height: 10,
              borderRadius: '50%',
              background: dotColor(p),
              border: '2px solid #fff',
              boxShadow: '0 0 2px rgba(0,0,0,.2)',
            }}
          />
        ))}
      </div>

      {/* 图例 */}
      <div style={{ marginTop: 8, display: 'flex', gap: 16, flexWrap: 'wrap', fontSize: 12, color: '#6b7280' }}>
        <Legend color="#3b82f6" text="原计划截拍点" />
        <Legend color="#ef4444" text="实际截拍点" />
        <Legend color="#10b981" text="出价（触发延时）" />
        <Legend color="#9ca3af" text="出价（未触发）" />
        <Legend color="#ef4444" text="出价失败" dot />
        <span>跨度：{(totalMs / 1000).toFixed(1)}s</span>
      </div>
    </div>
  );
}

function Marker({ pct, color, label, top = false }: { pct: number; color: string; label: string; top?: boolean }) {
  return (
    <div
      style={{
        position: 'absolute',
        left: `calc(${pct}% + 16px - ${(pct / 100) * 32}px)`,
        top: 0,
        bottom: 0,
        width: 2,
        background: color,
      }}
    >
      <div
        style={{
          position: 'absolute',
          [top ? 'top' : 'bottom']: -2,
          left: 4,
          fontSize: 11,
          color,
          whiteSpace: 'nowrap',
          background: '#fff',
          padding: '0 4px',
        }}
      >
        {label}
      </div>
    </div>
  );
}

function Legend({ color, text, dot }: { color: string; text: string; dot?: boolean }) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      <span
        style={{
          display: 'inline-block',
          width: dot ? 10 : 14,
          height: dot ? 10 : 4,
          background: color,
          borderRadius: dot ? '50%' : 2,
        }}
      />
      {text}
    </span>
  );
}

function dotColor(p: { bid_ok: boolean; triggered: boolean }) {
  if (!p.bid_ok) return '#ef4444';
  if (p.triggered) return '#10b981';
  return '#9ca3af';
}

function buildLayout(
  events: AntiSnipeTimelineEvent[],
  originalEndTime?: string,
  actualEndTime?: string,
) {
  if (events.length === 0 && !originalEndTime && !actualEndTime) return null;

  const tsList: number[] = [];
  const parseCache = new Map<string, number>();
  events.forEach((e) => {
    const t = Date.parse(e.at);
    parseCache.set(e.at, t);
    if (!Number.isNaN(t)) tsList.push(t);
  });
  const origMs = originalEndTime ? Date.parse(originalEndTime) : NaN;
  const actMs = actualEndTime ? Date.parse(actualEndTime) : NaN;
  if (!Number.isNaN(origMs)) tsList.push(origMs);
  if (!Number.isNaN(actMs)) tsList.push(actMs);
  if (tsList.length === 0) return null;

  let startMs = Math.min(...tsList);
  let endMs = Math.max(...tsList);
  // 防止零跨度
  if (endMs - startMs < 1000) {
    endMs = startMs + 1000;
  }
  // 头尾各留 5% padding
  const span = endMs - startMs;
  startMs -= span * 0.05;
  endMs += span * 0.05;
  const total = endMs - startMs;

  const pctOf = (ms: number) => Math.max(0, Math.min(100, ((ms - startMs) / total) * 100));

  return {
    startMs,
    endMs,
    originalPct: Number.isNaN(origMs) ? null : pctOf(origMs),
    actualPct: Number.isNaN(actMs) ? null : pctOf(actMs),
    eventPoints: events.map((e) => ({
      pct: pctOf(parseCache.get(e.at) ?? Date.parse(e.at)),
      user_id: e.user_id,
      bid_ok: e.bid_ok,
      triggered: e.triggered,
      delay_used_sec: e.delay_used_sec,
    })),
  };
}
