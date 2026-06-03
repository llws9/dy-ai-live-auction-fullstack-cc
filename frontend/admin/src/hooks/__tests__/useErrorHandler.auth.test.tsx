import React from 'react';
import { renderHook, act } from '@testing-library/react';
import { ToastProvider } from '@/components/Toast';
import { useErrorHandler } from '../useErrorHandler';

const navigateMock = jest.fn();

jest.mock('@/utils/errorMessages', () => ({
  getErrorMessage: (error: any) => ({
    message: error.message || '未授权',
    action: error.status === 401 ? 'redirect_login' : undefined,
  }),
  logError: jest.fn(),
}));

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => navigateMock,
}));

describe('useErrorHandler admin auth handling', () => {
  beforeEach(() => {
    localStorage.clear();
    navigateMock.mockClear();
  });

  it('clears admin auth storage and redirects to admin login on 401', () => {
    localStorage.setItem('admin_auth_token', 'expired-admin-token');
    localStorage.setItem('admin_auth_user', JSON.stringify({ id: 999 }));
    localStorage.setItem('token', 'legacy-token');
    localStorage.setItem('userInfo', JSON.stringify({ id: 1 }));

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <ToastProvider>{children}</ToastProvider>
    );
    const { result } = renderHook(() => useErrorHandler(), { wrapper });

    act(() => {
      result.current.handleError({ status: 401, message: '未授权' });
    });

    expect(localStorage.getItem('admin_auth_token')).toBeNull();
    expect(localStorage.getItem('admin_auth_user')).toBeNull();
    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('userInfo')).toBeNull();
    expect(navigateMock).toHaveBeenCalledWith('/admin-login', { replace: true });
  });
});
