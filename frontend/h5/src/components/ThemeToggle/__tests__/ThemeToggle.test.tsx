import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider } from '../../../store/themeContext';
import ThemeToggle from '../ThemeToggle';

function renderWithProvider() {
  return render(
    <ThemeProvider>
      <ThemeToggle />
    </ThemeProvider>,
  );
}

describe('ThemeToggle', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('默认 dark 时按钮 aria-label 提示切换到浅色模式', () => {
    renderWithProvider();
    const btn = screen.getByRole('button', { name: /切换到浅色模式/ });
    expect(btn).toBeInTheDocument();
  });

  it('点击按钮可在 dark / light 间切换 DOM 与 localStorage', async () => {
    const user = userEvent.setup();
    renderWithProvider();

    const btn = screen.getByRole('button', { name: /切换到浅色模式/ });
    await user.click(btn);

    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    expect(localStorage.getItem('h5.theme')).toBe('light');

    const btnAfter = screen.getByRole('button', { name: /切换到深色模式/ });
    await user.click(btnAfter);

    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
    expect(localStorage.getItem('h5.theme')).toBe('dark');
  });

  it('a11y：按钮拥有动态 aria-label 且不冗余使用 aria-pressed', async () => {
    const user = userEvent.setup();
    renderWithProvider();

    const btn = screen.getByRole('button');
    // 状态信息全部由 aria-label 翻转承载，故不应再叠加 aria-pressed
    expect(btn).not.toHaveAttribute('aria-pressed');
    expect(btn).toHaveAttribute('aria-label', '切换到浅色模式');

    await user.click(btn);

    expect(btn).toHaveAttribute('aria-label', '切换到深色模式');
    expect(btn).not.toHaveAttribute('aria-pressed');
  });
});
