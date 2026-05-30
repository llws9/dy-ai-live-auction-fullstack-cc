import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getHistory, type TestResult } from '@/api/test';

export default function History() {
  const [items, setItems] = useState<TestResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const r = await getHistory({ page: 1, page_size: 50 });
      setItems(r.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
        <h1 style={{ fontSize: 22 }}>历史记录</h1>
        <button
          type="button"
          onClick={load}
          disabled={loading}
          style={{
            padding: '6px 12px',
            border: '1px solid #d1d5db',
            borderRadius: 6,
            background: '#fff',
            cursor: loading ? 'not-allowed' : 'pointer',
          }}
        >
          {loading ? '加载中...' : '刷新'}
        </button>
      </div>

      {error && <div style={{ color: '#ef4444', marginBottom: 12 }}>错误：{error}</div>}

      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
        <thead>
          <tr style={{ background: '#f3f4f6', textAlign: 'left' }}>
            <th style={{ padding: 8 }}>ID</th>
            <th style={{ padding: 8 }}>类型</th>
            <th style={{ padding: 8 }}>状态</th>
            <th style={{ padding: 8 }}>创建时间</th>
            <th style={{ padding: 8 }}>操作</th>
          </tr>
        </thead>
        <tbody>
          {items.length === 0 && !loading && (
            <tr>
              <td colSpan={5} style={{ padding: 16, textAlign: 'center', color: '#9ca3af' }}>
                暂无记录
              </td>
            </tr>
          )}
          {items.map((it) => (
            <tr key={it.ID} style={{ borderBottom: '1px solid #e5e7eb' }}>
              <td style={{ padding: 8, fontFamily: 'monospace' }}>{it.ID.slice(0, 8)}…</td>
              <td style={{ padding: 8 }}>{it.TestType}</td>
              <td style={{ padding: 8 }}>{it.Status}</td>
              <td style={{ padding: 8 }}>{new Date(it.CreatedAt).toLocaleString()}</td>
              <td style={{ padding: 8 }}>
                <Link to={`/test/report/${it.ID}`}>查看报告</Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
