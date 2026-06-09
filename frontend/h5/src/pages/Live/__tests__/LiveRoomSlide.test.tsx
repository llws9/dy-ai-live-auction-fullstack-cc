import React from 'react';
import { render, screen, fireEvent, waitFor, act, within } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import LiveRoomSlide from '../LiveRoomSlide';
import { auctionApi, bidApi, followApi, liveStreamApi, productApi, skyLampApi } from '../../../services/api';
import WebSocketService from '../../../services/websocket';
import { useFixedPriceItems } from '../../../hooks/useFixedPriceItems';
import { trackBusinessEvent } from '../../../utils/businessEvent';
import { useDemo } from '../../../store/demoContext';

const mockShowGlobalToast = jest.fn();
const mockNavigate = jest.fn();
const mockSetCurrentAuctionId = jest.fn();
const mockSetCurrentLiveStreamId = jest.fn();
let mockAuthUser = { id: 9, name: '测试用户', role: 0 };
const mockWebSocketInstance = {
  on: jest.fn(),
  off: jest.fn(),
  onNotification: jest.fn(),
  connect: jest.fn().mockResolvedValue(undefined),
  requestSync: jest.fn(),
  disconnect: jest.fn(),
};

jest.mock('../../../services/api', () => ({
  auctionApi: {
    get: jest.fn(),
    getBids: jest.fn(),
  },
  bidApi: {
    getRanking: jest.fn(),
    placeBid: jest.fn(),
  },
  skyLampApi: {
    startSubscription: jest.fn(),
    listSubscriptions: jest.fn(),
  },
  followApi: {
    followLiveStream: jest.fn(),
    unfollowLiveStream: jest.fn(),
    getFollowersStats: jest.fn(),
    getFollowStatus: jest.fn(),
  },
  liveStreamApi: {
    get: jest.fn(),
  },
  productApi: {
    get: jest.fn(),
  },
}));

jest.mock('../../../services/websocket', () => ({
  __esModule: true,
  default: jest.fn(() => mockWebSocketInstance),
}));

jest.mock('../../../hooks/useFixedPriceItems', () => ({
  useFixedPriceItems: jest.fn(),
}));

jest.mock('../../../utils/businessEvent', () => ({
  trackBusinessEvent: jest.fn(),
}));

jest.mock('../../../components/LiveChat/ChatPanel', () => ({
  ChatPanel: () => <div data-testid="chat-panel" />,
}));

jest.mock('../../../store/liveChatStore', () => ({
  useLiveChatStore: {
    getState: () => ({
      receive: jest.fn(),
      reset: jest.fn(),
    }),
  },
}));

jest.mock('@/utils/env', () => ({
  IS_DEV: true,
  IS_PROD: false,
  ENV: {
    API_BASE_URL: '',
    GROWTHBOOK_API_HOST: 'http://localhost:3200',
    GROWTHBOOK_CLIENT_KEY: 'dev-client-key',
  },
}));

jest.mock('react-router-dom', () => {
  const actual = jest.requireActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

jest.mock('../../../store/authContext', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    user: mockAuthUser,
    token: 'token-1',
    loading: false,
  }),
}));

jest.mock('../../../store/demoContext', () => ({
  useDemo: jest.fn(),
}));

jest.mock('../../../components/Toast', () => ({
  __esModule: true,
  useToast: () => ({
    showToast: mockShowGlobalToast,
    hideToast: jest.fn(),
  }),
  showGlobalToast: jest.fn(),
}));

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedBidApi = bidApi as jest.Mocked<typeof bidApi>;
const mockedSkyLampApi = skyLampApi as jest.Mocked<typeof skyLampApi>;
const mockedFollowApi = followApi as jest.Mocked<typeof followApi>;
const mockedLiveStreamApi = liveStreamApi as jest.Mocked<typeof liveStreamApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;
const MockedWebSocketService = WebSocketService as jest.MockedClass<typeof WebSocketService>;
const mockedUseFixedPriceItems = useFixedPriceItems as jest.MockedFunction<typeof useFixedPriceItems>;
const mockedTrackBusinessEvent = trackBusinessEvent as jest.MockedFunction<typeof trackBusinessEvent>;
const mockedUseDemo = useDemo as jest.MockedFunction<typeof useDemo>;

const LocationDisplay: React.FC = () => {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
};

const toUtf8Mojibake = (text: string) =>
  encodeURIComponent(text).replace(/%([0-9A-F]{2})/g, (_, hex: string) => String.fromCharCode(parseInt(hex, 16)));

const renderSlide = (props: Partial<React.ComponentProps<typeof LiveRoomSlide>> = {}) =>
  render(
    <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <LiveRoomSlide
        liveStreamId={3}
        currentAuctionId={5}
        active
        {...props}
      />
      <LocationDisplay />
    </MemoryRouter>
  );

