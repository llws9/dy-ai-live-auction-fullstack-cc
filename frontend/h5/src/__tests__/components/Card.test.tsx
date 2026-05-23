import { render, screen } from '@testing-library/react';
import { Card } from '@/components/shared/Card';

describe('Card Component', () => {
  it('renders with children', () => {
    render(<Card>Card content</Card>);
    expect(screen.getByText('Card content')).toBeInTheDocument();
  });

  it('renders with default variant', () => {
    render(<Card>Default</Card>);
    const card = screen.getByText('Default').parentElement;
    expect(card).toHaveClass('card');
  });

  it('renders with elevated variant', () => {
    render(<Card variant="elevated">Elevated</Card>);
    const card = screen.getByText('Elevated').parentElement;
    expect(card).toHaveClass('elevated');
  });

  it('renders with outlined variant', () => {
    render(<Card variant="outlined">Outlined</Card>);
    const card = screen.getByText('Outlined').parentElement;
    expect(card).toHaveClass('outlined');
  });

  it('applies different padding sizes', () => {
    const { rerender } = render(<Card padding="none">No Padding</Card>);
    expect(screen.getByText('No Padding').parentElement).toHaveClass('paddingNone');

    rerender(<Card padding="sm">Small Padding</Card>);
    expect(screen.getByText('Small Padding').parentElement).toHaveClass('paddingSm');

    rerender(<Card padding="md">Medium Padding</Card>);
    expect(screen.getByText('Medium Padding').parentElement).toHaveClass('paddingMd');

    rerender(<Card padding="lg">Large Padding</Card>);
    expect(screen.getByText('Large Padding').parentElement).toHaveClass('paddingLg');
  });

  it('applies clickable class when onClick is provided', () => {
    const handleClick = jest.fn();
    render(<Card onClick={handleClick}>Clickable</Card>);
    const card = screen.getByText('Clickable').parentElement;
    expect(card).toHaveClass('clickable');
  });

  it('calls onClick when clicked', () => {
    const handleClick = jest.fn();
    render(<Card onClick={handleClick}>Click me</Card>);
    screen.getByText('Click me').click();
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('applies custom className', () => {
    render(<Card className="custom-card">Custom</Card>);
    const card = screen.getByText('Custom').parentElement;
    expect(card).toHaveClass('custom-card');
  });
});
