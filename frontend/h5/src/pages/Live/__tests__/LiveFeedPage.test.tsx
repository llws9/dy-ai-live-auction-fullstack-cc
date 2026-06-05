import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import LiveFeedPage from '../LiveFeedPage';
import { auctionApi, liveStreamApi, productReminderApi } from '@/services/api';

jest.mock('@/services/api', () => ({
  liveStreamApi: {
    list: jest.fn(),
  },
  auctionApi: {
    list: jest.fn(),
  },
  productReminderApi: {
    list: jest.fn(),
    subscribe: jest.fn(),
  },
}));

const mockAuthState = { isAuthenticated: true, loading: false };
jest.mock('@/store/authContext', () => ({ useAuth: () => mockAuthState }));

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
const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedProductReminderApi = productReminderApi as jest.Mocked<typeof productReminderApi>;

const LocationProbe = () => {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}{location.search}</div>;
};

const renderFeed = (entry: string) =>
  render(
    <MemoryRouter initialEntries={[entry]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <LiveFeedPage />
      <LocationProbe />
    </MemoryRouter>
  );

describe('LiveFeedPage feed 骨架', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthState.isAuthenticated = true;
    mockAuthState.loading = false;
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [
        { id: 3, name: '房间A', current_auction_id: 11 },
        { id: 4, name: '房间B', current_auction_id: 12 },
      ],
      total: 2,
      page: 1,
      page_size: 20,
    });
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 2 });
    mockedProductReminderApi.list.mockResolvedValue({ items: [] });
    mockedProductReminderApi.subscribe.mockResolvedValue({ product_id: 501 });
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

  it('无 id 时跳过没有当前竞拍的直播间，展示推荐竞拍房间', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [
        { id: 3, name: '空直播间', current_auction_id: null },
        { id: 4, name: '竞拍直播间', current_auction_id: 12 },
      ],
      total: 2,
      page: 1,
      page_size: 20,
    });

    renderFeed('/live');

    const slide = await screen.findByTestId('live-room-slide');
    expect(slide).toHaveTextContent('slide:4:12:undefined:true');
  });

  it('list 为空时展示可行动空态文案', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 });
    renderFeed('/live');
    await waitFor(() => expect(screen.getByText('当前没有竞拍直播')).toBeInTheDocument());
    expect(screen.getByRole('link', { name: '去首页看拍品' })).toHaveAttribute('href', '/');
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

  it('底部导航进入无 id 的 /live 时，桌面拖拽上滑也能切到下一个房间', async () => {
    renderFeed('/live');
    const slide = await screen.findByTestId('live-room-slide');
    expect(slide).toHaveTextContent('slide:3:11:undefined:true');

    const container = slide.parentElement as HTMLElement;
    fireEvent.mouseDown(container, { clientX: 100, clientY: 300 });
    fireEvent.mouseUp(container, { clientX: 100, clientY: 220 });

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

  it('无正在竞拍直播间且 upcoming 返回 3 条时，展示预告空态且只展示最近 2 条', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [{ id: 3, name: '空直播间', current_auction_id: null }],
      total: 1,
      page: 1,
      page_size: 20,
    });
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        { id: 701, product_id: 501, product_name: '青花瓷瓶', start_time: '2026-06-05T21:00:00Z', start_price: '1200.00' },
        { id: 702, product_id: 502, product_name: '紫砂壶', start_time: '2026-06-05T22:00:00Z', start_price: '800.00' },
        { id: 703, product_id: 503, product_name: '翡翠手镯', start_time: '2026-06-05T23:00:00Z', start_price: '3000.00' },
      ],
      total: 3,
      page: 1,
      page_size: 2,
    });

    renderFeed('/live');

    await waitFor(() =>
      expect(mockedAuctionApi.list).toHaveBeenCalledWith({ status: '0', upcoming: true, page: 1, page_size: 2 })
    );
    expect(await screen.findByText('下一场竞拍正在准备')).toBeInTheDocument();
    expect(screen.getByText('即将开播')).toBeInTheDocument();
    expect(screen.getByText('青花瓷瓶')).toBeInTheDocument();
    expect(screen.getByText('紫砂壶')).toBeInTheDocument();
    expect(screen.queryByText('翡翠手镯')).not.toBeInTheDocument();
  });

  it('点击预告行非按钮区域跳转到竞拍详情页', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [{ id: 3, name: '空直播间', current_auction_id: null }],
      total: 1,
      page: 1,
      page_size: 20,
    });
    mockedAuctionApi.list.mockResolvedValue({
      list: [{ id: 701, product_id: 501, product_name: '青花瓷瓶', start_time: '2026-06-05T21:00:00Z', start_price: '1200.00' }],
      total: 1,
      page: 1,
      page_size: 2,
    });

    renderFeed('/live');

    fireEvent.click(await screen.findByText('青花瓷瓶'));

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/detail?id=701'));
  });

  it('点击订阅按钮调用商品提醒接口且不跳转', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [{ id: 3, name: '空直播间', current_auction_id: null }],
      total: 1,
      page: 1,
      page_size: 20,
    });
    mockedAuctionApi.list.mockResolvedValue({
      list: [{ id: 701, product_id: 501, product_name: '青花瓷瓶', start_time: '2026-06-05T21:00:00Z', start_price: '1200.00' }],
      total: 1,
      page: 1,
      page_size: 2,
    });

    renderFeed('/live');

    fireEvent.click(await screen.findByRole('button', { name: '订阅' }));

    await waitFor(() => expect(mockedProductReminderApi.subscribe).toHaveBeenCalledWith(501));
    expect(screen.getByTestId('location')).toHaveTextContent('/live');
  });

  it('upcoming 接口失败时降级展示去首页看拍品入口', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [{ id: 3, name: '空直播间', current_auction_id: null }],
      total: 1,
      page: 1,
      page_size: 20,
    });
    mockedAuctionApi.list.mockRejectedValue(new Error('upcoming failed'));

    renderFeed('/live');

    expect(await screen.findByText('当前没有竞拍直播')).toBeInTheDocument();
    const homeLink = screen.getByRole('link', { name: '去首页看拍品' });
    expect(homeLink).toHaveAttribute('href', '/');
  });
});
