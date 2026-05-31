import { ReactNode } from 'react';
import { act, render, screen } from '@testing-library/react';
import { AuthProvider, useAuth } from '../authContext';
import { authService } from '../../services/auth';

jest.mock('../../services/auth', () => ({
  authService: {
    getToken: jest.fn(),
    getCurrentUser: jest.fn(),
    login: jest.fn(),
    logout: jest.fn(),
    isAdmin: jest.fn(),
    isMerchant: jest.fn(),
  },
}));

const mockedAuthService = authService as jest.Mocked<typeof authService>;

function LoginProbe({ children }: { children?: ReactNode }) {
  const { login } = useAuth();
  return (
    <>
      <button type="button" onClick={() => login({ phone: '13800138000', password: 'secret123' })}>
        login
      </button>
      {children}
    </>
  );
}

describe('AuthProvider', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
    mockedAuthService.getToken.mockReturnValue(null);
    mockedAuthService.getCurrentUser.mockReturnValue(null);
    mockedAuthService.login.mockResolvedValue({
      token: 'token-1',
      user: { id: 1, email: 'buyer@example.com', name: '测试用户', role: 0 },
    });
  });

  it('does not write local live reminder marker after login', async () => {
    render(
      <AuthProvider>
        <LoginProbe />
      </AuthProvider>,
    );

    await act(async () => {
      screen.getByRole('button', { name: 'login' }).click();
    });

    expect(localStorage.getItem('pending_live_reminder')).toBeNull();
  });
});
