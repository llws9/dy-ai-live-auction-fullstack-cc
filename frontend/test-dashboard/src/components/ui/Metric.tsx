export function Metric({ label, value, ok, bad }: { label: string; value: string; ok?: boolean; bad?: boolean }) {
  const color = ok ? '#10b981' : bad ? '#ef4444' : ok === false ? '#ef4444' : '#1f2937';
  return (
    <div style={{ background: '#f8fafc', border: '1px solid #e5e7eb', borderRadius: 6, padding: '10px 12px' }}>
      <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 4 }}>{label}</div>
      <div style={{ fontSize: 18, fontFamily: 'monospace', fontWeight: 600, color }}>{value}</div>
    </div>
  );
}
