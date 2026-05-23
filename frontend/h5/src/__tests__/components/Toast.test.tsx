import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { Toast } from '@/components/shared/Toast';

describe('Toast Component', () => {
  it('renders with message', () => {
    render(<Toast message="Test message" visible />);
    expect(screen.getByText('Test message')).toBeInTheDocument();
  });

  it('renders with different types', () => {
    const { rerender } = render(<Toast message="Info" type="info" visible />);
    expect(document.querySelector('.info')).toBeInTheDocument();

    rerender(<Toast message="Success" type="success" visible />);
    expect(document.querySelector('.success')).toBeInTheDocument();

    rerender(<Toast message="Warning" type="warning" visible />);
    expect(document.querySelector('.warning')).toBeInTheDocument();

    rerender(<Toast message="Error" type="error" visible />);
    expect(document.querySelector('.error')).toBeInTheDocument();
  });

  it('does not render when not visible', () => {
    render(<Toast message="Hidden" visible={false} />);
    expect(screen.queryByText('Hidden')).not.toBeInTheDocument();
  });

  it('calls onClose after duration', async () => {
    jest.useFakeTimers();
    const handleClose = jest.fn();
    render(<Toast message="Auto close" visible duration={3000} onClose={handleClose} />);

    jest.advanceTimersByTime(3000);
    expect(handleClose).toHaveBeenCalled();
    jest.useRealTimers();
  });

  it('applies custom className', () => {
    render(<Toast message="Custom" visible className="custom-toast" />);
    const toast = screen.getByText('Custom').parentElement;
    expect(toast).toHaveClass('custom-toast');
  });
});
