import { useMemo } from 'react';

// StepTimeline 把 WS 推送的步骤事件渲染为纵向时间轴。
// 每一步显示状态色（绿=成功 / 红=失败 / 灰=等待）+ 名称 + 耗时 + 消息。

export interface StepEvent {
  step: string;
  ok?: boolean;
  duration_ms?: number;
  message?: string;
  ref_id?: number;
  ts?: number;
}

interface Props {
  events: StepEvent[];
}

// 步骤名 → 中文标签（缺失则原样显示）
const stepLabels: Record<string, string> = {
  create_product: '创建拍品',
  create_auction: '创建拍卖',
  wait_started: '等待开拍',
  skylamp_subscribe: '订阅点天灯',
  bid: '出价',
  wait_ended: '等待截拍',
  get_auction: '查询拍卖',
  verify_winner: '校验中标',
  find_orders: '查询订单',
  verify_order_unique: '校验订单唯一',
  prepare: '准备测试数据',
  enter_live: '进入直播间',
  reminder: '关注提醒',
  auction_bid: '竞拍出价',
  sky_lamp: '点天灯',
  fixed_price_purchase: '一口价购买',
  verify: '汇总校验',
  cleanup: '清理副作用',
};

export default function StepTimeline({ events }: Props) {
  // 同名 step 多次（如 bid）做 #N 编号，但保留时间顺序
  const items = useMemo(() => {
    const counter: Record<string, number> = {};
    const stepCounts: Record<string, number> = {};
    events.forEach((e) => {
      stepCounts[e.step] = (stepCounts[e.step] || 0) + 1;
    });
    return events.map((e, i) => {
      counter[e.step] = (counter[e.step] || 0) + 1;
      const total = stepCounts[e.step];
      const idxLabel = total > 1 ? ` #${counter[e.step]}` : '';
      const label = (stepLabels[e.step] || e.step) + idxLabel;
      return { ...e, key: `${e.step}-${i}`, label };
    });
  }, [events]);

  if (items.length === 0) {
    return <div style={{ color: '#9ca3af', fontSize: 13 }}>等待启动...</div>;
  }

  return (
    <ol style={listStyle}>
      {items.map((it, idx) => (
        <li key={it.key} style={liStyle}>
          <Dot ok={it.ok} last={idx === items.length - 1} />
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
              <span style={{ fontWeight: 600, color: it.ok === false ? '#ef4444' : '#1f2937' }}>
                {it.label}
              </span>
              <span style={{ color: '#9ca3af', fontFamily: 'monospace', fontSize: 12 }}>
                {fmtDuration(it.duration_ms)}
              </span>
            </div>
            {it.ref_id ? (
              <div style={{ fontSize: 12, color: '#6b7280' }}>ref_id: {it.ref_id}</div>
            ) : null}
            {it.message ? (
              <div style={{ fontSize: 12, color: it.ok === false ? '#ef4444' : '#6b7280' }}>
                {it.message}
              </div>
            ) : null}
          </div>
        </li>
      ))}
    </ol>
  );
}

function Dot({ ok, last }: { ok?: boolean; last: boolean }) {
  const color = ok === undefined ? '#9ca3af' : ok ? '#10b981' : '#ef4444';
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', marginRight: 12 }}>
      <span
        style={{
          width: 12,
          height: 12,
          borderRadius: '50%',
          background: color,
          boxShadow: `0 0 0 3px ${color}22`,
        }}
      />
      {!last && <span style={{ flex: 1, width: 2, background: '#e5e7eb', marginTop: 4, minHeight: 24 }} />}
    </div>
  );
}

function fmtDuration(ms?: number): string {
  if (ms == null) return '';
  if (ms < 1000) return `${ms} ms`;
  return `${(ms / 1000).toFixed(2)} s`;
}

const listStyle: React.CSSProperties = {
  listStyle: 'none',
  padding: 0,
  margin: 0,
};

const liStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'stretch',
  paddingBottom: 12,
};
