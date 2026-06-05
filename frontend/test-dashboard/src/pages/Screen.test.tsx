import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Screen from './Screen';
import { useWSStore } from '@/store/wsStore';
import { discoverWS, startUserJourney } from '@/api/test';

const connect = vi.fn();
const disconnect = vi.fn();
const pollStart = vi.fn();
const pollCancel = vi.fn();

vi.mock('@/api/test', () => ({
  startUserJourney: vi.fn(async () => 'tj_demo'),
  discoverWS: vi.fn(async () => 'ws://localhost:18092/ws/test/progress?test_id=tj_demo'),
  cancelTest: vi.fn(async () => undefined),
}));

vi.mock('@/hooks/usePollReport', () => ({
  usePollReport: () => ({ start: pollStart, cancel: pollCancel }),
}));

describe('Screen demo theater', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWSStore.setState({
      connected: false,
      testID: null,
      progress: 0,
      step: '',
      metrics: {},
      history: [],
      socket: null,
      connect,
      disconnect,
    });
  });

  it('renders judge-facing theater instead of history dashboard', () => {
    renderScreen();

    expect(screen.getByText('AI 直播竞拍全链路验收')).toBeInTheDocument();
    expect(screen.getByText('H5 直播间同步画面')).toBeInTheDocument();
    expect(screen.getByText('事件证据流')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '开始演示' })).toBeInTheDocument();
    expect(screen.getByText('业务闭环成立')).toBeInTheDocument();
    expect(screen.queryByText('总任务')).not.toBeInTheDocument();
  });

  it('starts standard user journey demo from the big screen', async () => {
    const user = userEvent.setup();
    renderScreen();

    await user.click(screen.getByRole('button', { name: '开始演示' }));

    await waitFor(() => {
      expect(startUserJourney).toHaveBeenCalledWith({
        include_reminder: true,
        include_sky_lamp: true,
        include_fixed_price: true,
        auction_duration_sec: 30,
        buyer_count: 1,
        keep_evidence: true,
      });
      expect(discoverWS).toHaveBeenCalledWith('tj_demo');
      expect(connect).toHaveBeenCalledWith('ws://localhost:18092/ws/test/progress?test_id=tj_demo', 'tj_demo');
      expect(pollStart).toHaveBeenCalled();
    });
  });

  it('keeps ws and report polling alive when connect triggers a rerender', async () => {
    connect.mockImplementationOnce((_wsURL: string, testID: string) => {
      useWSStore.setState({
        connected: true,
        testID,
        progress: 1,
        step: 'prepare',
        metrics: {},
        history: [],
        socket: null,
        connect,
        disconnect,
      });
    });
    const user = userEvent.setup();
    renderScreen();

    await user.click(screen.getByRole('button', { name: '开始演示' }));

    await waitFor(() => {
      expect(connect).toHaveBeenCalledWith('ws://localhost:18092/ws/test/progress?test_id=tj_demo', 'tj_demo');
      expect(pollStart).toHaveBeenCalled();
    });
    expect(disconnect).not.toHaveBeenCalled();
    expect(pollCancel).not.toHaveBeenCalled();
  });

  it('shows live bid and sky lamp story from ws history', () => {
    useWSStore.setState({
      connected: true,
      testID: 'tj_live',
      progress: 62,
      step: 'sky_lamp',
      metrics: {},
      history: [
        {
          test_id: 'tj_live',
          progress: 50,
          step: 'auction_bid',
          metrics: {
            demo_snapshot: {
              current_price: '110.00',
              leader_label: '买家 2001',
              bid_count: 1,
              stock_before: 1,
              stock_after: 1,
              highlighted_event: 'bid',
            },
          },
          ts: Date.now(),
        },
        {
          test_id: 'tj_live',
          progress: 62,
          step: 'sky_lamp',
          metrics: {
            demo_snapshot: {
              current_price: '110.00',
              leader_label: '买家 2001',
              bid_count: 1,
              highlighted_event: 'sky_lamp',
            },
          },
          ts: Date.now(),
        },
      ],
      socket: null,
      connect,
      disconnect,
    });

    renderScreen();

    expect(screen.getAllByText('¥110.00').length).toBeGreaterThan(0);
    expect(screen.getByText('买家 2001')).toBeInTheDocument();
    expect(screen.getByText('买家 2001 正在领先')).toBeInTheDocument();
    expect(screen.getAllByText('点天灯触发').length).toBeGreaterThan(0);
    expect(screen.getByText('天灯锁定领先')).toBeInTheDocument();
  });

  it('shows fixed-price purchase as a live-room deal moment', () => {
    useWSStore.setState({
      connected: true,
      testID: 'tj_deal',
      progress: 75,
      step: 'fixed_price_purchase',
      metrics: {},
      history: [
        {
          test_id: 'tj_deal',
          progress: 75,
          step: 'fixed_price_purchase',
          metrics: {
            demo_snapshot: {
              current_price: '110.00',
              leader_label: '买家 2001',
              bid_count: 1,
              order_count: 1,
              stock_before: 1,
              stock_after: 0,
              highlighted_event: 'order',
            },
          },
          ts: Date.now(),
        },
      ],
      socket: null,
      connect,
      disconnect,
    });

    renderScreen();

    expect(screen.getByText('成交弹幕')).toBeInTheDocument();
    expect(screen.getByText('库存 1 → 0')).toBeInTheDocument();
    expect(screen.getByText('订单已生成')).toBeInTheDocument();
  });
});

function renderScreen() {
  render(
    <MemoryRouter>
      <Screen />
    </MemoryRouter>,
  );
}
