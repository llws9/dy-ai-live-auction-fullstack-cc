import { fireEvent, render, screen } from '@testing-library/react';
import { ToastProvider, useToast } from '../index';

function LegacyTrigger() {
  const { showToast } = useToast();
  return <button onClick={() => showToast('旧提示', 'success', 3000)}>legacy</button>;
}

function RichTrigger({ onAction }: { onAction: () => void }) {
  const { showToast } = useToast();
  return (
    <button
      onClick={() =>
        showToast({
          type: 'danger',
          title: '您已被超价',
          message: '当前最高价已更新',
          actionText: '重新出价',
          onAction,
          duration: 3000,
        })
      }
    >
      rich
    </button>
  );
}

function QueueTrigger() {
  const { showToast } = useToast();
  return (
    <button
      onClick={() => {
        showToast({ type: 'info', message: '一', duration: 3000 });
        showToast({ type: 'info', message: '二', duration: 3000 });
        showToast({ type: 'info', message: '三', duration: 3000 });
        showToast({ type: 'info', message: '四', duration: 3000 });
      }}
    >
      queue
    </button>
  );
}

describe('ToastProvider', () => {
  it('keeps legacy showToast signature', () => {
    render(
      <ToastProvider>
        <LegacyTrigger />
      </ToastProvider>,
    );

    fireEvent.click(screen.getByText('legacy'));
    expect(screen.getByRole('status')).toHaveTextContent('旧提示');
  });

  it('renders rich toast and runs action before closing', () => {
    const onAction = jest.fn();

    render(
      <ToastProvider>
        <RichTrigger onAction={onAction} />
      </ToastProvider>,
    );

    fireEvent.click(screen.getByText('rich'));
    expect(screen.getByText('您已被超价')).toBeInTheDocument();
    expect(screen.getByText('当前最高价已更新')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '重新出价' }));
    expect(onAction).toHaveBeenCalledTimes(1);
    expect(screen.queryByText('您已被超价')).not.toBeInTheDocument();
  });

  it('shows at most three toast items at once', () => {
    render(
      <ToastProvider>
        <QueueTrigger />
      </ToastProvider>,
    );

    fireEvent.click(screen.getByText('queue'));
    expect(screen.getAllByRole('status')).toHaveLength(3);
    expect(screen.getByText('一')).toBeInTheDocument();
    expect(screen.getByText('三')).toBeInTheDocument();
    expect(screen.queryByText('四')).not.toBeInTheDocument();
  });
});
