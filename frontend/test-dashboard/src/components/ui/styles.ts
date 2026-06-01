export const cardStyle: React.CSSProperties = {
  padding: 16,
  border: '1px solid #e5e7eb',
  borderRadius: 8,
  marginBottom: 16,
};

export const titleStyle: React.CSSProperties = {
  fontSize: 16,
  marginBottom: 12,
};

export const inputStyle: React.CSSProperties = {
  padding: '6px 10px',
  border: '1px solid #d1d5db',
  borderRadius: 6,
  fontSize: 14,
};

export const codeStyle: React.CSSProperties = {
  background: '#f1f5f9',
  padding: '1px 6px',
  borderRadius: 3,
  fontSize: 12,
};

export const primaryBtn = (disabled: boolean): React.CSSProperties => ({
  padding: '8px 16px',
  background: 'var(--color-primary, #3b82f6)',
  color: '#fff',
  border: 'none',
  borderRadius: 6,
  cursor: disabled ? 'not-allowed' : 'pointer',
  opacity: disabled ? 0.6 : 1,
});

export const secondaryBtn = (disabled: boolean): React.CSSProperties => ({
  padding: '8px 16px',
  background: '#fff',
  color: '#1f2937',
  border: '1px solid #d1d5db',
  borderRadius: 6,
  cursor: disabled ? 'not-allowed' : 'pointer',
  opacity: disabled ? 0.6 : 1,
});

export const caseChipStyle = (ok: boolean, active: boolean): React.CSSProperties => ({
  padding: '6px 12px',
  borderRadius: 16,
  border: `1px solid ${ok ? '#10b981' : '#ef4444'}`,
  background: active ? (ok ? '#10b981' : '#ef4444') : '#fff',
  color: active ? '#fff' : ok ? '#10b981' : '#ef4444',
  fontSize: 13,
  cursor: 'pointer',
});
