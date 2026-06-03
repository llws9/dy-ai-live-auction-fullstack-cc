import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import MobileContainer from '../../components/MobileShell/MobileContainer';
import BottomNav from '../../components/MobileShell/BottomNav';
import { notificationApi } from '../../services/notification';
import { useAuth } from '../../store/authContext';
import { ThemeProvider } from '../../store/themeContext';

jest.mock('../../services/notification', () => ({
  notificationApi: {
    getTouchpointSummary: jest.fn(),
    getPendingLiveReminder: jest.fn(),
  },
}));

jest.mock('../../store/authContext', () => ({
  useAuth: jest.fn(),
}));

const mockGetTouchpointSummary = notificationApi.getTouchpointSummary as jest.MockedFunction<
  typeof notificationApi.getTouchpointSummary
>;
const mockGetPendingLiveReminder = notificationApi.getPendingLiveReminder as jest.MockedFunction<
  typeof notificationApi.getPendingLiveReminder
>;
const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, resolve, reject };
}

const authenticatedAuthState = {
  isAuthenticated: true,
  user: { id: 1, email: 'buyer@example.com', name: '测试用户', role: 0 },
  token: 'token-1',
  loading: false,
  login: jest.fn(),
  setAuth: jest.fn(),
  logout: jest.fn(),
  isAdmin: jest.fn(() => false),
  isMerchant: jest.fn(() => false),
};

