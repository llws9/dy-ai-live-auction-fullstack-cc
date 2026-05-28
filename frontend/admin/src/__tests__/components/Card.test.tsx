import { render, screen, fireEvent } from '@testing-library/react';
import { Card } from '@/components/shared/Card';

describe('Card Component', () => {
  it('renders with children', () => {
    render(<Card>Card Content</Card>);
    expect(screen.getByText('Card Content')).toBeInTheDocument();
  });

  it('renders with default variant', () => {
    render(<Card>Default Card</Card>);
    // Card renders as a div with class applied
    const card = screen.getByText('Default Card').closest('div');
    expect(card?.className).toMatch(/card/);
  });

  it('renders with elevated variant', () => {
    render(<Card variant="elevated">Elevated Card</Card>);
    const card = screen.getByText('Elevated Card').closest('div');
    expect(card?.className).toMatch(/elevated/);
  });

  it('renders with outlined variant', () => {
    render(<Card variant="outlined">Outlined Card</Card>);
    const card = screen.getByText('Outlined Card').closest('div');
    expect(card?.className).toMatch(/outlined/);
  });

  it('renders with different padding sizes', () => {
    const { rerender } = render(<Card padding="none">No Padding</Card>);
    const card = screen.getByText('No Padding').closest('div');
    expect(card?.className).toMatch(/padding-none/);

    rerender(<Card padding="sm">Small Padding</Card>);
    expect(screen.getByText('Small Padding').closest('div')?.className).toMatch(/padding-sm/);

    rerender(<Card padding="lg">Large Padding</Card>);
    expect(screen.getByText('Large Padding').closest('div')?.className).toMatch(/padding-lg/);
  });

  it('handles click events', () => {
    const handleClick = jest.fn();
    render(<Card onClick={handleClick}>Clickable Card</Card>);
    fireEvent.click(screen.getByText('Clickable Card'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('has role="button" when clickable', () => {
    render(<Card onClick={() => {}}>Clickable Card</Card>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<Card className="custom-class">Custom Card</Card>);
    const card = screen.getByText('Custom Card').closest('div');
    expect(card?.className).toMatch(/custom-class/);
  });

  it('renders with testId', () => {
    render(<Card testId="test-card">Test Card</Card>);
    expect(screen.getByTestId('test-card')).toBeInTheDocument();
  });
});