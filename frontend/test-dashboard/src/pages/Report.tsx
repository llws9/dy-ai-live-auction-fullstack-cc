import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { getReport, type TestResult, type UserJourneyReport } from '@/api/test';

export default function Report() {
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<TestResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    getReport(id).then(setData).catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, [id]);

  return (
    <div>
      <div style={{ marginBottom: 12 }}>
        <Link to="/test/history">← 返回历史</Link>
      </div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>测试报告</h1>

      {error && <div style={{ color: '#ef4444' }}>错误：{error}</div>}
      {!data && !error && <div style={{ color: '#9ca3af' }}>加载中...</div>}
      {data && (
        <div style={{ display: 'grid', gap: 12 }}>
          <Field label="ID" value={data.ID} />
          <Field label="类型" value={data.TestType} />
          <Field label="状态" value={data.Status} />
          <Field label="创建时间" value={new Date(data.CreatedAt).toLocaleString()} />
          {data.CompletedAt && (
            <Field label="完成时间" value={new Date(data.CompletedAt).toLocaleString()} />
          )}
          <Field label="ReplayToken" value={data.ReplayToken || '-'} />
          {data.ErrorMsg && <Field label="错误" value={data.ErrorMsg} />}
          {data.TestType === 'user_journey' && <UserJourneySummary raw={data.ResultJSON} />}
          <details>
            <summary style={{ cursor: 'pointer' }}>Config JSON</summary>
            <pre style={preStyle}>{prettyJSON(data.ConfigJSON)}</pre>
          </details>
          <details open>
            <summary style={{ cursor: 'pointer' }}>Result JSON</summary>
            <pre style={preStyle}>{prettyJSON(data.ResultJSON)}</pre>
          </details>
        </div>
      )}
    </div>
  );
}

function UserJourneySummary({ raw }: { raw: string }) {
  const report = parseUserJourneyReport(raw);
  if (!report) return null;
  return (
    <section style={{ border: '1px solid #e5e7eb', borderRadius: 6, padding: 12 }}>
      <h3 style={{ fontSize: 15, marginBottom: 10 }}>用户验收摘要</h3>
      <div style={{ display: 'grid', gap: 8 }}>
        <Field label="整体成功" value={report.all_ok ? 'OK' : 'FAIL'} />
        <Field label="资源 ID" value={`live=${fmtNum(report.live_stream_id)} auction=${fmtNum(report.auction_id)} fixed=${fmtNum(report.fixed_price_item_id)}`} />
        <Field label="订单 ID" value={fmtNum(report.order_id)} />
        <Field label="余额变化" value={`${report.balance_before ?? '-'} → ${report.balance_after ?? '-'}`} />
        <Field label="库存变化" value={`${fmtNum(report.stock_before)} → ${fmtNum(report.stock_after)}`} />
        {report.warnings && report.warnings.length > 0 && <Field label="Warnings" value={report.warnings.join('；')} />}
      </div>
    </section>
  );
}

const preStyle: React.CSSProperties = {
  background: '#0f172a',
  color: '#e2e8f0',
  padding: 12,
  borderRadius: 6,
  fontSize: 12,
  overflow: 'auto',
};

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ display: 'flex', gap: 12, fontSize: 14 }}>
      <span style={{ width: 100, color: '#6b7280' }}>{label}</span>
      <span style={{ fontFamily: 'monospace' }}>{value}</span>
    </div>
  );
}

function prettyJSON(s: string): string {
  if (!s) return '(空)';
  try {
    return JSON.stringify(JSON.parse(s), null, 2);
  } catch {
    return s;
  }
}

function parseUserJourneyReport(raw: string): UserJourneyReport | null {
  if (!raw) return null;
  try {
    return JSON.parse(raw) as UserJourneyReport;
  } catch {
    return null;
  }
}

function fmtNum(v?: number): string {
  return v == null ? '-' : String(v);
}
