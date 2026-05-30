import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { getHistory, type TestResult } from '@/api/test';

// 大屏模式：1920×1080 优化、暗色主题、自动轮播多张卡片
// 路由 `/test/screen`，独立于 Layout，无侧栏
export default function Screen() {
  const [items, setItems] = useState<TestResult[]>([]);
  const [tick, setTick] = useState(0);
  const [now, setNow] = useState(new Date());

  // 每 3 秒拉一次最近 20 条
  useEffect(() => {
    let stopped = false;
    const fetchOnce = async () => {
      try {
        const r = await getHistory({ page: 1, page_size: 20 });
        if (!stopped) setItems(r.items);
      } catch {
        /* ignore */
      }
    };
    fetchOnce();
    const id = setInterval(fetchOnce, 3000);
    const clk = setInterval(() => setNow(new Date()), 1000);
    return () => {
      stopped = true;
      clearInterval(id);
      clearInterval(clk);
    };
  }, []);

  // 每 5 秒切一组卡片
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 5000);
    return () => clearInterval(id);
  }, []);

  const stats = useMemo(() => {
    const total = items.length;
    const ok = items.filter((x) => x.Status === 'completed').length;
    const fail = items.filter((x) => x.Status === 'failed').length;
    const running = items.filter((x) => x.Status === 'running').length;
    const cancelled = items.filter((x) => x.Status === 'cancelled').length;
    return { total, ok, fail, running, cancelled };
  }, [items]);

  // 轮播 4 条最近的非 running 任务
  const recent = items.filter((x) => x.Status !== 'running').slice(0, 8);
  const start = (tick * 4) % Math.max(1, recent.length);
  const visible: TestResult[] = [];
  for (let i = 0; i < 4 && recent.length > 0; i++) {
    visible.push(recent[(start + i) % recent.length]);
  }

  return (
    <div style={page}>
      <header style={header}>
        <div style={{ fontSize: 36, fontWeight: 700, color: '#fbbf24' }}>测试演示大屏</div>
        <div style={{ fontSize: 18, color: '#94a3b8' }}>{now.toLocaleString()}</div>
        <Link to="/test" style={exitBtn}>
          返回控制台
        </Link>
      </header>

      <section style={statsRow}>
        <Stat label="总任务" value={String(stats.total)} color="#3b82f6" />
        <Stat label="成功" value={String(stats.ok)} color="#10b981" />
        <Stat label="失败" value={String(stats.fail)} color="#ef4444" />
        <Stat label="运行中" value={String(stats.running)} color="#fbbf24" />
        <Stat label="取消" value={String(stats.cancelled)} color="#94a3b8" />
        <Stat
          label="通过率"
          value={`${stats.total > 0 ? ((stats.ok / stats.total) * 100).toFixed(1) : '0.0'}%`}
          color="#34d399"
        />
      </section>

      <section style={cardGrid}>
        {visible.map((it, i) => (
          <RecentCard key={it.ID + i} item={it} />
        ))}
        {visible.length === 0 && (
          <div style={{ ...recentCard, gridColumn: '1 / -1', textAlign: 'center', color: '#64748b', fontSize: 24 }}>
            暂无历史任务，请先在控制台启动一些测试
          </div>
        )}
      </section>

      <footer style={footer}>
        <span>5s 切换 · 3s 刷新 · F11 进入全屏</span>
      </footer>
    </div>
  );
}

function RecentCard({ item }: { item: TestResult }) {
  const okColor = item.Status === 'completed' ? '#10b981' : item.Status === 'failed' ? '#ef4444' : '#fbbf24';
  let summary = '';
  try {
    const r = JSON.parse(item.ResultJSON || '{}');
    if (r.all_ok !== undefined) summary = r.all_ok ? 'all_ok' : 'has_failure';
    if (r.completed_bids !== undefined) summary = `bids=${r.completed_bids}`;
    if (r.qps_avg !== undefined) summary = `qps=${Number(r.qps_avg).toFixed(1)}`;
  } catch {
    /* ignore */
  }
  return (
    <div style={recentCard}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
        <span style={{ fontSize: 14, color: '#94a3b8' }}>{item.TestType}</span>
        <span style={{ fontSize: 14, color: okColor, fontWeight: 700 }}>{item.Status.toUpperCase()}</span>
      </div>
      <div style={{ fontSize: 18, color: '#e2e8f0', wordBreak: 'break-all', marginBottom: 12 }}>{item.ID}</div>
      <div style={{ fontSize: 16, color: '#fbbf24', fontFamily: 'monospace' }}>{summary || '(no summary)'}</div>
      <div style={{ fontSize: 12, color: '#64748b', marginTop: 12 }}>
        {new Date(item.CreatedAt).toLocaleTimeString()}
      </div>
    </div>
  );
}

function Stat({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div style={statBox}>
      <div style={{ fontSize: 18, color: '#94a3b8', marginBottom: 8 }}>{label}</div>
      <div style={{ fontSize: 56, fontWeight: 800, color, lineHeight: 1, fontFamily: 'monospace' }}>{value}</div>
    </div>
  );
}

const page: React.CSSProperties = {
  minHeight: '100vh',
  width: '100vw',
  background: '#0f172a',
  color: '#e2e8f0',
  padding: 24,
  display: 'flex',
  flexDirection: 'column',
  gap: 24,
};
const header: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
};
const exitBtn: React.CSSProperties = {
  padding: '6px 14px',
  border: '1px solid #475569',
  borderRadius: 6,
  color: '#94a3b8',
  textDecoration: 'none',
  fontSize: 14,
};
const statsRow: React.CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(6, 1fr)',
  gap: 16,
};
const statBox: React.CSSProperties = {
  background: '#1e293b',
  border: '1px solid #334155',
  borderRadius: 12,
  padding: 24,
  textAlign: 'center',
};
const cardGrid: React.CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(4, 1fr)',
  gap: 16,
  flex: 1,
};
const recentCard: React.CSSProperties = {
  background: '#1e293b',
  border: '1px solid #334155',
  borderRadius: 12,
  padding: 20,
  display: 'flex',
  flexDirection: 'column',
  justifyContent: 'space-between',
};
const footer: React.CSSProperties = {
  fontSize: 12,
  color: '#475569',
  textAlign: 'center',
};
