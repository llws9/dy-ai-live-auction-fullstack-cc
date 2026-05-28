import { render, screen } from '@testing-library/react';
import { Loading } from '@/components/shared/Loading';

describe('Loading Component', () => {
  it('renders loading spinner', () => {
    render(<Loading />);
    // Loading doesn't have role="status", so we check for spinner class
    const spinner = document.querySelector('[class*="spinner"]');
    expect(spinner).toBeInTheDocument();
  });

  it('renders with text', () => {
    render(<Loading text="Loading..." />);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders without text when not provided', () => {
    render(<Loading />);
    expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
  });

  it('renders with different sizes', () => {
    const { rerender } = render(<Loading size="sm" />);
    const spinnerSm = document.querySelector('[class*="spinner"]');
    expect(spinnerSm?.className).toMatch(/sm/);

    rerender(<Loading size="md" />);
    const spinnerMd = document.querySelector('[class*="spinner"]');
    expect(spinnerMd?.className).toMatch(/md/);

    rerender(<Loading size="lg" />);
    const spinnerLg = document.querySelector('[class*="spinner"]');
    expect(spinnerLg?.className).toMatch(/lg/);
  });

  it('applies fullscreen class when fullscreen is true', () => {
    render(<Loading fullscreen />);
    const container = document.querySelector('[class*="container"]');
    expect(container?.className).toMatch(/fullscreen/);
  });

  it('applies custom className', () => {
    render(<Loading className="custom-loading" />);
    const container = document.querySelector('[class*="container"]');
    expect(container?.className).toMatch(/custom-loading/);
  });
});