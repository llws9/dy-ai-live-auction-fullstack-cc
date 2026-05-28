// components/ErrorBoundary/index.tsx

import React, { Component, ErrorInfo, ReactNode } from 'react';
import { logError } from '../../utils/errorMessages';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
    errorInfo: null,
  };

  public static getDerivedStateFromError(error: Error): State {
    // 更新 state 以便下一次渲染能够显示降级后的 UI
    return { hasError: true, error, errorInfo: null };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // 记录错误日志
    logError(error, `React ErrorBoundary: ${errorInfo.componentStack}`);

    this.setState({
      error,
      errorInfo,
    });

    // 开发环境下打印错误
    if (import.meta.env.DEV) {
      console.error('ErrorBoundary caught an error:', error, errorInfo);
    }
  }

  private handleReload = () => {
    window.location.reload();
  };

  private handleGoHome = () => {
    window.location.href = '/';
  };

  public render() {
    if (this.state.hasError) {
      // 如果提供了自定义降级 UI，则使用它
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // 默认的错误提示 UI（移动端适配）
      return (
        <div style={styles.container}>
          <div style={styles.content}>
            <div style={styles.iconContainer}>
              <span style={styles.icon}>❌</span>
            </div>
            <h1 style={styles.title}>页面出错了</h1>
            <p style={styles.message}>
              很抱歉，页面遇到了一些问题
            </p>
            {import.meta.env.DEV && this.state.error && (
              <details style={styles.details}>
                <summary style={styles.summary}>查看错误详情</summary>
                <pre style={styles.errorText}>{this.state.error.toString()}</pre>
              </details>
            )}
            <div style={styles.actions}>
              <button style={styles.primaryButton} onClick={this.handleReload}>
                刷新页面
              </button>
              <button style={styles.secondaryButton} onClick={this.handleGoHome}>
                返回首页
              </button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    minHeight: '100vh',
    padding: '16px',
    backgroundColor: '#f7f8fa',
  },
  content: {
    width: '100%',
    maxWidth: '320px',
    padding: '32px 24px',
    backgroundColor: 'white',
    borderRadius: '16px',
    boxShadow: '0 2px 12px rgba(100, 101, 102, 0.08)',
    textAlign: 'center',
  },
  iconContainer: {
    width: '64px',
    height: '64px',
    margin: '0 auto 20px',
    borderRadius: '50%',
    backgroundColor: '#fff1f0',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  icon: {
    fontSize: '32px',
  },
  title: {
    fontSize: '18px',
    fontWeight: 600,
    color: '#323233',
    marginBottom: '12px',
    margin: '0 0 12px 0',
  },
  message: {
    fontSize: '14px',
    color: '#969799',
    marginBottom: '24px',
    lineHeight: 1.6,
    margin: '0 0 24px 0',
  },
  details: {
    marginBottom: '16px',
    padding: '12px',
    backgroundColor: '#f7f8fa',
    borderRadius: '8px',
    textAlign: 'left',
  },
  summary: {
    cursor: 'pointer',
    fontWeight: 500,
    marginBottom: '8px',
    color: '#969799',
    fontSize: '12px',
  },
  errorText: {
    fontSize: '11px',
    color: '#ee0a24',
    overflow: 'auto',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
    margin: '8px 0 0 0',
  },
  actions: {
    display: 'flex',
    flexDirection: 'column',
    gap: '12px',
  },
  primaryButton: {
    width: '100%',
    padding: '14px',
    fontSize: '16px',
    fontWeight: 500,
    color: 'white',
    backgroundColor: '#1989fa',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
    transition: 'opacity 0.2s',
  },
  secondaryButton: {
    width: '100%',
    padding: '14px',
    fontSize: '16px',
    fontWeight: 500,
    color: '#1989fa',
    backgroundColor: 'white',
    border: '1px solid #1989fa',
    borderRadius: '8px',
    cursor: 'pointer',
    transition: 'opacity 0.2s',
  },
};

export default ErrorBoundary;
