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

      // 默认的错误提示 UI
      return (
        <div style={styles.container}>
          <div style={styles.content}>
            <div style={styles.icon}>❌</div>
            <h1 style={styles.title}>页面出错了</h1>
            <p style={styles.message}>
              很抱歉，页面遇到了一些问题。请尝试刷新页面或返回首页。
            </p>
            {import.meta.env.DEV && this.state.error && (
              <details style={styles.details}>
                <summary style={styles.summary}>错误详情（仅开发环境可见）</summary>
                <pre style={styles.errorText}>{this.state.error.toString()}</pre>
                {this.state.errorInfo && (
                  <pre style={styles.stackTrace}>{this.state.errorInfo.componentStack}</pre>
                )}
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
    padding: '20px',
    backgroundColor: '#f5f5f5',
  },
  content: {
    maxWidth: '600px',
    padding: '40px',
    backgroundColor: 'white',
    borderRadius: '12px',
    boxShadow: '0 2px 12px rgba(0, 0, 0, 0.1)',
    textAlign: 'center',
  },
  icon: {
    fontSize: '64px',
    marginBottom: '20px',
  },
  title: {
    fontSize: '24px',
    fontWeight: 'bold',
    color: '#333',
    marginBottom: '16px',
  },
  message: {
    fontSize: '16px',
    color: '#666',
    marginBottom: '24px',
    lineHeight: 1.6,
  },
  details: {
    marginBottom: '24px',
    padding: '16px',
    backgroundColor: '#f8f8f8',
    borderRadius: '8px',
    textAlign: 'left',
  },
  summary: {
    cursor: 'pointer',
    fontWeight: 'bold',
    marginBottom: '12px',
    color: '#666',
  },
  errorText: {
    fontSize: '12px',
    color: '#d32f2f',
    overflow: 'auto',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
  },
  stackTrace: {
    fontSize: '12px',
    color: '#666',
    marginTop: '12px',
    overflow: 'auto',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
  },
  actions: {
    display: 'flex',
    gap: '12px',
    justifyContent: 'center',
  },
  primaryButton: {
    padding: '12px 24px',
    fontSize: '16px',
    color: 'white',
    backgroundColor: '#1890ff',
    border: 'none',
    borderRadius: '6px',
    cursor: 'pointer',
    transition: 'background-color 0.3s',
  },
  secondaryButton: {
    padding: '12px 24px',
    fontSize: '16px',
    color: '#1890ff',
    backgroundColor: 'white',
    border: '1px solid #1890ff',
    borderRadius: '6px',
    cursor: 'pointer',
    transition: 'all 0.3s',
  },
};

export default ErrorBoundary;
