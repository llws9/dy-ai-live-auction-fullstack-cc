import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveFeedPage from '../LiveFeedPage';
import { liveStreamApi } from '@/services/api';

jest.mock('@/services/api', () => ({
  liveStreamApi: {
    list: jest.fn(),
  },
}));

const mockShowToast = jest.fn();
jest.mock('../../../components/Toast', () => ({
  useToast: () => ({ showToast: mockShowToast }),
}));

jest.mock('../LiveRoomSlide', () => ({
  __esModule: true,
  default: (props: { liveStreamId: number; currentAuctionId?: number | null; urlAuctionId?: number; active: boolean; onBidPendingChange?: (pending: boolean) => void }) => (
    <div data-testid="live-room-slide">
      slide:{props.liveStreamId}:{String(props.currentAuctionId)}:{String(props.urlAuctionId)}:{String(props.active)}
      <button type="button" onClick={() => props.onBidPendingChange?.(true)}>mock-set-pending</button>
      <button type="button" onClick={() => props.onBidPendingChange?.(false)}>mock-clear-pending</button>
    </div>
  ),
}));

const mockedLiveStreamApi = liveStreamApi as jest.Mocked<typeof liveStreamApi>;

const renderFeed = (entry: string) =>
  render(
    <MemoryRouter initialEntries={[entry]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <LiveFeedPage />
    </MemoryRouter>
  );

describe('LiveFeedPage feed 骨架', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [
        { id: 3, name: '房间A', current_auction_id: 11 },
        { id: 4, name: '房间B', current_auction_id: 12 },
      ],
      total: 2,
      page: 1,
      page_size: 20,
    });
  });

  it('按 URL id 初始定位到对应房间（id=4 → 房间B）', async () => {
    renderFeed('/live?id=4');
    const slide = await screen.findByTestId('live-room-slide');
    expect(slide).toHaveTextContent('slide:4:12:undefined:true');
  });

  it('无 id 时展示第一个房间（房间A）', async () => {
    renderFeed('/live');
    const slide = await screen.findByTestId('live-room-slide');
    expect(slide).toHaveTextContent('slide:3:11:undefined:true');
  });

  it('list 为空时展示空态文案', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 });
    renderFeed('/live');
    await waitFor(() => expect(screen.getByText('暂无直播中房间')).toBeInTheDocument());
  });

  it('手指上滑超过阈值切到下一个房间并 replace URL', async () => {
    renderFeed('/live?id=3');
    const slide = await screen.findByTestId('live-room-slide');
    expect(slide).toHaveTextContent('slide:3:11:undefined:true');

    const container = slide.parentElement as HTMLElement;
    fireEvent.touchStart(container, { touches: [{ clientX: 100, clientY: 300 }] });
    fireEvent.touchEnd(container, { changedTouches: [{ clientX: 100, clientY: 220 }] });

    await waitFor(() =>
      expect(screen.getByTestId('live-room-slide')).toHaveTextContent('slide:4:12:12:true')
    );
  });

  it('到末尾继续上滑提示没有更多', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [{ id: 3, name: '房间A', current_auction_id: 11 }],
      total: 1,
      page: 1,
      page_size: 20,
    });
    renderFeed('/live?id=3');
    const slide = await screen.findByTestId('live-room-slide');
    expect(slide).toHaveTextContent('slide:3:11:undefined:true');

    const container = slide.parentElement as HTMLElement;
    fireEvent.touchStart(container, { touches: [{ clientX: 100, clientY: 300 }] });
    fireEvent.touchEnd(container, { changedTouches: [{ clientX: 100, clientY: 220 }] });

    await waitFor(() => expect(mockShowToast).toHaveBeenCalled());
    expect(mockShowToast.mock.calls[0][0]).toContain('没有更多');
    expect(screen.getByTestId('live-room-slide')).toHaveTextContent('slide:3:11:undefined:true');
  });

  it('横向滑动不切房', async () => {
    renderFeed('/live?id=3');
    const slide = await screen.findByTestId('live-room-slide');
    expect(slide).toHaveTextContent('slide:3:11:undefined:true');

    const container = slide.parentElement as HTMLElement;
    // 横向位移占主导，纵向位移很小
    fireEvent.touchStart(container, { touches: [{ clientX: 300, clientY: 300 }] });
    fireEvent.touchEnd(container, { changedTouches: [{ clientX: 100, clientY: 290 }] });

    expect(screen.getByTestId('live-room-slide')).toHaveTextContent('slide:3:11:undefined:true');
  });

  it('出价 pending 时锁房，清除 pending 后恢复切房', async () => {
    renderFeed('/live?id=3');
    const slide = await screen.findByTestId('live-room-slide');
    expect(slide).toHaveTextContent('slide:3:11:undefined:true');

    const container = slide.parentElement as HTMLElement;

    // 置 pending=true → 上滑应被拦截，仍停留房间3
    fireEvent.click(screen.getByRole('button', { name: 'mock-set-pending' }));
    fireEvent.touchStart(container, { touches: [{ clientX: 100, clientY: 300 }] });
    fireEvent.touchEnd(container, { changedTouches: [{ clientX: 100, clientY: 220 }] });
    expect(screen.getByTestId('live-room-slide')).toHaveTextContent('slide:3:11:undefined:true');

    // 清除 pending → 上滑切到房间4
    fireEvent.click(screen.getByRole('button', { name: 'mock-clear-pending' }));
    fireEvent.touchStart(container, { touches: [{ clientX: 100, clientY: 300 }] });
    fireEvent.touchEnd(container, { changedTouches: [{ clientX: 100, clientY: 220 }] });
    await waitFor(() =>
      expect(screen.getByTestId('live-room-slide')).toHaveTextContent('slide:4:12:12:true')
    );
  });
});
