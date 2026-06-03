// hooks/useErrorHandler.ts

import { useCallback } from 'react';
import { useToast } from '../components/Toast';
import { getErrorMessage, logError } from '../utils/errorMessages';
import { useNavigate } from 'react-router-dom';

interface UseErrorHandlerOptions {
  showErrorToast?: boolean;
  logErrors?: boolean;
  redirectToLogin?: boolean;
}

/**
 * 统一错误处理 Hook
 */
export function useErrorHandler(options: UseErrorHandlerOptions = {}) {
  const { showErrorToast = true, logErrors = true, redirectToLogin = true } = options;
  const { showToast } = useToast();
  const navigate = useNavigate();

  const handleError = useCallback((error: any, context?: string) => {
    const errorMsg = getErrorMessage(error);

    // 记录错误日志
    if (logErrors) {
      logError(error, context);
    }

    // 显示错误提示
    if (showErrorToast) {
      showToast(errorMsg.message, 'error');
    }

    // 401 错误自动跳转登录页
    if (redirectToLogin && (error.status === 401 || errorMsg.action === 'redirect_login')) {
      localStorage.removeItem('admin_auth_token');
      localStorage.removeItem('admin_auth_user');
      localStorage.removeItem('token');
      localStorage.removeItem('userInfo');
      navigate('/admin-login', { replace: true });
    }

    return errorMsg;
  }, [showErrorToast, logErrors, redirectToLogin, showToast, navigate]);

  /**
   * 包装异步函数，自动处理错误
   */
  const wrapAsync = useCallback(<T extends any[], R>(
    fn: (...args: T) => Promise<R>
  ) => {
    return async (...args: T): Promise<R | undefined> => {
      try {
        return await fn(...args);
      } catch (error) {
        handleError(error, fn.name);
        return undefined;
      }
    };
  }, [handleError]);

  return {
    handleError,
    wrapAsync,
  };
}

export default useErrorHandler;
