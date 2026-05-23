import React, { useEffect, useState } from 'react';
import styles from './Toast.module.css';

type ToastType = 'info' | 'success' | 'warning' | 'error';

interface ToastProps {
  /** 提示内容 */
  message: string;
  /** 提示类型 */
  type?: ToastType;
  /** 显示时长 (ms) */
  duration?: number;
  /** 关闭回调 */
  onClose?: () => void;
  /** 是否显示 */
  visible?: boolean;
}

/**
 * 消息提示组件
 */
export const Toast: React.FC<ToastProps> = ({
  message,
  type = 'info',
  duration = 3000,
  onClose,
  visible = true,
}) => {
  const [isVisible, setIsVisible] = useState(visible);

  useEffect(() => {
    setIsVisible(visible);
    if (visible && duration > 0) {
      const timer = setTimeout(() => {
        setIsVisible(false);
        onClose?.();
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [visible, duration, onClose]);

  if (!isVisible) return null;

  return (
    <div className={`${styles.toast} ${styles[type]}`} role="alert">
      <span className={styles.icon}>
        {type === 'success' && '✓'}
        {type === 'error' && '✕'}
        {type === 'warning' && '⚠'}
        {type === 'info' && 'ℹ'}
      </span>
      <span className={styles.message}>{message}</span>
    </div>
  );
};

export default Toast;
