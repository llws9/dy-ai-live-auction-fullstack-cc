import React from 'react';
import styles from './Button.module.css';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** 按钮变体 */
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost';
  /** 按钮尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 加载状态 */
  loading?: boolean;
  /** 全宽按钮 */
  fullWidth?: boolean;
  /** 子元素 */
  children: React.ReactNode;
}

/**
 * 可复用的按钮组件
 * 支持多种变体、尺寸和加载状态
 */
export const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  size = 'md',
  loading = false,
  fullWidth = false,
  children,
  className,
  disabled,
  ...props
}) => {
  const buttonClasses = [
    styles.button,
    styles[variant],
    styles[size],
    fullWidth ? styles.fullWidth : '',
    loading ? styles.loading : '',
    className || '',
  ].filter(Boolean).join(' ');

  return (
    <button
      className={buttonClasses}
      disabled={disabled || loading}
      {...props}
    >
      {loading && <span className={styles.spinner} aria-hidden="true" />}
      <span className={loading ? styles.textHidden : ''}>{children}</span>
    </button>
  );
};

export default Button;
