import { readFileSync } from 'fs';
import { join } from 'path';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveReminderModal from '../index';

jest.mock('../../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
}));

jest.mock('../../../utils/businessEvent', () => ({
  trackBusinessEvent: jest.fn(),
}));

describe('LiveReminderModal', () => {
  it('does not render an empty image src when stream avatar is missing', () => {
    render(
      <MemoryRouter>
        <LiveReminderModal
          isOpen
          onClose={() => {}}
          stream={{ id: 6, name: 'Demo 商家直播间', avatarUrl: '', statusText: '正在直播' }}
        />
      </MemoryRouter>,
    );

    expect(screen.queryByRole('img', { name: 'Demo 商家直播间' })).not.toBeInTheDocument();
    expect(screen.getByText('D')).toBeInTheDocument();
  });

  it('renders the v1 decorative svg camera icon without emoji text', () => {
    render(
      <MemoryRouter>
        <LiveReminderModal
          isOpen
          onClose={() => {}}
          stream={{ id: 6, name: 'Demo 商家直播间', avatarUrl: '', statusText: '正在直播' }}
        />
      </MemoryRouter>,
    );

    const dialog = screen.getByRole('dialog', { name: '直播开播提醒' });
    expect(dialog).toBeInTheDocument();
    expect(screen.queryByText('🎥')).not.toBeInTheDocument();
    expect(screen.getByTestId('live-reminder-camera-icon')).toHaveAttribute('aria-hidden', 'true');
    expect(screen.getByText('正在直播')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '立即前往' })).toBeInTheDocument();
  });

  it('keeps v1 visual styles bound to semantic theme tokens', () => {
    const css = readFileSync(
      join(__dirname, '../LiveReminderModal.module.css'),
      'utf8',
    );

    expect(css).toContain('border-radius: 24px');
    expect(css).toContain('background: var(--bg-surface)');
    expect(css).toContain('background: var(--item-subtle-bg)');
    expect(css).toContain('background: var(--gradient-primary)');
    expect(css).toContain('color: var(--text-secondary');
    expect(css).toContain('@media (prefers-reduced-motion: reduce)');
  });
});
