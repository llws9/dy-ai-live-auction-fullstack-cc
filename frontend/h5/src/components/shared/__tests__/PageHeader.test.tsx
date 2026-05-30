import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import PageHeader from '../PageHeader';
import { ThemeProvider } from '../../../store/themeContext';

const wrap = (ui: React.ReactNode, route = '/') =>
  render(
    <MemoryRouter initialEntries={[route]}>
      <ThemeProvider>{ui}</ThemeProvider>
    </MemoryRouter>,
  );

const cleanupTheme = () => {
  localStorage.clear();
  document.documentElement.removeAttribute('data-theme');
};

describe('PageHeader', () => {
  afterEach(cleanupTheme);

  it('renders title without back button when back is omitted', () => {
    wrap(<PageHeader classes={{ header: 'h' }} title="奢华竞拍" />);

    expect(screen.getByRole('heading', { level: 1, name: '奢华竞拍' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '返回' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '返回' })).not.toBeInTheDocument();
  });

  it('renders Link back when back.to is provided', () => {
    wrap(
      <PageHeader
        classes={{ header: 'h', backButton: 'b' }}
        back={{ to: '/' }}
        title="详情"
      />,
    );

    expect(screen.getByRole('link', { name: '返回' })).toHaveAttribute('href', '/');
  });

  it('renders button back with onClick', () => {
    const onClick = jest.fn();
    wrap(
      <PageHeader
        classes={{ header: 'h', backButton: 'b' }}
        back={{ onClick }}
        title="结果"
      />,
    );

    screen.getByRole('button', { name: '返回' }).click();
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('renders eyebrow + title block when eyebrow provided', () => {
    wrap(
      <PageHeader
        classes={{ header: 'h', eyebrow: 'e' }}
        eyebrow="FOLLOWING"
        title="关注"
      />,
    );

    expect(screen.getByText('FOLLOWING')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: '关注' })).toBeInTheDocument();
  });

  it('appends ThemeToggle by default at the end of actions', () => {
    wrap(
      <PageHeader
        classes={{ header: 'h' }}
        title="主页"
        actions={<span data-testid="custom-action">自定义</span>}
      />,
    );

    expect(screen.getByTestId('custom-action')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /切换到/ })).toBeInTheDocument();
  });

  it('hides ThemeToggle when hideThemeToggle is true', () => {
    wrap(
      <PageHeader classes={{ header: 'h' }} title="登录" hideThemeToggle />,
    );

    expect(screen.queryByRole('button', { name: /切换到/ })).not.toBeInTheDocument();
  });
});
