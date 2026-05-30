import { createContext, useCallback, useContext, useMemo, useState, ReactNode } from 'react';
import styles from './Toast.module.css';

export type ToastType = 'success' | 'warning' | 'danger' | 'error' | 'info' | 'loading';

export interface ToastConfig {
  type?: ToastType;
  title?: string;
  message: string;
  duration?: number;
  actionText?: string;
  onAction?: () => void;
}

interface ToastMessage extends ToastConfig {
  id: number;
  type: ToastType;
  duration: number;
}

interface ToastContextType {
  showToast: {
    (message: string, type?: ToastType, duration?: number): number;
    (config: ToastConfig): number;
  };
  showLoading: (message: string) => () => void;
}

const ToastContext = createContext<ToastContextType | null>(null);
let toastId = 0;
const MAX_VISIBLE_TOASTS = 3;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const showToast = useCallback((input: string | ToastConfig, type: ToastType = 'info', duration = 3000) => {
    const config: ToastConfig = typeof input === 'string' ? { message: input, type, duration } : input;
    const id = ++toastId;
    const toast: ToastMessage = {
      id,
      type: config.type || 'info',
      title: config.title,
      message: config.message,
      duration: config.duration ?? duration,
      actionText: config.actionText,
      onAction: config.onAction,
    };

    setToasts((prev) => [...prev, toast]);

    if (toast.duration > 0 && toast.type !== 'loading') {
      window.setTimeout(() => removeToast(id), toast.duration);
    }

    return id;
  }, [removeToast]);

  const showLoading = useCallback((message: string) => {
    const id = showToast({ message, type: 'loading', duration: 0 });
    return () => removeToast(id);
  }, [removeToast, showToast]);

  const visibleToasts = useMemo(() => toasts.slice(0, MAX_VISIBLE_TOASTS), [toasts]);

  return (
    <ToastContext.Provider value={{ showToast, showLoading }}>
      {children}
      <div className={styles.container} aria-live="polite">
        {visibleToasts.map((toast) => (
          <div key={toast.id} className={`${styles.toast} ${styles[toast.type]}`} role="status">
            <div className={styles.icon} aria-hidden="true">
              {getIcon(toast.type)}
            </div>
            <div className={styles.body}>
              {toast.title && <strong className={styles.title}>{toast.title}</strong>}
              <span className={styles.message}>{toast.message}</span>
            </div>
            {toast.actionText && (
              <button
                type="button"
                className={styles.action}
                onClick={() => {
                  toast.onAction?.();
                  removeToast(toast.id);
                }}
              >
                {toast.actionText}
              </button>
            )}
            <button
              type="button"
              className={styles.close}
              aria-label="关闭提示"
              onClick={() => removeToast(toast.id)}
            >
              <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" strokeWidth="2" fill="none" strokeLinecap="round" strokeLinejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}

function getIcon(type: ToastType): ReactNode {
  switch (type) {
    case 'success':
      return (
        <svg viewBox="0 0 24 24" width="12" height="12" stroke="currentColor" strokeWidth="3" fill="none" strokeLinecap="round" strokeLinejoin="round">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
      );
    case 'error':
    case 'danger':
      return (
        <svg viewBox="0 0 24 24" width="12" height="12" stroke="currentColor" strokeWidth="3" fill="none" strokeLinecap="round" strokeLinejoin="round">
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
      );
    case 'warning':
      return (
        <svg viewBox="0 0 24 24" width="12" height="12" stroke="currentColor" strokeWidth="3" fill="none" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <polyline points="12 6 12 12 16 14"></polyline>
        </svg>
      );
    case 'loading':
      return <div className={styles.spinner} />;
    case 'info':
    default:
      return (
        <svg viewBox="0 0 24 24" width="12" height="12" stroke="currentColor" strokeWidth="3" fill="none" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="16" x2="12" y2="12"></line>
          <line x1="12" y1="8" x2="12.01" y2="8"></line>
        </svg>
      );
  }
}

export default ToastProvider;
