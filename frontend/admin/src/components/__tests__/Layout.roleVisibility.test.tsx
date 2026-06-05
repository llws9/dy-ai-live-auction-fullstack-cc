import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { Layout } from '@/components/Layout';
import { AuthProvider } from '@/shared/auth';

function renderLayoutWithUser(role: number, initialPath = '/dashboard') {
  localStorage.setItem('admin_auth_token', 'token');
  localStorage.setItem('admin_auth_user', JSON.stringify({
    id: role === 2 ? 1003 : 1002,
    name: role === 2 ? '系统管理员' : '商家用户',
    email: role === 2 ? 'admin@example.com' : 'merchant@example.com',
    role,
    created_at: '2026-06-05T00:00:00Z',
  }));

  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <AuthProvider>
        <Layout>
          <div>页面内容</div>
        </Layout>
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('Layout role visibility', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('hides platform-only and admin-forbidden operation entries for merchants', async () => {
    renderLayoutWithUser(1, '/live/list');

    expect(await screen.findByText('商家用户')).toBeInTheDocument();
    expect(screen.getByText('商家/主播')).toBeInTheDocument();

    expect(screen.getByRole('link', { name: '我的直播间' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '直播间列表' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '用户统计' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '角色管理' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '用户管理' })).not.toBeInTheDocument();
  });

  it('hides merchant-only operation entries for platform admins', async () => {
    renderLayoutWithUser(2, '/live/list');

    expect(await screen.findByText('系统管理员')).toBeInTheDocument();
    expect(screen.getByText('平台管理员')).toBeInTheDocument();

    expect(screen.getByRole('link', { name: '直播间列表' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '我的直播间' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '创建商品' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '规则模板' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '一口价上下架' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '创建直播间' })).not.toBeInTheDocument();
  });
});
