interface Props {
  value: number; // 0~100
  label?: string;
}

export default function ProgressBar({ value, label }: Props) {
  const v = Math.max(0, Math.min(100, value));
  return (
    <div style={{ width: '100%' }}>
      {label && (
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6, fontSize: 13 }}>
          <span>{label}</span>
          <span>{v.toFixed(0)}%</span>
        </div>
      )}
      <div
        style={{
          width: '100%',
          height: 10,
          background: '#e5e7eb',
          borderRadius: 6,
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            width: `${v}%`,
            height: '100%',
            background: 'var(--color-primary, #3b82f6)',
            transition: 'width 200ms ease',
          }}
        />
      </div>
    </div>
  );
}
