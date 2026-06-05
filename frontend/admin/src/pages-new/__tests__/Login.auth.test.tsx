import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Login from '../Login';
import { AuthProvider, RequireAuth, RequireRole, useAuth } from '@/shared/auth';
import { authApi } from '@/shared/api/auth';

jest.mock('@/shared/api/auth', () => ({
  authApi: {
    login: jest.fn(),
  },
}));

const mockedAuthApi = authApi as jest.Mocked<typeof authApi>;

function DashboardGreeting() {
  const { user } = useAuth();
  return <h1>欢迎，{user?.name || '管理员'}</h1>;
}

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
                <DashboardGreeting />
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
      expect(screen.getByText('欢迎，本地管理员')).toBeInTheDocument();
    });
    expect(localStorage.getItem('admin_auth_token')).toBe('admin-token');
  });

  it('repairs mojibake user names before rendering the dashboard greeting', async () => {
    mockedAuthApi.login.mockResolvedValue({
      token: 'admin-token',
      user: {
        id: 1003,
        name: 'ç³»ç»Ÿç®¡ç†å‘˜',
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
      expect(screen.getByText('欢迎，系统管理员')).toBeInTheDocument();
    });
    expect(JSON.parse(localStorage.getItem('admin_auth_user') || '{}').name).toBe('系统管理员');
  });

  it('redirects authenticated users away from role-forbidden pages', async () => {
    localStorage.setItem('admin_auth_token', 'admin-token');
    localStorage.setItem('admin_auth_user', JSON.stringify({
      id: 1003,
      name: '系统管理员',
      email: 'admin@example.com',
      role: 2,
      created_at: '2026-06-05T00:00:00Z',
    }));

    render(
      <MemoryRouter initialEntries={['/goods/create']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/goods/create"
              element={
                <RequireRole allowedRoles={[1]}>
                  <h1>创建商品</h1>
                </RequireRole>
              }
            />
            <Route path="/dashboard" element={<h1>经营总览</h1>} />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    expect(await screen.findByRole('heading', { name: '经营总览' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '创建商品' })).not.toBeInTheDocument();
  });
});