const getWebSocketHandler = (type: string) => mockWebSocketInstance.on.mock.calls.find((call) => call[0] === type)?.[1] as
  | ((data: any) => void)
  | undefined;

describe('LiveRoomSlide', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthUser = { id: 9, name: '测试用户', role: 0 };
    mockedUseDemo.mockReturnValue({
      currentAuctionId: null,
      setCurrentAuctionId: mockSetCurrentAuctionId,
      currentLiveStreamId: null,
      setCurrentLiveStreamId: mockSetCurrentLiveStreamId,
    });
    mockWebSocketInstance.connect.mockResolvedValue(undefined);
    mockWebSocketInstance.onNotification.mockReturnValue(jest.fn());
    mockedUseFixedPriceItems.mockReturnValue({ items: [], byId: {}, socket: null, latestListedItem: null });

    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(Date.now() + 60_000).toISOString(),
    });
    mockedAuctionApi.getBids.mockResolvedValue([
      { id: 1, user_id: 2, user_name: '李四', amount: 1100, created_at: new Date().toISOString() },
    ]);
    mockedProductApi.get.mockResolvedValue({
      id: 7,
      name: '明代紫砂壶',
      images: ['/product.jpg'],
      rules: { start_price: 800, increment: 100 },
    });
    mockedLiveStreamApi.get.mockResolvedValue({
      id: 3,
      name: '瓷器珍藏夜场',
      host_name: '拍卖师王老师',
      viewer_count: 128,
      is_following: false,
      followers_count: 12,
    });
    mockedBidApi.getRanking.mockResolvedValue([
      { rank: 1, user_id: 1, user_name: '张三', amount: 1200 },
    ]);
    mockedBidApi.placeBid.mockResolvedValue({
      current_price: 1300,
      ranking: [{ rank: 1, user_id: 9, user_name: '测试用户', amount: 1300 }],
    });
    mockedSkyLampApi.startSubscription.mockResolvedValue({
      code: 200,
      message: '点天灯订阅已开启',
      subscription: { id: 18, auction_id: 5 },
    });
    mockedSkyLampApi.listSubscriptions.mockResolvedValue({
      code: 200,
      subscriptions: [],
    });
    mockedFollowApi.followLiveStream.mockResolvedValue({});
    mockedFollowApi.unfollowLiveStream.mockResolvedValue({});
    mockedFollowApi.getFollowersStats.mockResolvedValue({ count: 12 });
    mockedFollowApi.getFollowStatus.mockResolvedValue({ is_following: false });
  });

  it('falls back to currentAuctionId when urlAuctionId does not belong to current live stream', async () => {
    mockedAuctionApi.get.mockImplementation(async (id: number) => {
      if (id === 999) {
        return { id: 999, live_stream_id: 4, product_id: 7, status: 1, current_price: 1200 };
      }
      return {
        id: 11,
        live_stream_id: 3,
        product_id: 8,
        status: 1,
        current_price: 1200,
        end_time: new Date(Date.now() + 60_000).toISOString(),
      };
    });
    mockedProductApi.get.mockResolvedValue({
      id: 8,
      name: '回退后的商品',
      images: ['/p8.jpg'],
      rules: { start_price: 800, increment: 100 },
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 11, urlAuctionId: 999 });

    await waitFor(() => expect(mockedAuctionApi.get).toHaveBeenCalledWith(11));
    expect(mockedAuctionApi.get).toHaveBeenCalledWith(999);
    expect((await screen.findAllByText('回退后的商品')).length).toBeGreaterThan(0);
  });

  it('does not open websocket when active is false', async () => {
    renderSlide({ active: false });

    await waitFor(() => expect(mockedAuctionApi.get).toHaveBeenCalledWith(5));
    expect(MockedWebSocketService).not.toHaveBeenCalled();
  });

  it('tracks live room entry after auction and product context are ready', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    await waitFor(() => expect(mockedTrackBusinessEvent).toHaveBeenCalledWith('live_room_enter', {
      source: 'live_room',
      liveStreamId: 3,
      auctionId: 5,
      productId: 7,
    }));
    expect(mockedTrackBusinessEvent.mock.calls.filter(([event]) => event === 'live_room_enter')).toHaveLength(1);
  });

  it('renders top follow action, online viewers, and a product detail link in the header', async () => {
    const { container } = renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    const header = container.querySelector('header');
    expect(header).toBeInTheDocument();
    expect(within(header as HTMLElement).getByRole('button', { name: '收藏' })).toBeInTheDocument();
    const viewersRow = within(header as HTMLElement).getByLabelText('在线人数');
    const likesPill = within(header as HTMLElement).getByLabelText('点赞数');
    const productDetailLink = within(header as HTMLElement).getByRole('link', { name: /商品详情/ });
    expect(viewersRow).toHaveTextContent('128');
    expect(likesPill).toHaveTextContent('0');
    expect(within(header as HTMLElement).queryByLabelText('退出直播间')).not.toBeInTheDocument();
    expect(within(header as HTMLElement).queryByText('正在竞拍')).not.toBeInTheDocument();
    expect(productDetailLink).toHaveAttribute('href', '/detail?id=5');
  });

  it('uses live presence updates as the authoritative online viewers state', async () => {
    const { container } = renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    await waitFor(() => expect(getWebSocketHandler('live_presence_update')).toBeDefined());

    act(() => {
      getWebSocketHandler('live_presence_update')?.({
        live_stream_id: 3,
        viewer_count: 3,
        viewers: [
          { user_id: 1, name: '张三', avatar_url: '/u1.png' },
          { user_id: 2, name: '李四', avatar_url: '/u2.png' },
          { user_id: 3, name: '王五', avatar_url: '/u3.png' },
          { user_id: 4, name: '赵六', avatar_url: '/u4.png' },
        ],
      });
    });

    const header = container.querySelector('header') as HTMLElement;
    const viewersRow = within(header).getByLabelText('在线人数');
    expect(viewersRow).toHaveTextContent('3');
    expect(viewersRow.querySelectorAll('img')).toHaveLength(3);
  });

  it('ignores live presence updates from other live streams', async () => {
    const { container } = renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    await waitFor(() => expect(getWebSocketHandler('live_presence_update')).toBeDefined());

    act(() => {
      getWebSocketHandler('live_presence_update')?.({
        live_stream_id: 4,
        viewer_count: 3,
        viewers: [
          { user_id: 1, name: '张三', avatar_url: '/u1.png' },
        ],
      });
    });

    const header = container.querySelector('header') as HTMLElement;
    const viewersRow = within(header).getByLabelText('在线人数');
    expect(viewersRow).toHaveTextContent('128');
    expect(viewersRow.querySelectorAll('img')).toHaveLength(0);
  });

  it('does not render the product card follow row in the bid drawer while keeping ranking visible', async () => {
    const { container } = renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    fireEvent.click(await screen.findByTestId('bid-dock'));

    await waitFor(() => {
      expect(screen.queryByText(/人收藏/)).not.toBeInTheDocument();
    });
    expect(screen.getByText('出价排行')).toBeInTheDocument();
    const heatBar = screen.getByLabelText('竞拍战况热度');
    expect(heatBar.parentElement).toBe(container.querySelector('.heatMarqueeContainer'));
  });

  it('publishes the resolved auction id to DemoContext and clears it on unmount', async () => {
    const { unmount } = renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    await waitFor(() => expect(mockSetCurrentAuctionId).toHaveBeenCalledWith(5));
    await waitFor(() => expect(mockSetCurrentLiveStreamId).toHaveBeenCalledWith(3));

    unmount();

    expect(mockSetCurrentAuctionId).toHaveBeenLastCalledWith(null);
    expect(mockSetCurrentLiveStreamId).toHaveBeenLastCalledWith(null);
  });

  it('places a bid and refreshes ranking', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    fireEvent.click(screen.getByRole('button', { name: /立即出价/ }));
    await waitFor(() => expect(mockedBidApi.placeBid).toHaveBeenCalledWith(5, 1300));
    // 出价成功后 sheet 自动收起，重新展开以校验排行已刷新
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    // 由于渲染逻辑修改，当前用户的出价在排行中会显示为 "我自己 (当前领先)"
    expect(await screen.findByText('我自己 (当前领先)')).toBeInTheDocument();
  });

  it('starts sky lamp only after confirming the A+C bid drawer action', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    const skyLampButton = screen.getByRole('button', { name: /点天灯/ });
    expect(skyLampButton).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /立即出价/ })).toBeInTheDocument();

    fireEvent.click(skyLampButton);

    expect(screen.getByText('确认开启点天灯？')).toBeInTheDocument();
    expect(mockedSkyLampApi.startSubscription).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: '确认开启' }));

    await waitFor(() => expect(mockedSkyLampApi.startSubscription).toHaveBeenCalledWith(5));
    expect(mockShowGlobalToast).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'success',
        title: '点天灯已开启',
      })
    );
  });

  it('locks the sky lamp state, closes the drawer, and highlights the dock after confirmation', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    fireEvent.click(screen.getByRole('button', { name: /点天灯/ }));
    fireEvent.click(screen.getByRole('button', { name: '确认开启' }));

    await waitFor(() => expect(mockedSkyLampApi.startSubscription).toHaveBeenCalledWith(5));
    expect(screen.queryByText('确认开启点天灯？')).not.toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId('location-search')).toBeEmptyDOMElement());

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    const lockedSkyLampButton = screen.getByRole('button', { name: /守护中/ });
    expect(lockedSkyLampButton).toBeDisabled();
    expect(lockedSkyLampButton.querySelector('[data-testid="sky-lamp-floating-icon"]')).toBeInTheDocument();
    expect(screen.getByText('测试用户 开启点天灯，自动守住领先')).toBeInTheDocument();
    expect(screen.getByTestId('bid-dock')).toHaveAttribute('data-sky-lamp-active', 'true');
    expect(screen.getByTestId('dock-sky-lamp-icon')).toBeInTheDocument();
  });

  it('repairs mojibake auth user name in the sky lamp notice', async () => {
    mockAuthUser = { id: 9, name: toUtf8Mojibake('测试用户'), role: 0 };

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    fireEvent.click(screen.getByRole('button', { name: /点天灯/ }));
    fireEvent.click(screen.getByRole('button', { name: '确认开启' }));

    await waitFor(() => expect(mockedSkyLampApi.startSubscription).toHaveBeenCalledWith(5));
    expect(screen.getByText('测试用户 开启点天灯，自动守住领先')).toBeInTheDocument();
    expect(screen.queryByText(/æ|å|ç|è/)).not.toBeInTheDocument();
  });

  it('repairs mojibake notification toast title and content', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    const notificationHandler = mockWebSocketInstance.onNotification.mock.calls[0][0] as (notification: any) => void;

    act(() => {
      notificationHandler({
        id: 66,
        type: 'auction_starting',
        title: toUtf8Mojibake('竞拍即将开始'),
        content: toUtf8Mojibake('商品【南红手串】的竞拍即将在30分钟后开始，不要错过！'),
        data: { auction_id: 5 },
      });
    });

    expect(mockShowGlobalToast).toHaveBeenCalledWith(expect.objectContaining({
      title: '竞拍即将开始',
      message: '商品【南红手串】的竞拍即将在30分钟后开始，不要错过！',
    }));
    expect(mockShowGlobalToast.mock.calls[0][0].message).not.toMatch(/æ|å|ç|è/);
  });

  it('treats an already-active sky lamp subscription as active UI state', async () => {
    mockedSkyLampApi.startSubscription.mockRejectedValueOnce(
      Object.assign(new Error('已有活跃的点天灯订阅'), { status: 400 })
    );
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    fireEvent.click(screen.getByRole('button', { name: /点天灯/ }));
    fireEvent.click(screen.getByRole('button', { name: '确认开启' }));

    await waitFor(() => expect(mockedSkyLampApi.startSubscription).toHaveBeenCalledWith(5));
    expect(screen.queryByText('确认开启点天灯？')).not.toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId('location-search')).toBeEmptyDOMElement());

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(screen.getByRole('button', { name: /守护中/ })).toBeDisabled();
    expect(screen.getByTestId('bid-dock')).toHaveAttribute('data-sky-lamp-active', 'true');
    expect(screen.getByText('测试用户 开启点天灯，自动守住领先')).toBeInTheDocument();
  });

  it('restores active sky lamp state from existing subscriptions after refresh', async () => {
    mockedSkyLampApi.listSubscriptions.mockResolvedValueOnce({
      code: 200,
      subscriptions: [{ id: 18, auction_id: 5, status: 1 }],
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    await waitFor(() => expect(mockedSkyLampApi.listSubscriptions).toHaveBeenCalledWith(1));
    expect(screen.getByTestId('bid-dock')).toHaveAttribute('data-sky-lamp-active', 'true');

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(screen.getByRole('button', { name: /守护中/ })).toBeDisabled();
    expect(screen.getByTestId('dock-sky-lamp-icon')).toBeInTheDocument();
  });

  it('shows the sky lamp notice again when auto-bid websocket message arrives', async () => {
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 130,
      end_time: new Date(Date.now() + 60_000).toISOString(),
    });
    mockedProductApi.get.mockResolvedValue({
      id: 7,
      name: '明代紫砂壶',
      images: ['/product.jpg'],
      rules: { start_price: 100, increment: 10 },
    });
    mockedBidApi.getRanking.mockResolvedValue([
      { rank: 1, user_id: 9102, user_name: '演示买家B', amount: 140 },
      { rank: 2, user_id: 9, user_name: '测试用户', amount: 130 },
    ]);

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(screen.getByText('当前最高价 ¥130')).toBeInTheDocument();
    expect(screen.getByText('演示买家B')).toBeInTheDocument();
    const autoBidHandler = getWebSocketHandler('sky_lamp_auto_bid');
    expect(autoBidHandler).toBeDefined();

    act(() => {
      autoBidHandler!({
        auction_id: 5,
        user_id: 9,
        amount: '150',
        remaining_budget: '9850',
        auto_bid_count: 1,
      });
    });

    expect(screen.getByText('测试用户 点天灯自动跟价 ¥150，继续守住领先')).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByText('¥150').length).toBeGreaterThan(0));
    expect(screen.getByText('我自己 (当前领先)')).toBeInTheDocument();
    expect(screen.getByText('演示买家B')).toBeInTheDocument();

    act(() => {
      autoBidHandler!({
        auction_id: 5,
        user_id: 9102,
        amount: '160',
        remaining_budget: '9840',
        auto_bid_count: 2,
      });
    });

    expect(screen.getByText('用户9102 点天灯自动跟价 ¥160，继续守住领先')).toBeInTheDocument();
  });

  it('ignores sky lamp auto-bid websocket messages from other auction rooms', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    const autoBidHandler = getWebSocketHandler('sky_lamp_auto_bid');
    expect(autoBidHandler).toBeDefined();

    act(() => {
      autoBidHandler!({
        auction_id: 999,
        user_id: 9,
        amount: '150',
      });
    });

    expect(screen.queryByText(/点天灯自动跟价/)).not.toBeInTheDocument();
  });

  it('uses total_count from followers stats when count aliases are absent', async () => {
    mockedFollowApi.getFollowersStats.mockResolvedValue({ total_count: 1 });
    mockedLiveStreamApi.get.mockResolvedValue({
      id: 3,
      name: '瓷器珍藏夜场',
      host_name: '拍卖师王老师',
      viewer_count: 128,
      is_following: false,
      followers_count: 0,
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    fireEvent.click(await screen.findByTestId('bid-dock'));

    await waitFor(() => expect(mockedFollowApi.getFollowersStats).toHaveBeenCalledWith(3));
    expect(screen.queryByText(/人收藏/)).not.toBeInTheDocument();
    expect(screen.getByText('出价排行')).toBeInTheDocument();
  });

  it('uses auction rule as authoritative increment when product detail has no rules', async () => {
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: '3400',
      start_price: '3000',
      rules: { start_price: '3000', increment: '200' },
      end_time: new Date(Date.now() + 60_000).toISOString(),
    });
    mockedProductApi.get.mockResolvedValue({
      id: 7,
      name: '明代紫砂壶',
      images: ['/product.jpg'],
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));

    expect(await screen.findByText('加价幅度 ¥200')).toBeInTheDocument();
    expect(screen.getByLabelText('输入出价金额')).toHaveValue(3600);
  });

  it('discards websocket messages whose auction_id does not belong to this room', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    const bidHandler = getWebSocketHandler('bid_placed');
    expect(bidHandler).toBeDefined();

    // 不匹配的消息（auction_id=999）应被丢弃，价格不更新
    act(() => {
      bidHandler!({ auction_id: 999, current_price: 8888 });
    });
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(screen.queryByText('¥8,888')).not.toBeInTheDocument();

    // 匹配的消息（auction_id=5）应更新价格
    act(() => {
      bidHandler!({ auction_id: 5, current_price: 1500 });
    });
    expect(await screen.findByText('¥1,500')).toBeInTheDocument();
  });

  it('shows bid flair and updates current price when another user bid arrives through rank_update', async () => {
    jest.useFakeTimers();

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    expect(screen.getByText('当前最高价 ¥1,200')).toBeInTheDocument();

    const rankUpdateHandler = getWebSocketHandler('rank_update');
    expect(rankUpdateHandler).toBeDefined();

    act(() => {
      rankUpdateHandler!({
        auction_id: 5,
        ranking: [
          { rank: 1, user_id: 9102, user_name: '演示买家B', amount: 1300 },
          { rank: 2, user_id: 9, user_name: '测试用户', amount: 1200 },
        ],
      });
    });

    expect(screen.getByText('当前最高价 ¥1,300')).toBeInTheDocument();

    act(() => {
      jest.advanceTimersByTime(240);
    });

    expect(screen.getByTestId('bid-flair-overlay')).toHaveTextContent('出价');
    expect(screen.getByTestId('bid-flair-overlay')).toHaveTextContent('¥1,300');

    jest.useRealTimers();
  });

  it('updates end_time and shows toast when delay_triggered belongs to this room', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(baseNow + 30_000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('00:30')).toBeInTheDocument();

    const delayHandler = getWebSocketHandler('delay_triggered');
    expect(delayHandler).toBeDefined();

    act(() => {
      delayHandler!({
        auction_id: 5,
        delay_duration: 30,
        new_end_time: baseNow + 180_000,
        remaining_delay: 60,
        max_delay: 90,
      });
    });

    expect(await screen.findByText('03:00')).toBeInTheDocument();
    expect(mockShowGlobalToast).toHaveBeenCalledWith(expect.objectContaining({
      type: 'info',
      title: '触发防狙击',
      message: '已有新出价，竞拍时间自动延长',
    }));

    dateNowSpy.mockRestore();
  });

  it('ignores delay_triggered messages from other auction rooms', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(baseNow + 30_000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('00:30')).toBeInTheDocument();

    const delayHandler = getWebSocketHandler('delay_triggered');
    expect(delayHandler).toBeDefined();

    act(() => {
      delayHandler!({
        auction_id: 999,
        delay_duration: 30,
        new_end_time: baseNow + 180_000,
        remaining_delay: 60,
        max_delay: 90,
      });
    });

    expect(screen.getByText('00:30')).toBeInTheDocument();
    expect(screen.queryByText('03:00')).not.toBeInTheDocument();
    expect(mockShowGlobalToast).not.toHaveBeenCalledWith(expect.objectContaining({
      title: '触发防狙击',
    }));

    dateNowSpy.mockRestore();
  });

  it('updates end_time from time_sync without showing antisnipe toast', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 2,
      current_price: 1200,
      end_time: new Date(baseNow + 30_000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('00:30')).toBeInTheDocument();

    const timeSyncHandler = getWebSocketHandler('time_sync');
    expect(timeSyncHandler).toBeDefined();

    act(() => {
      timeSyncHandler!({
        server_time: baseNow,
        end_time: baseNow + 120_000,
      });
    });

    expect(await screen.findByText('02:00')).toBeInTheDocument();
    expect(mockShowGlobalToast).not.toHaveBeenCalledWith(expect.objectContaining({
      title: '触发防狙击',
    }));

    dateNowSpy.mockRestore();
  });

  it('ignores time_sync from another auction room', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 2,
      current_price: 1200,
      end_time: new Date(baseNow + 30_000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('00:30')).toBeInTheDocument();

    const timeSyncHandler = getWebSocketHandler('time_sync');
    expect(timeSyncHandler).toBeDefined();

    act(() => {
      timeSyncHandler!({
        auction_id: 999,
        server_time: baseNow,
        end_time: baseNow + 120_000,
      });
    });

    expect(screen.getByText('00:30')).toBeInTheDocument();
    expect(screen.queryByText('02:00')).not.toBeInTheDocument();

    dateNowSpy.mockRestore();
  });

  it('opens sheet via URL push and clears it on bid success', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    // 打开「出价」面板 → URL 写入 sheet=bid
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    await waitFor(() => expect(screen.getByTestId('location-search')).toHaveTextContent('sheet=bid'));

    // 出价成功 → URL 去除 sheet
    fireEvent.click(screen.getByRole('button', { name: /立即出价/ }));
    await waitFor(() => expect(mockedBidApi.placeBid).toHaveBeenCalledWith(5, 1300));
    await waitFor(() => expect(screen.getByTestId('location-search')).not.toHaveTextContent('sheet'));
  });

  it('shows a pending settlement summary when an active-status auction has reached end_time', async () => {
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(Date.now() - 1000).toISOString(),
    });
    mockedUseFixedPriceItems.mockReturnValue({
      items: [{
        id: 7001,
        auction_id: 5,
        product_brief: { id: 8001, title: '一口价翡翠', cover_image: '/fp.jpg' },
        price: '99.00',
        total_stock: 10,
        remaining_stock: 10,
        status: 'on_sale',
      }],
      byId: {},
      socket: null,
      latestListedItem: null,
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect(await screen.findByText('本场竞拍已结束')).toBeInTheDocument();
    expect(screen.getByText('结算中...')).toBeInTheDocument();
    expect(screen.queryByText('流拍')).not.toBeInTheDocument();
    expect(screen.queryByText(/成交价/)).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: '查看竞拍结果' })).toHaveAttribute('href', '/result?id=5');
    expect(screen.queryByTestId('bid-dock')).not.toBeInTheDocument();
    expect(screen.queryByTestId('chat-panel')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('一口价商品列表')).not.toBeInTheDocument();
    expect(screen.queryByRole('article', { name: /一口价翡翠 一口价商品/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '出价' })).not.toBeInTheDocument();
    expect(mockedBidApi.placeBid).not.toHaveBeenCalled();
  });

  it('switches to ended summary when the backend auction_end event arrives', async () => {
    mockedUseFixedPriceItems.mockReturnValue({
      items: [{
        id: 7001,
        auction_id: 5,
        product_brief: { id: 8001, title: '一口价翡翠', cover_image: '/fp.jpg' },
        price: '99.00',
        total_stock: 10,
        remaining_stock: 10,
        status: 'on_sale',
      }],
      byId: {},
      socket: null,
      latestListedItem: null,
    });
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    expect(screen.getByTestId('bid-dock')).toBeInTheDocument();
    expect(screen.getByTestId('chat-panel')).toBeInTheDocument();
    expect(screen.getByRole('article', { name: /一口价翡翠 一口价商品/ })).toBeInTheDocument();

    const auctionEndHandler = getWebSocketHandler('auction_end');
    expect(auctionEndHandler).toBeDefined();

    act(() => {
      auctionEndHandler!({
        auction_id: 5,
        winner_id: 9,
        final_price: '1300.00',
      });
    });

    expect(await screen.findByText('本场竞拍已结束')).toBeInTheDocument();
    expect(screen.queryByTestId('bid-dock')).not.toBeInTheDocument();
    expect(screen.queryByTestId('chat-panel')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('一口价商品列表')).not.toBeInTheDocument();
    expect(screen.getByText('成交价 ¥1,300')).toBeInTheDocument();
  });

  it('renders UnsoldAnimation when auction_end event arrives without a winner', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    const auctionEndHandler = getWebSocketHandler('auction_end');
    expect(auctionEndHandler).toBeDefined();

    act(() => {
      auctionEndHandler!({
        auction_id: 5,
        winner_id: 0,
        final_price: '800.00',
      });
    });

    const unsoldAnim = await screen.findByTestId('unsold-animation');
    expect(unsoldAnim).toBeInTheDocument();
    expect(within(unsoldAnim).getByText('遗憾流拍')).toBeInTheDocument();
    expect(screen.getByText('流拍')).toBeInTheDocument();
    expect(screen.queryByText(/成交价/)).not.toBeInTheDocument();
  });

  it('does not infer unsold state when the local countdown reaches zero before backend end event', async () => {
    jest.useFakeTimers();
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    let nowMs = baseNow;
    const dateNowSpy = jest.spyOn(Date, 'now').mockImplementation(() => nowMs);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 800,
      winner_id: 0,
      end_time: new Date(baseNow + 1000).toISOString(),
    });
    mockedBidApi.getRanking.mockResolvedValue([]);
    mockedAuctionApi.getBids.mockResolvedValue([]);

    try {
      renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

      expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
      expect(screen.queryByTestId('unsold-animation')).not.toBeInTheDocument();

      await act(async () => {
        nowMs = baseNow + 1000;
        jest.advanceTimersByTime(1000);
      });

      expect(screen.queryByTestId('unsold-animation')).not.toBeInTheDocument();
      expect(screen.getByText('结算中...')).toBeInTheDocument();
      expect(screen.queryByText('流拍')).not.toBeInTheDocument();
      expect(screen.queryByText(/成交价/)).not.toBeInTheDocument();
    } finally {
      dateNowSpy.mockRestore();
      jest.useRealTimers();
    }
  });

  it('repairs mojibake product and room copy in collapsed and expanded states', async () => {
    mockedProductApi.get.mockResolvedValue({
      id: 7,
      name: toUtf8Mojibake('明代紫砂壶'),
      description: toUtf8Mojibake('名家手作孤品'),
      images: ['/product.jpg'],
      rules: { start_price: 800, increment: 100 },
    });
    mockedLiveStreamApi.get.mockResolvedValue({
      id: 3,
      name: toUtf8Mojibake('瓷器珍藏夜场'),
      host_name: '拍卖师王老师',
      viewer_count: 128,
      is_following: false,
      followers_count: 12,
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    expect(screen.getByText('名家手作孤品')).toBeInTheDocument();
    expect(screen.queryByText(/æ|å|ç|è/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId('bid-dock'));

    expect(await screen.findByText('出价排行')).toBeInTheDocument();
    expect(screen.queryByText(/æ|å|ç|è/)).not.toBeInTheDocument();
  });

  it('repairs mojibake ranking user names before rendering', async () => {
    mockedBidApi.getRanking.mockResolvedValue([
      { rank: 1, user_id: 9102, user_name: toUtf8Mojibake('演示买家B'), amount: 1200 },
    ]);

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });
    fireEvent.click(await screen.findByText('明代紫砂壶'));

    expect(await screen.findByText('演示买家B')).toBeInTheDocument();
    expect(screen.queryByText(/æ¼|ç¤|ä¹/)).not.toBeInTheDocument();
  });

  it('toggles haptic feedback and triggers vibration on bid success', async () => {
    const mockVibrate = jest.fn();
    Object.defineProperty(global.navigator, 'vibrate', {
      value: mockVibrate,
      configurable: true,
      writable: true,
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    fireEvent.click(await screen.findByTestId('bid-dock'));
    const bidButton = await screen.findByRole('button', { name: '立即出价' });
    
    act(() => {
      fireEvent.click(bidButton);
    });

    await waitFor(() => {
      expect(mockVibrate).toHaveBeenCalledWith(50);
    });

    const toggleButton = screen.getByText(/🎵/);
    fireEvent.click(toggleButton);

    mockVibrate.mockClear();

    fireEvent.click(await screen.findByTestId('bid-dock'));
    act(() => {
      fireEvent.click(screen.getByRole('button', { name: '立即出价' }));
    });

    await waitFor(() => {
      expect(mockVibrate).not.toHaveBeenCalled();
    });
  });

  it('renders TapBurstHearts and triggers hearts on double click', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    const tapContainer = document.querySelector('[data-testid="tap-burst-hearts"]');
    expect(tapContainer).toBeInTheDocument();
    expect(tapContainer?.children.length).toBe(0);

    fireEvent.doubleClick(document.body);

    await waitFor(() => {
      expect(tapContainer?.children.length).toBeGreaterThan(0);
    });
  });

  it('shows likes in the top-right data island and increments them on double click without showing auction status', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    expect(screen.getByLabelText('点赞数')).toHaveTextContent('0');
    expect(screen.queryByText('正在竞拍')).not.toBeInTheDocument();

    fireEvent.doubleClick(document.body);

    await waitFor(() => {
      expect(screen.getByLabelText('点赞数')).toHaveTextContent('1');
    });
  });

  it('applies urgent color to countdown when time left is less than 10s and shows marquee heat', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(baseNow + 9_000).toISOString(),
    });
    mockedBidApi.getRanking.mockResolvedValue([
      { rank: 1, user_id: 1, user_name: '张三', amount: 1200 },
      { rank: 2, user_id: 2, user_name: '李四', amount: 1100 },
    ]);

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    fireEvent.click(await screen.findByTestId('bid-dock'));

    const countdownEl = await screen.findByText('00:09');
    expect(countdownEl).toHaveClass('countdownUrgentText');

    expect(screen.getByText('战况冷静')).toBeInTheDocument();
    expect(screen.getByText('已有 2 人出价')).toBeInTheDocument();
    expect(screen.getByText('128 人围观')).toBeInTheDocument();

    dateNowSpy.mockRestore();
  });

  it('renders glitch countdown when time left is <= 5s and drawer is closed, but hides it when drawer is open', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(baseNow + 5_000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    
    // Default sheet is null, countdown should be visible
    const glitchEl = await screen.findByTestId('glitch-countdown');
    expect(glitchEl).toBeInTheDocument();
    expect(glitchEl).toHaveTextContent('5');

    // Open drawer
    fireEvent.click(screen.getByTestId('bid-dock'));
    await waitFor(() => {
      expect(screen.queryByTestId('glitch-countdown')).not.toBeInTheDocument();
    });

    dateNowSpy.mockRestore();
  });

  it('triggers intro animation when new fixed price item is added', async () => {
    const fixedPriceItem = {
      id: 7001,
      auction_id: 5,
      product_brief: { id: 8001, title: '一口价翡翠', cover_image: '/fp.jpg' },
      price: '99.00',
      total_stock: 10,
      remaining_stock: 10,
      status: 'on_sale' as const,
    };
    
    // Start with empty items
    mockedUseFixedPriceItems.mockReturnValue({ items: [], byId: {}, socket: null, latestListedItem: null });
    const { rerender } = render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        <LocationDisplay />
      </MemoryRouter>
    );

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    expect(screen.queryByText('一口价翡翠')).not.toBeInTheDocument();

    // Update with new item
    mockedUseFixedPriceItems.mockReturnValue({ items: [fixedPriceItem], byId: { 7001: fixedPriceItem }, socket: null, latestListedItem: null });
    rerender(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        <LocationDisplay />
      </MemoryRouter>
    );

    expect(screen.queryByText('新上架 一口价')).not.toBeInTheDocument();

    mockedUseFixedPriceItems.mockReturnValue({
      items: [fixedPriceItem],
      byId: { 7001: fixedPriceItem },
      socket: null,
      latestListedItem: { item: fixedPriceItem, sequence: 1 },
    });
    rerender(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        <LocationDisplay />
      </MemoryRouter>
    );

    // The animation component should render the badge
    expect(await screen.findByText('新上架 一口价')).toBeInTheDocument();
    expect(screen.queryByLabelText('一口价翡翠 一口价商品')).not.toBeInTheDocument();
    
    // Simulate animation end
    const animationCard = screen.getByText('新上架 一口价').parentElement;
    if (animationCard) {
      const event = new Event('animationend', { bubbles: true });
      Object.defineProperty(event, 'animationName', { value: 'flyToBottomRight_somehash' });
      fireEvent(animationCard, event);
    }

    // The animation should be removed, and it should pulse
    await waitFor(() => {
      expect(screen.queryByText('新上架 一口价')).not.toBeInTheDocument();
    });
    expect(screen.getByLabelText('一口价翡翠 一口价商品')).toBeInTheDocument();
  });

});
