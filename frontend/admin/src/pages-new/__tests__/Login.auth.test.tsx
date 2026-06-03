import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Login from '../Login';
import { AuthProvider, RequireAuth } from '@/shared/auth';
import { authApi } from '@/shared/api/auth';

jest.mock('@/shared/api/auth', () => ({
  authApi: {
    login: jest.fn(),
  },
}));

const mockedAuthApi = authApi as jest.Mocked<typeof authApi>;

function renderLoginFlow() {
  return render(
    <MemoryRouter initialEntries={['/admin-login']}>
      <AuthProvider>
        <Routes>
          <Route path="/admin-login" element={<Login />} />
          <Route
            path="/dashboard"
            element={
              <RequireAuth>
                <div>后台首页</div>
              </RequireAuth>
            }
          />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('Login auth flow', () => {
  beforeEach(() => {
    localStorage.clear();
    mockedAuthApi.login.mockReset();
  });

  it('updates auth context and enters dashboard after a successful login', async () => {
    mockedAuthApi.login.mockResolvedValue({
      token: 'admin-token',
      user: {
        id: 1003,
        name: '本地管理员',
        phone: '13900000003',
        role: 2,
        status: 1,
      },
    } as any);

    renderLoginFlow();

    await userEvent.click(screen.getByRole('button', { name: '手机登录' }));
    await userEvent.type(screen.getByPlaceholderText('手机号码'), '13900000003');
    await userEvent.type(screen.getByPlaceholderText('密码'), 'Demo@123456');
    await userEvent.click(screen.getByRole('button', { name: /立即登录/ }));

    await waitFor(() => {
      expect(screen.getByText('后台首页')).toBeInTheDocument();
    });
    expect(localStorage.getItem('admin_auth_token')).toBe('admin-token');
  });
});
