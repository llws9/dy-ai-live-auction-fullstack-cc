import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import MobileContainer from '../../components/MobileShell/MobileContainer';
import BottomNav from '../../components/MobileShell/BottomNav';
import { ThemeProvider } from '../../store/themeContext';

describe('MobileShell', () => {
  afterEach(() => {
    jest.restoreAllMocks();
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('renders children inside the mobile container without startup demo timers', () => {
    const setTimeoutSpy = jest.spyOn(window, 'setTimeout');

    render(
      <MemoryRouter>
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
  });

  it('shows retained bottom navigation entries and active route state', () => {
    render(
      <MemoryRouter initialEntries={['/profile']}>
        <BottomNav />
      </MemoryRouter>,
    );

    expect(screen.getByRole('link', { name: /首页/ })).toHaveAttribute('href', '/');
    expect(screen.getByRole('link', { name: /直播间/ })).toHaveAttribute('href', '/live');
    expect(screen.getByRole('link', { name: /我的/ })).toHaveAttribute('href', '/profile');
    expect(screen.getByRole('link', { name: /我的/ })).toHaveAttribute('aria-current', 'page');
  });

  it('shows unread total badge on profile nav item', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <BottomNav />
      </MemoryRouter>,
    );

    expect(screen.getByLabelText('3 条待处理提醒')).toHaveTextContent('3');
  });

  it.each(['/detail', '/result', '/notifications', '/following', '/history', '/login'])(
    'hides bottom navigation on %s',
    (path) => {
      render(
        <MemoryRouter initialEntries={[path]}>
          <BottomNav />
        </MemoryRouter>,
      );

      expect(screen.queryByRole('navigation', { name: '底部导航' })).not.toBeInTheDocument();
    },
  );

  it('opens live reminder once when pending login marker exists', () => {
    localStorage.setItem('pending_live_reminder', '1');

    render(
      <MemoryRouter>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('直播开播提醒')).toBeInTheDocument();
    expect(localStorage.getItem('pending_live_reminder')).toBeNull();
  });

  it('does not open live reminder without pending login marker', () => {
    localStorage.removeItem('pending_live_reminder');

    render(
      <MemoryRouter>
        <ThemeProvider>
          <MobileContainer>
            <main>页面内容</main>
          </MobileContainer>
        </ThemeProvider>
      </MemoryRouter>,
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
