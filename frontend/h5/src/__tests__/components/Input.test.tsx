import { render, screen, fireEvent } from '@testing-library/react';
import { Input } from '@/components/shared/Input';

describe('Input Component', () => {
  it('renders with placeholder', () => {
    render(<Input placeholder="Enter text" />);
    expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument();
  });

  it('renders with label', () => {
    render(<Input label="Username" />);
    expect(screen.getByText('Username')).toBeInTheDocument();
  });

  it('renders with default variant', () => {
    render(<Input placeholder="Default" />);
    const input = screen.getByPlaceholderText('Default');
    expect(input).toHaveClass('input');
  });

  it('renders with error state', () => {
    render(<Input placeholder="Error" error="This field is required" />);
    const input = screen.getByPlaceholderText('Error');
    expect(input.parentElement).toHaveClass('error');
    expect(screen.getByText('This field is required')).toBeInTheDocument();
  });

  it('renders with success state', () => {
    render(<Input placeholder="Success" success />);
    const input = screen.getByPlaceholderText('Success');
    expect(input.parentElement).toHaveClass('success');
  });

  it('renders disabled state', () => {
    render(<Input placeholder="Disabled" disabled />);
    const input = screen.getByPlaceholderText('Disabled');
    expect(input).toBeDisabled();
  });

  it('shows clear button when clearable and has value', () => {
    render(<Input placeholder="Clearable" clearable value="test" readOnly />);
    const input = screen.getByPlaceholderText('Clearable') as HTMLInputElement;
    expect(input.value).toBe('test');
    // Clear button should be present
    const clearBtn = document.querySelector('.clearButton');
    expect(clearBtn).toBeInTheDocument();
  });

  it('calls onChange when value changes', () => {
    const handleChange = jest.fn();
    render(<Input placeholder="Type" onChange={handleChange} />);
    const input = screen.getByPlaceholderText('Type');
    fireEvent.change(input, { target: { value: 'new value' } });
    expect(handleChange).toHaveBeenCalled();
  });

  it('calls onFocus when focused', () => {
    const handleFocus = jest.fn();
    render(<Input placeholder="Focus" onFocus={handleFocus} />);
    const input = screen.getByPlaceholderText('Focus');
    fireEvent.focus(input);
    expect(handleFocus).toHaveBeenCalled();
  });

  it('calls onBlur when blurred', () => {
    const handleBlur = jest.fn();
    render(<Input placeholder="Blur" onBlur={handleBlur} />);
    const input = screen.getByPlaceholderText('Blur');
    fireEvent.blur(input);
    expect(handleBlur).toHaveBeenCalled();
  });

  it('applies custom className', () => {
    render(<Input placeholder="Custom" className="custom-input" />);
    const container = screen.getByPlaceholderText('Custom').closest('.custom-input');
    expect(container).toHaveClass('custom-input');
  });
});
