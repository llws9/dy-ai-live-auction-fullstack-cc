import { render, screen, fireEvent, act } from '@testing-library/react';
import { readFileSync } from 'fs';
import { join } from 'path';
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
    const onSend = jest.fn(() => true);
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

  it('keeps text editable when send fails', () => {
    const onSend = jest.fn(() => false);
    render(<ChatPanel currentUserId={1} onSend={onSend} />);
    const input = screen.getByPlaceholderText(/说点什么/);
    fireEvent.change(input, { target: { value: 'hi' } });
    fireEvent.click(screen.getByRole('button', { name: /发送/ }));

    expect(onSend).toHaveBeenCalledWith('hi', expect.any(String));
    expect(input).toHaveValue('hi');
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

  it('renders quick chat bubbles and sends them on click', () => {
    const onSend = jest.fn(() => true);
    render(<ChatPanel currentUserId={1} onSend={onSend} />);
    const quickChats = screen.getAllByTestId('quick-chat-bubble');
    expect(quickChats.length).toBeGreaterThan(0);

    const firstQuickChatText = quickChats[0].textContent;
    fireEvent.click(quickChats[0]);
    expect(onSend).toHaveBeenCalledWith(firstQuickChatText, expect.any(String));
  });

  it('keeps the chat panel in normal sheet flow without a dark overlay bar', () => {
    const css = readFileSync(join(__dirname, '..', 'ChatPanel.module.css'), 'utf8');

    expect(css).not.toMatch(/\.panel\s*\{[\s\S]*?position:\s*absolute;/);
    expect(css).not.toMatch(/\.inputBar\s*\{[\s\S]*?position:\s*absolute;/);
    expect(css).not.toContain('rgba(0, 0, 0, 0.6)');
  });

  it('keeps the floating input bar compact at roughly 90 percent of the previous size', () => {
    const css = readFileSync(join(__dirname, '..', 'ChatPanel.module.css'), 'utf8');

    expect(css).toMatch(/\.inputBar\s*\{[\s\S]*?padding:\s*5px;/);
    expect(css).toMatch(/\.input\s*\{[\s\S]*?height:\s*25px;/);
    expect(css).toMatch(/\.sendBtn\s*\{[\s\S]*?height:\s*25px;/);
    expect(css).toMatch(/\.sendBtn\s*\{[\s\S]*?min-width:\s*41px;/);
  });
});
