import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import MobileContainer from '../../components/MobileShell/MobileContainer';
import BottomNav from '../../components/MobileShell/BottomNav';

describe('MobileShell', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('renders children inside the mobile container without startup demo timers', () => {
    const setTimeoutSpy = jest.spyOn(window, 'setTimeout');

    render(
      <MemoryRouter>
        <MobileContainer>
          <main>页面内容</main>
        </MobileContainer>
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
});
