import { render, screen, fireEvent, act } from '@testing-library/react';
import { ChatPanel } from '../ChatPanel';
import { useLiveChatStore } from '../../../store/liveChatStore';

describe('ChatPanel', () => {
  beforeEach(() => {
    useLiveChatStore.getState().reset();
    jest.useFakeTimers();
  });
  afterEach(() => {
    jest.useRealTimers();
  });

  it('disables send button while empty', () => {
    render(<ChatPanel currentUserId={1} onSend={jest.fn()} />);
    expect(screen.getByRole('button', { name: /发送/ })).toBeDisabled();
  });

  it('rejects text exceeding 50 chars', () => {
    const onSend = jest.fn();
    render(<ChatPanel currentUserId={1} onSend={onSend} />);
    const input = screen.getByPlaceholderText(/说点什么/);
    fireEvent.change(input, { target: { value: 'a'.repeat(51) } });
    fireEvent.click(screen.getByRole('button', { name: /发送/ }));
    expect(onSend).not.toHaveBeenCalled();
  });

  it('sends valid text and triggers cooldown', () => {
    const onSend = jest.fn();
    render(<ChatPanel currentUserId={1} onSend={onSend} />);
    const input = screen.getByPlaceholderText(/说点什么/);
    fireEvent.change(input, { target: { value: 'hi' } });
    fireEvent.click(screen.getByRole('button', { name: /发送/ }));
    expect(onSend).toHaveBeenCalledWith('hi', expect.any(String));
    expect(screen.getByRole('button', { name: /发送/ })).toBeDisabled(); // 1s cooldown

    act(() => {
      jest.advanceTimersByTime(1100);
    });
    fireEvent.change(input, { target: { value: 'next' } });
    expect(screen.getByRole('button', { name: /发送/ })).not.toBeDisabled();
  });

  it('renders messages from store', () => {
    useLiveChatStore.getState().receive({
      live_stream_id: 1,
      user_id: 9,
      user_name: 'Bob',
      text: 'arriving',
      sent_at: Date.now(),
    });
    render(<ChatPanel currentUserId={1} onSend={jest.fn()} />);
    expect(screen.getByText('Bob')).toBeInTheDocument();
    expect(screen.getByText('arriving')).toBeInTheDocument();
  });
});
