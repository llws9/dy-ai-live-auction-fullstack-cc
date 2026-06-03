import { inputStyle } from './styles';

export function NumField({
  label,
  value,
  min,
  step,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  step?: number;
  onChange: (v: number) => void;
}) {
  return (
    <label style={{ display: 'flex', flexDirection: 'column', fontSize: 13 }}>
      <span style={{ color: '#6b7280', marginBottom: 4 }}>{label}</span>
      <input
        type="number"
        value={value}
        min={min}
        step={step ?? 1}
        onChange={(e) => onChange(Number(e.target.value) || min)}
        style={inputStyle}
      />
    </label>
  );
}

export function TextField({
  label,
  value,
  onChange,
  type = 'text',
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
}) {
  return (
    <label style={{ display: 'flex', flexDirection: 'column', fontSize: 13 }}>
      <span style={{ color: '#6b7280', marginBottom: 4 }}>{label}</span>
      <input type={type} value={value} onChange={(e) => onChange(e.target.value)} style={inputStyle} />
    </label>
  );
}
