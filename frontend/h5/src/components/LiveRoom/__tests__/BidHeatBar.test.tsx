import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { render, screen } from '@testing-library/react';
import { BidHeatBar } from '../BidHeatBar';

describe('BidHeatBar', () => {
  it('renders calm label and audience stats', () => {
    render(<BidHeatBar level="calm" bidderCount={1} viewerCount={88} />);

    expect(screen.getByText('战况冷静')).toBeInTheDocument();
    expect(screen.getByText('已有 1 人出价')).toBeInTheDocument();
    expect(screen.getByText('88 人围观')).toBeInTheDocument();
    expect(screen.getByRole('meter', { name: '战况热度' })).toHaveAttribute('aria-valuemax', '100');
    expect(screen.getByRole('meter', { name: '战况热度' })).toHaveAttribute('aria-valuenow', '24');
    expect(screen.getByTestId('bid-heat-fill')).toHaveStyle({ transform: 'scaleX(0.24)' });
  });

  it('renders warming label and warm state class', () => {
    render(<BidHeatBar level="warming" bidderCount={3} viewerCount={128} />);

    expect(screen.getByText('战况升温')).toBeInTheDocument();
    expect(screen.getByTestId('bid-heat-bar')).toHaveClass('warming');
    expect(screen.getByText('已有 3 人出价')).toBeInTheDocument();
    expect(screen.getByText('128 人围观')).toBeInTheDocument();
  });

  it('renders blazing label and full heat meter', () => {
    render(<BidHeatBar level="blazing" bidderCount={5} viewerCount={256} />);

    expect(screen.getByText('战况白热')).toBeInTheDocument();
    expect(screen.getByTestId('bid-heat-bar')).toHaveClass('blazing');
    expect(screen.getByRole('meter', { name: '战况热度' })).toHaveAttribute('aria-valuenow', '100');
  });

  it('uses design tokens instead of hardcoded hex colors', () => {
    const css = readFileSync(join(__dirname, '..', 'BidHeatBar.module.css'), 'utf8');

    expect(css).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(css).toContain('var(--color-');
    expect(css).toContain('var(--text-');
  });

  it('keeps the component slim enough for the bid sheet', () => {
    const css = readFileSync(join(__dirname, '..', 'BidHeatBar.module.css'), 'utf8');

    expect(css).toContain('gap: 4px;');
    expect(css).toContain('padding: 6px 10px;');
    expect(css).toContain('height: 6px;');
    expect(css).toContain('width: 100%;');
  });
});
