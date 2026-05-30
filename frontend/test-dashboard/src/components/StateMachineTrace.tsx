import { useMemo } from 'react';

export interface TraceEntry {
  at: string; // RFC3339
  state: string;
  note?: string;
}

export interface StateMachineTraceProps {
  trace: TraceEntry[];
}

/**
 * 状态机流图：把 Trace 串成一条横向的"节点 → 节点"链。
 * 节点颜色由状态名决定；同名节点重复出现会画多次（如 Sending → Unknown → Sending → Confirmed）。
 */

const NODES = [
  'Pending',
  'Sending',
  'Confirmed',
  'Unknown',
  'Probing',
  'DLQ',
  'Rejected',
];

const NODE_COLORS: Record<string, string> = {
  Pending: '#94a3b8',
  Sending: '#3b82f6',
  Confirmed: '#10b981',
  Unknown: '#f59e0b',
  Probing: '#8b5cf6',
  DLQ: '#ef4444',
  Rejected: '#dc2626',
};

export default function StateMachineTrace({ trace }: StateMachineTraceProps) {
  const visited = useMemo(() => {
    const set = new Set<string>();
    trace.forEach((t) => set.add(t.state));
    return set;
  }, [trace]);

  if (!trace || trace.length === 0) {
    return <div style={{ color: '#9ca3af', fontSize: 13 }}>暂无轨迹。</div>;
  }

  return (
    <div>
      {/* 全状态总览 */}
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
        {NODES.map((n) => (
          <span
            key={n}
            style={{
              padding: '4px 12px',
              borderRadius: 16,
              fontSize: 12,
              border: `1px solid ${NODE_COLORS[n]}`,
              background: visited.has(n) ? NODE_COLORS[n] : '#fff',
              color: visited.has(n) ? '#fff' : NODE_COLORS[n],
              opacity: visited.has(n) ? 1 : 0.5,
            }}
          >
            {n}
          </span>
        ))}
      </div>

      {/* Trace 链 */}
      <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start', overflowX: 'auto', paddingBottom: 8 }}>
        {trace.map((t, idx) => (
          <div key={idx} style={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
            <div
              style={{
                minWidth: 120,
                padding: '8px 12px',
                borderRadius: 8,
                border: `2px solid ${NODE_COLORS[t.state] || '#94a3b8'}`,
                background: '#fff',
              }}
            >
              <div
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  color: NODE_COLORS[t.state] || '#1f2937',
                }}
              >
                {idx + 1}. {t.state}
              </div>
              {t.note && (
                <div style={{ fontSize: 11, color: '#6b7280', marginTop: 4 }} title={t.note}>
                  {t.note.length > 32 ? t.note.slice(0, 32) + '…' : t.note}
                </div>
              )}
              <div style={{ fontSize: 10, color: '#9ca3af', marginTop: 4 }}>
                {fmtTime(t.at)}
              </div>
            </div>
            {idx < trace.length - 1 && (
              <div style={{ padding: '0 6px', color: '#cbd5e1', fontSize: 18 }}>→</div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function fmtTime(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toISOString().slice(11, 23);
}
