import { useEffect, useMemo, useState } from 'react';
import styles from './Home.module.css';

export interface PriceRange {
  min?: number;
  max?: number;
}

interface PricePreset {
  label: string;
  value: PriceRange;
}

interface PriceFilterSheetProps {
  open: boolean;
  value: PriceRange;
  onClose: () => void;
  onConfirm: (value: PriceRange) => void;
}

const PRICE_PRESETS: PricePreset[] = [
  { label: '不限', value: {} },
  { label: '0 - 1000', value: { min: 0, max: 1000 } },
  { label: '1000 - 5000', value: { min: 1000, max: 5000 } },
  { label: '5000 以上', value: { min: 5000 } },
];

const rangeToDraft = (range: PriceRange) => ({
  min: range.min === undefined ? '' : String(range.min),
  max: range.max === undefined ? '' : String(range.max),
});

const parsePriceInput = (raw: string) => {
  const trimmed = raw.trim();
  if (trimmed === '') return { value: undefined };

  const value = Number(trimmed);
  if (!Number.isFinite(value)) {
    return { error: '请输入有效数字' };
  }
  if (value < 0) {
    return { error: '价格不能为负数' };
  }

  return { value };
};

const isSameRange = (a: PriceRange, b: PriceRange) => a.min === b.min && a.max === b.max;

const PriceFilterSheet = ({ open, value, onClose, onConfirm }: PriceFilterSheetProps) => {
  const [draft, setDraft] = useState(() => rangeToDraft(value));

  useEffect(() => {
    if (open) {
      setDraft(rangeToDraft(value));
    }
  }, [open, value.min, value.max]);

  const validation = useMemo(() => {
    const min = parsePriceInput(draft.min);
    const max = parsePriceInput(draft.max);

    if (min.error) return { error: min.error };
    if (max.error) return { error: max.error };
    if (min.value !== undefined && max.value !== undefined && min.value > max.value) {
      return { error: '最低价不能高于最高价' };
    }

    return { value: { min: min.value, max: max.value } };
  }, [draft]);

  if (!open) return null;

  const selectedRange = validation.value;
  const errorText = validation.error;

  const handlePresetClick = (preset: PricePreset) => {
    setDraft(rangeToDraft(preset.value));
  };

  const handleConfirm = () => {
    if (!selectedRange) return;
    onConfirm(selectedRange);
  };

  return (
    <div className={styles.sheetOverlay} role="presentation" onClick={onClose}>
      <section
        className={styles.sheet}
        role="dialog"
        aria-modal="true"
        aria-labelledby="price-filter-sheet-title"
        onClick={(event) => event.stopPropagation()}
      >
        <h2 className={styles.sheetTitle} id="price-filter-sheet-title">
          价格区间
        </h2>

        <div className={styles.sheetPresets} aria-label="价格预设">
          {PRICE_PRESETS.map((preset) => (
            <button
              key={preset.label}
              type="button"
              className={`${styles.sheetPreset} ${
                selectedRange && isSameRange(selectedRange, preset.value) ? styles.filterPillActive : ''
              }`}
              onClick={() => handlePresetClick(preset)}
            >
              {preset.label}
            </button>
          ))}
        </div>

        <div className={styles.sheetCustom}>
          <input
            className={styles.sheetInput}
            inputMode="decimal"
            placeholder="最低价"
            aria-label="最低价"
            value={draft.min}
            onChange={(event) => setDraft((current) => ({ ...current, min: event.target.value }))}
          />
          <span className={styles.sheetDash}>-</span>
          <input
            className={styles.sheetInput}
            inputMode="decimal"
            placeholder="最高价"
            aria-label="最高价"
            value={draft.max}
            onChange={(event) => setDraft((current) => ({ ...current, max: event.target.value }))}
          />
        </div>

        {errorText && <p className={styles.sheetError}>{errorText}</p>}

        <button
          className={styles.sheetConfirm}
          type="button"
          disabled={Boolean(errorText)}
          onClick={handleConfirm}
        >
          确定
        </button>
      </section>
    </div>
  );
};

export default PriceFilterSheet;