describe('MobileShell', () => {
  let authState = authenticatedAuthState;

  beforeEach(() => {
    authState = authenticatedAuthState;
    mockUseAuth.mockImplementation(() => authState);
    mockGetTouchpointSummary.mockResolvedValue({
      unreadTotal: 7,
      pendingPayment: 2,
      wonNotPaid: 1,
      outbid: 3,
      endingSoon: 1,
    });
    mockGetPendingLiveReminder.mockResolvedValue({ hasReminder: false, stream: null });
  });

  afterEach(() => {
    jest.restoreAllMocks();
    jest.clearAllMocks();
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('renders children inside the mobile container without startup demo timers', async () => {
    const setTimeoutSpy = jest.spyOn(window, 'setTimeout');

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    expect(screen.getByText('页面内容')).toBeInTheDocument();
    expect(screen.getByTestId('mobile-shell')).toBeInTheDocument();
    expect(setTimeoutSpy).not.toHaveBeenCalled();
    setTimeoutSpy.mockRestore();
    await waitFor(() => expect(mockGetPendingLiveReminder).toHaveBeenCalledTimes(1));
  });

  it('shows retained bottom navigation entries and active route state', async () => {
    render(
      <MemoryRouter initialEntries={['/profile']} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <BottomNav />
      </MemoryRouter>,
    );

    expect(screen.getByRole('link', { name: /首页/ })).toHaveAttribute('href', '/');
    expect(screen.getByRole('link', { name: /直播间/ })).toHaveAttribute('href', '/live');
    expect(screen.getByRole('link', { name: /我的/ })).toHaveAttribute('href', '/profile');
    expect(screen.getByRole('link', { name: /我的/ })).toHaveAttribute('aria-current', 'page');
    expect(await screen.findByLabelText('7 条待处理提醒')).toHaveTextContent('7');
  });

  it('uses a live-safe content layout on live routes so the room stops above the bottom navigation', async () => {
    render(
      <MemoryRouter initialEntries={['/live']} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <ThemeProvider>
          <MobileContainer>
            <main>直播间内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    expect(screen.getByText('直播间内容').parentElement).toHaveClass('contentLive');
    await waitFor(() => expect(mockGetPendingLiveReminder).toHaveBeenCalledTimes(1));
  });

  it('shows unread total badge on profile nav item from backend summary', async () => {
    render(
      <MemoryRouter initialEntries={['/']} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <BottomNav />
      </MemoryRouter>,
    );

    expect(await screen.findByLabelText('7 条待处理提醒')).toHaveTextContent('7');
    expect(mockGetTouchpointSummary).toHaveBeenCalledTimes(1);
  });

  it('refetches touchpoint summary for account changes and ignores stale responses', async () => {
    const firstRequest = createDeferred<{
      unreadTotal: number;
      pendingPayment: number;
      wonNotPaid: number;
      outbid: number;
      endingSoon: number;
    }>();
    mockGetTouchpointSummary
      .mockReturnValueOnce(firstRequest.promise)
      .mockResolvedValueOnce({
        unreadTotal: 4,
        pendingPayment: 1,
        wonNotPaid: 1,
        outbid: 1,
        endingSoon: 1,
      });

    const { rerender } = render(
      <MemoryRouter initialEntries={['/']} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <BottomNav />
      </MemoryRouter>,
    );

    await waitFor(() => expect(mockGetTouchpointSummary).toHaveBeenCalledTimes(1));

    authState = {
      ...authenticatedAuthState,
      user: { id: 2, email: 'buyer2@example.com', name: '新用户', role: 0 },
      token: 'token-2',
    };
    rerender(
      <MemoryRouter initialEntries={['/']} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <BottomNav />
      </MemoryRouter>,
    );

    await waitFor(() => expect(mockGetTouchpointSummary).toHaveBeenCalledTimes(2));
    expect(await screen.findByLabelText('4 条待处理提醒')).toHaveTextContent('4');

    firstRequest.resolve({
      unreadTotal: 9,
      pendingPayment: 9,
      wonNotPaid: 9,
      outbid: 9,
      endingSoon: 9,
    });

    await waitFor(() => expect(screen.queryByLabelText('9 条待处理提醒')).not.toBeInTheDocument());
    expect(screen.getByLabelText('4 条待处理提醒')).toHaveTextContent('4');
  });

  it.each(['/detail', '/result', '/notifications', '/following', '/history', '/login'])(
    'hides bottom navigation on %s',
    async (path) => {
      mockGetTouchpointSummary.mockRejectedValue(new Error('hidden nav should not render badges'));

      render(
        <MemoryRouter initialEntries={[path]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
          <BottomNav />
        </MemoryRouter>,
      );

      expect(screen.queryByRole('navigation', { name: '底部导航' })).not.toBeInTheDocument();
      await waitFor(() => expect(screen.queryByRole('navigation', { name: '底部导航' })).not.toBeInTheDocument());
    },
  );

  it('opens live reminder once when backend returns a pending stream', async () => {
    mockGetPendingLiveReminder.mockResolvedValue({
      hasReminder: true,
      stream: {
        id: 1,
        name: '云端珍藏直播间',
        avatarUrl: '',
        statusText: '正在直播',
        liveRoomId: 1,
        startedAt: 1717000000000,
      },
    });
    localStorage.setItem('pending_live_reminder', '1');

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('直播开播提醒')).toBeInTheDocument();
    expect(mockGetPendingLiveReminder).toHaveBeenCalledTimes(1);
  });

  it('refetches pending reminder for account changes and ignores stale responses', async () => {
    const firstRequest = createDeferred<{
      hasReminder: boolean;
      stream: {
        id: string | number;
        name: string;
        avatarUrl: string;
        statusText?: string;
      } | null;
    }>();
    mockGetPendingLiveReminder.mockReturnValueOnce(firstRequest.promise).mockResolvedValueOnce({
      hasReminder: false,
      stream: null,
    });

    const { rerender } = render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    await waitFor(() => expect(mockGetPendingLiveReminder).toHaveBeenCalledTimes(1));

    authState = {
      ...authenticatedAuthState,
      user: { id: 2, email: 'buyer2@example.com', name: '新用户', role: 0 },
      token: 'token-2',
    };
    rerender(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    await waitFor(() => expect(mockGetPendingLiveReminder).toHaveBeenCalledTimes(2));

    firstRequest.resolve({
      hasReminder: true,
      stream: { id: 1, name: '旧账号直播间', avatarUrl: '', statusText: '正在直播' },
    });

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(screen.queryByText('旧账号直播间')).not.toBeInTheDocument();
  });

  it('does not request pending reminder before login is confirmed', async () => {
    mockUseAuth.mockReturnValue({
      isAuthenticated: false,
      user: null,
      token: null,
      loading: false,
      login: jest.fn(),
      setAuth: jest.fn(),
      logout: jest.fn(),
      isAdmin: jest.fn(() => false),
      isMerchant: jest.fn(() => false),
    });
    mockGetPendingLiveReminder.mockResolvedValue({
      hasReminder: true,
      stream: { id: 1, name: '直播间', avatarUrl: '' },
    });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(mockGetTouchpointSummary).not.toHaveBeenCalled();
    expect(mockGetPendingLiveReminder).not.toHaveBeenCalled();
  });

  it('does not open live reminder when backend returns empty', async () => {
    mockGetPendingLiveReminder.mockResolvedValue({ hasReminder: false, stream: null });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    await waitFor(() => expect(mockGetPendingLiveReminder).toHaveBeenCalled());
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('does not fall back to mock reminder when backend fails', async () => {
    localStorage.setItem('pending_live_reminder', '1');
    mockGetPendingLiveReminder.mockRejectedValue(new Error('network'));

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    await waitFor(() => expect(mockGetPendingLiveReminder).toHaveBeenCalled());
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(localStorage.getItem('pending_live_reminder')).toBeNull();
  });
});
