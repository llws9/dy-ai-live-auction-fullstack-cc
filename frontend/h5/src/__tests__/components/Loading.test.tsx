import { render, screen } from '@testing-library/react';
import { Loading } from '@/components/shared/Loading';

describe('Loading Component', () => {
  it('renders spinner by default', () => {
    render(<Loading />);
    expect(document.querySelector('.spinner')).toBeInTheDocument();
  });

  it('renders with different sizes', () => {
    const { rerender } = render(<Loading size="sm" />);
    expect(document.querySelector('.spinner')).toHaveClass('sm');

    rerender(<Loading size="md" />);
    expect(document.querySelector('.spinner')).toHaveClass('md');

    rerender(<Loading size="lg" />);
    expect(document.querySelector('.spinner')).toHaveClass('lg');
  });

  it('renders with text', () => {
    render(<Loading text="Loading..." />);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<Loading className="custom-loading" />);
    const container = document.querySelector('.custom-loading');
    expect(container).toBeInTheDocument();
  });

  it('renders full screen overlay when fullScreen is true', () => {
    render(<Loading fullScreen />);
    const overlay = document.querySelector('.fullScreen');
    expect(overlay).toBeInTheDocument();
  });
});
