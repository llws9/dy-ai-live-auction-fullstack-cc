import { render, screen } from '@testing-library/react';
import BadgeDot from '../index';

describe('BadgeDot', () => {
  it('does not render for empty count', () => {
    const { container } = render(<BadgeDot count={0} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders numeric count', () => {
    render(<BadgeDot count={3} />);
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('caps count with max suffix', () => {
    render(<BadgeDot count={120} max={99} />);
    expect(screen.getByText('99+')).toBeInTheDocument();
  });

  it('renders dot mode without number text', () => {
    render(<BadgeDot dot ariaLabel="有新提醒" />);
    expect(screen.getByLabelText('有新提醒')).toBeInTheDocument();
    expect(screen.queryByText('1')).not.toBeInTheDocument();
  });
});
