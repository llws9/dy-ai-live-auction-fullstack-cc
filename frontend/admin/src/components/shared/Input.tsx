import React, { forwardRef, useState } from 'react';
import styles from './Input.module.css';

interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'> {
  /** 输入框标签 */
  label?: string;
  /** 错误信息 */
  error?: string;
  /** 成功状态 */
  success?: boolean;
  /** 输入框尺寸 */
  inputSize?: 'sm' | 'md' | 'lg';
  /** 全宽输入框 */
  fullWidth?: boolean;
  /** 显示清除按钮 */
  clearable?: boolean;
  /** 清除回调 */
  onClear?: () => void;
}

/**
 * 可复用的输入框组件
 * 支持验证状态、清除按钮和多种尺寸
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(({
  label,
  error,
  success,
  inputSize = 'md',
  fullWidth = false,
  clearable = false,
  onClear,
  className,
  value,
  ...props
}, ref) => {
  const [isFocused, setIsFocused] = useState(false);

  const containerClasses = [
    styles.container,
    fullWidth ? styles.fullWidth : '',
    className || '',
  ].filter(Boolean).join(' ');

  const inputWrapperClasses = [
    styles.inputWrapper,
    styles[inputSize],
    error ? styles.error : success ? styles.success : '',
    isFocused ? styles.focused : '',
    props.disabled ? styles.disabled : '',
  ].filter(Boolean).join(' ');

  const handleClear = () => {
    onClear?.();
  };

  return (
    <div className={containerClasses}>
      {label && (
        <label className={styles.label} htmlFor={props.id}>
          {label}
        </label>
      )}
      <div className={inputWrapperClasses}>
        <input
          ref={ref}
          className={styles.input}
          value={value}
          onFocus={(e) => {
            setIsFocused(true);
            props.onFocus?.(e);
          }}
          onBlur={(e) => {
            setIsFocused(false);
            props.onBlur?.(e);
          }}
          {...props}
        />
        {clearable && value && !props.disabled && (
          <button
            type="button"
            className={styles.clearButton}
            onClick={handleClear}
            aria-label="清除"
          >
            ✕
          </button>
        )}
      </div>
      {error && <span className={styles.errorText}>{error}</span>}
    </div>
  );
});

Input.displayName = 'Input';

export default Input;
