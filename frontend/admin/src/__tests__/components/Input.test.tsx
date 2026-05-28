import { render, screen, fireEvent } from '@testing-library/react';
import { Input } from '@/components/shared/Input';

describe('Input Component', () => {
  it('renders input element', () => {
    render(<Input />);
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('renders with label', () => {
    render(<Input label="Username" id="username" />);
    expect(screen.getByLabelText('Username')).toBeInTheDocument();
  });

  it('renders with placeholder', () => {
    render(<Input placeholder="Enter text" />);
    expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument();
  });

  it('renders with error state', () => {
    render(<Input error="Invalid input" />);
    const input = screen.getByRole('textbox');
    expect(input.parentElement?.className).toMatch(/error/);
    expect(screen.getByText('Invalid input')).toBeInTheDocument();
  });

  it('renders with success state', () => {
    render(<Input success />);
    const input = screen.getByRole('textbox');
    expect(input.parentElement?.className).toMatch(/success/);
  });

  it('renders with different sizes', () => {
    const { rerender } = render(<Input inputSize="sm" />);
    expect(screen.getByRole('textbox').parentElement?.className).toMatch(/sm/);

    rerender(<Input inputSize="md" />);
    expect(screen.getByRole('textbox').parentElement?.className).toMatch(/md/);

    rerender(<Input inputSize="lg" />);
    expect(screen.getByRole('textbox').parentElement?.className).toMatch(/lg/);
  });

  it('handles value changes', () => {
    const handleChange = jest.fn();
    render(<Input value="test" onChange={handleChange} />);
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'new value' } });
    expect(handleChange).toHaveBeenCalled();
  });

  it('shows clear button when clearable and has value', () => {
    render(<Input value="text" clearable onChange={() => {}} />);
    expect(screen.getByLabelText('清除')).toBeInTheDocument();
  });

  it('does not show clear button when empty', () => {
    render(<Input value="" clearable onChange={() => {}} />);
    expect(screen.queryByLabelText('清除')).not.toBeInTheDocument();
  });

  it('calls onClear when clear button clicked', () => {
    const handleClear = jest.fn();
    render(<Input value="text" clearable onChange={() => {}} onClear={handleClear} />);
    fireEvent.click(screen.getByLabelText('清除'));
    expect(handleClear).toHaveBeenCalled();
  });

  it('does not show clear button when disabled', () => {
    render(<Input value="text" clearable disabled onChange={() => {}} />);
    expect(screen.queryByLabelText('清除')).not.toBeInTheDocument();
  });

  it('applies fullWidth class', () => {
    render(<Input fullWidth />);
    // fullWidth is applied to the outermost container, not inputWrapper
    const outerContainer = screen.getByRole('textbox').closest('div')?.parentElement;
    expect(outerContainer?.className).toMatch(/fullWidth/);
  });

  it('applies custom className', () => {
    render(<Input className="custom-input" />);
    // className is applied to the outermost container
    const outerContainer = screen.getByRole('textbox').closest('div')?.parentElement;
    expect(outerContainer?.className).toMatch(/custom-input/);
  });

  it('handles focus and blur events', () => {
    const handleFocus = jest.fn();
    const handleBlur = jest.fn();
    render(<Input onFocus={handleFocus} onBlur={handleBlur} />);

    const input = screen.getByRole('textbox');
    fireEvent.focus(input);
    expect(handleFocus).toHaveBeenCalled();

    fireEvent.blur(input);
    expect(handleBlur).toHaveBeenCalled();
  });
});