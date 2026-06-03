import { render, screen } from '@testing-library/react';
import { ChatBubble } from '../ChatBubble';

describe('ChatBubble', () => {
  const baseMsg = {
    live_stream_id: 1,
    user_id: 9,
    user_name: 'Alice',
    text: 'hello world',
    sent_at: 1700000000000,
  };

  it('renders user name and text', () => {
    render(<ChatBubble msg={baseMsg} isSelf={false} />);
    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText('hello world')).toBeInTheDocument();
  });

  it('does not interpret HTML in text', () => {
    render(<ChatBubble msg={{ ...baseMsg, text: '<img src=x onerror=alert(1)>' }} isSelf={false} />);
    expect(screen.queryByRole('img')).toBeNull();
    expect(screen.getByText('<img src=x onerror=alert(1)>')).toBeInTheDocument();
  });

  it('marks self messages', () => {
    const { container } = render(<ChatBubble msg={baseMsg} isSelf={true} />);
    expect(container.firstChild).toHaveClass('bubbleSelf');
  });
});
