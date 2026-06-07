import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveDetail from '@/pages-new/LiveDetail';
import { AuthProvider } from '@/shared/auth';
import { liveStreamApi } from '@/shared/api';

jest.mock('@/shared/api', () => ({
  authApi: {
    getCurrentUser: jest.fn(),
  },
  liveStreamApi: {
    get: jest.fn(),
    adminGet: jest.fn(),
    start: jest.fn(),
    end: jest.fn(),
    adminEnd: jest.fn(),
    ban: jest.fn(),
  },
}));

function renderLiveDetailAs(role: number, liveStream: Record<string, unknown>) {
  localStorage.setItem('admin_auth_token', 'token');
  localStorage.setItem('admin_auth_user', JSON.stringify({
    id: role === 2 ? 1003 : 1002,
    name: role === 2 ? '系统管理员' : '商家用户',
    email: role === 2 ? 'admin@example.com' : 'merchant@example.com',
    role,
    created_at: '2026-06-05T00:00:00Z',
  }));

  (liveStreamApi.adminGet as jest.Mock).mockResolvedValue(liveStream);

  return render(
    <MemoryRouter initialEntries={['/live/detail?id=501']}>
      <AuthProvider>
        <LiveDetail />
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('LiveDetail start live', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
    jest.spyOn(window, 'alert').mockImplementation(() => undefined);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('lets merchants start the current live stream from detail page with explicit phase-one copy', async () => {
    jest.spyOn(window, 'confirm').mockReturnValue(true);
    (liveStreamApi.start as jest.Mock).mockResolvedValue({ success: true });

    renderLiveDetailAs(1, {
      id: 501,
      name: '商家直播间',
      streamer_id: 1002,
      streamer_name: '商家用户',
      status: 0,
      viewer_count: 0,
      auction_count: 2,
      created_at: '2026-06-05T00:00:00Z',
    });

    expect(await screen.findByRole('heading', { name: '直播间控制台' })).toBeInTheDocument();
    expect(liveStreamApi.adminGet).toHaveBeenCalledWith(501);
    expect(liveStreamApi.get).not.toHaveBeenCalled();
    expect(screen.getByText(/当前版本支持通过 PC 管理端发起直播状态/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /开始直播/ }));

    expect(window.confirm).toHaveBeenCalledWith(expect.stringContaining('确认开始直播'));
    await waitFor(() => expect(liveStreamApi.start).toHaveBeenCalledWith(501));
    expect(screen.getByRole('button', { name: /结束直播/ })).toBeInTheDocument();
  });

  it('lets merchants end a live stream from detail page', async () => {
    jest.spyOn(window, 'confirm').mockReturnValue(true);
    (liveStreamApi.end as jest.Mock).mockResolvedValue({ status: 2 });

    renderLiveDetailAs(1, {
      id: 501,
      name: '商家直播间',
      streamer_id: 1002,
      streamer_name: '商家用户',
      status: 1,
      viewer_count: 10,
      auction_count: 2,
      created_at: '2026-06-05T00:00:00Z',
    });

    expect(await screen.findByRole('heading', { name: '直播间控制台' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /结束直播/ }));

    expect(window.confirm).toHaveBeenCalledWith(expect.stringContaining('确认结束当前直播'));
    await waitFor(() => expect(liveStreamApi.end).toHaveBeenCalledWith(501));
    expect(screen.getByRole('button', { name: /已结束/ })).toBeDisabled();
  });

  it('does not show merchant start action for admins', async () => {
    renderLiveDetailAs(2, {
      id: 501,
      name: '平台巡检直播间',
      streamer_id: 1002,
      streamer_name: '商家用户',
      status: 0,
      viewer_count: 0,
      auction_count: 2,
      created_at: '2026-06-05T00:00:00Z',
    });

    expect(await screen.findByRole('heading', { name: '直播间控制台' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /开始直播/ })).not.toBeInTheDocument();
  });

  it('does not allow merchants to start banned live streams', async () => {
    renderLiveDetailAs(1, {
      id: 501,
      name: '被封禁直播间',
      streamer_id: 1002,
      streamer_name: '商家用户',
      status: 3,
      viewer_count: 0,
      auction_count: 0,
      created_at: '2026-06-05T00:00:00Z',
    });

    expect(await screen.findByRole('heading', { name: '直播间控制台' })).toBeInTheDocument();
    expect(screen.getAllByText('已封禁').length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: /已封禁/ })).toBeDisabled();

    fireEvent.click(screen.getByRole('button', { name: /已封禁/ }));

    expect(liveStreamApi.start).not.toHaveBeenCalled();
  });

  it('lets admins view banned live streams without showing merchant start action', async () => {
    renderLiveDetailAs(2, {
      id: 501,
      name: '平台封禁直播间',
      streamer_id: 1002,
      streamer_name: '商家用户',
      status: 3,
      viewer_count: 0,
      auction_count: 0,
      created_at: '2026-06-05T00:00:00Z',
    });

    expect(await screen.findByRole('heading', { name: '直播间控制台' })).toBeInTheDocument();
    expect(screen.getByText('已封禁')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /开始直播|已封禁/ })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /封禁直播间/ })).toBeInTheDocument();
  });
});
