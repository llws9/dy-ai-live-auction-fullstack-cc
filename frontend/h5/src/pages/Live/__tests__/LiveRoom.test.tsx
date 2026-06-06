import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveRoomSlide from '../LiveRoomSlide';
import { fetchMyPurchase, purchase } from '../../../api/fixedPrice';
import { useFixedPriceItems } from '../../../hooks/useFixedPriceItems';
import { auctionApi, bidApi, followApi, liveStreamApi, productApi, skyLampApi } from '../../../services/api';
import WebSocketService from '../../../services/websocket';
import { DemoProvider } from '../../../store/demoContext';
import { useLiveChatStore } from '../../../store/liveChatStore';

const mockShowGlobalToast = jest.fn();
const mockNavigate = jest.fn();
const mockWebSocketInstance = {
  on: jest.fn(),
  off: jest.fn(),
  sendChat: jest.fn(() => true),
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
  skyLampApi: {
    listSubscriptions: jest.fn(),
    startSubscription: jest.fn(),
  },
}));

jest.mock('../../../services/websocket', () => ({
  __esModule: true,
  default: jest.fn(() => mockWebSocketInstance),
}));

jest.mock('../../../hooks/useFixedPriceItems', () => ({
  useFixedPriceItems: jest.fn(),
}));

jest.mock('../../../api/fixedPrice', () => ({
  fetchMyPurchase: jest.fn(),
  generateIdempotencyKey: jest.fn(() => 'idem-live-page-001'),
  purchase: jest.fn(),
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
    user: { id: 9, name: '测试用户', role: 0 },
    token: 'token-1',
    loading: false,
  }),
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
const mockedFollowApi = followApi as jest.Mocked<typeof followApi>;
const mockedLiveStreamApi = liveStreamApi as jest.Mocked<typeof liveStreamApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;
const mockedSkyLampApi = skyLampApi as jest.Mocked<typeof skyLampApi>;
const mockedUseFixedPriceItems = useFixedPriceItems as jest.MockedFunction<typeof useFixedPriceItems>;
const mockedFetchMyPurchase = fetchMyPurchase as jest.MockedFunction<typeof fetchMyPurchase>;
const mockedPurchase = purchase as jest.MockedFunction<typeof purchase>;
const MockedWebSocketService = WebSocketService as jest.MockedClass<typeof WebSocketService>;

describe('LiveRoom migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockWebSocketInstance.connect.mockResolvedValue(undefined);
    mockWebSocketInstance.onNotification.mockReturnValue(jest.fn());
    mockWebSocketInstance.on.mockImplementation(() => undefined);
    mockWebSocketInstance.off.mockImplementation(() => undefined);
    mockedUseFixedPriceItems.mockReturnValue({ items: [], byId: {}, socket: null });

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
    mockedFollowApi.followLiveStream.mockResolvedValue({});
    mockedFollowApi.unfollowLiveStream.mockResolvedValue({});
    mockedFollowApi.getFollowersStats.mockResolvedValue({ count: 12 });
    mockedFollowApi.getFollowStatus.mockResolvedValue({ is_following: false });
    mockedSkyLampApi.listSubscriptions.mockResolvedValue({ subscriptions: [] } as any);
    mockedSkyLampApi.startSubscription.mockResolvedValue({} as any);
    mockedFetchMyPurchase.mockResolvedValue({ i_bought: false });
    mockedPurchase.mockResolvedValue({
      purchase_id: 88,
      item_id: 7001,
      price: '88.00',
      remaining_stock: 4,
      status: 'success',
    });
  });

  it('loads live auction data and wires bid/follow actions', async () => {
    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    expect(screen.getByText('拍卖师王老师')).toBeInTheDocument();

    // 排行与出价/收藏按钮均在 sheet 内，先展开 sheet
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(screen.getByText('张三')).toBeInTheDocument();

    expect(mockedLiveStreamApi.get).toHaveBeenCalledWith(3);
    expect(mockedAuctionApi.get).toHaveBeenCalledWith(5);
    expect(mockedProductApi.get).toHaveBeenCalledWith(7);
    expect(mockedBidApi.getRanking).toHaveBeenCalledWith(5, 10);

    fireEvent.click(screen.getByRole('button', { name: /收藏/ }));
    await waitFor(() => expect(mockedFollowApi.followLiveStream).toHaveBeenCalledWith(3));

    fireEvent.click(screen.getByRole('button', { name: /立即出价/ }));
    await waitFor(() => expect(mockedBidApi.placeBid).toHaveBeenCalledWith(5, 1300));
    // 出价成功后 sheet 自动收起，重新展开以校验排行已刷新
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('我自己 (当前领先)')).toBeInTheDocument();
    expect(screen.getAllByText('¥1,300').length).toBeGreaterThan(0);
  });

  it('shows a bid success flair after normal bid succeeds and closes the bid sheet', async () => {
    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    fireEvent.click(await screen.findByRole('button', { name: '出价' }));
    fireEvent.click(screen.getByRole('button', { name: /立即出价/ }));

    await waitFor(() => expect(mockedBidApi.placeBid).toHaveBeenCalledWith(5, 1300));
    await waitFor(() => expect(screen.queryByLabelText('收起竞拍面板')).not.toBeInTheDocument());

    const bidFlair = await screen.findByTestId('bid-success-flair');
    expect(bidFlair).toHaveTextContent('测试用户 刚刚出价');
    expect(bidFlair).toHaveTextContent('¥1,300');
  });

  it('does not show the normal bid flair when starting sky lamp', async () => {
    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    fireEvent.click(await screen.findByRole('button', { name: '出价' }));
    fireEvent.click(screen.getByRole('button', { name: /点天灯/ }));
    fireEvent.click(screen.getByRole('button', { name: /确认开启/ }));

    await waitFor(() => expect(mockedSkyLampApi.startSubscription).toHaveBeenCalledWith(5));
    expect(screen.queryByTestId('bid-success-flair')).not.toBeInTheDocument();
  });

  it('uses follow-status endpoint as the authoritative is_following source when logged in', async () => {
    // 详情接口占位返回 false（后端 T2.4 当前固定 false）
    mockedLiveStreamApi.get.mockResolvedValue({
      id: 3,
      name: '瓷器珍藏夜场',
      host_name: '拍卖师王老师',
      viewer_count: 128,
      is_following: false,
      followers_count: 12,
    });
    // 权威接口返回 true，应覆盖详情占位
    mockedFollowApi.getFollowStatus.mockResolvedValue({ is_following: true });

    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    await waitFor(() => expect(mockedFollowApi.getFollowStatus).toHaveBeenCalledWith(3));

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    // 按钮文案应为「已收藏」，点击触发取消收藏
    const followBtn = await screen.findByRole('button', { name: /已收藏/ });
    fireEvent.click(followBtn);
    await waitFor(() => expect(mockedFollowApi.unfollowLiveStream).toHaveBeenCalledWith(3));
  });

  it('maps notification websocket messages to global toast and removes demo triggers', async () => {
    const unsubscribeNotification = jest.fn();
    let notificationHandler: ((notification: any) => void) | undefined;
    mockWebSocketInstance.onNotification.mockImplementation((handler) => {
      notificationHandler = handler;
      return unsubscribeNotification;
    });

    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    await waitFor(() => expect(MockedWebSocketService).toHaveBeenCalledWith(5, 'token-1', 3));
    expect(mockWebSocketInstance.onNotification).toHaveBeenCalledTimes(1);
    expect(screen.queryByLabelText('触达 Toast 测试')).not.toBeInTheDocument();

    notificationHandler?.({
      id: 101,
      type: 'bid_outbid',
      title: '您已被超价',
      content: '当前最高价已更新',
    });
    notificationHandler?.({
      id: 101,
      type: 'bid_outbid',
      title: '您已被超价',
      content: '重复消息',
    });

    expect(mockShowGlobalToast).toHaveBeenCalledTimes(1);
    expect(mockShowGlobalToast).toHaveBeenCalledWith(expect.objectContaining({
      type: 'danger',
      title: '您已被超价',
      message: '当前最高价已更新',
      actionText: '重新出价',
    }));
  });

  it('navigates auction won toast action to result with notification auction id', async () => {
    let notificationHandler: ((notification: any) => void) | undefined;
    mockWebSocketInstance.onNotification.mockImplementation((handler) => {
      notificationHandler = handler;
      return jest.fn();
    });

    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    await waitFor(() => expect(mockWebSocketInstance.onNotification).toHaveBeenCalledTimes(1));

    notificationHandler?.({
      id: 201,
      type: 'auction_won',
      title: '恭喜中标',
      content: '请尽快完成支付',
      data: { auction_id: 99 },
    });

    const toastConfig = mockShowGlobalToast.mock.calls[0][0];
    expect(toastConfig).toEqual(expect.objectContaining({
      type: 'success',
      title: '恭喜中标',
      message: '请尽快完成支付',
      actionText: '去支付',
    }));

    toastConfig.onAction();
    expect(mockNavigate).toHaveBeenCalledWith('/result?id=99');
  });

  it('falls back to current auction id for auction won toast action', async () => {
    let notificationHandler: ((notification: any) => void) | undefined;
    mockWebSocketInstance.onNotification.mockImplementation((handler) => {
      notificationHandler = handler;
      return jest.fn();
    });

    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    await waitFor(() => expect(mockWebSocketInstance.onNotification).toHaveBeenCalledTimes(1));

    notificationHandler?.({
      id: 202,
      type: 'auction_won',
      title: '恭喜中标',
      content: '请尽快完成支付',
      data: {},
    });

    const toastConfig = mockShowGlobalToast.mock.calls[0][0];
    toastConfig.onAction();
    expect(mockNavigate).toHaveBeenCalledWith('/result?id=5');
  });

  it('mounts fixed-price cards, purchase modal, and flair without treating purchase voucher as order id', async () => {
    let fixedPriceFlairHandler: ((message: any) => void) | undefined;
    const fixedPriceSocket = {
      on: jest.fn((type: string, handler: (message: any) => void) => {
        if (type === 'fixed_price_flair') {
          fixedPriceFlairHandler = handler;
        }
      }),
      off: jest.fn(),
    };
    mockWebSocketInstance.on.mockImplementation((type: string) => {
      if (type === 'fixed_price_flair') {
        throw new Error('fixed-price flair must use liveStreamId socket, not auction socket');
      }
    });
    mockedUseFixedPriceItems.mockReturnValue({
      items: [{
        id: 7001,
        product_id: 5001,
        price: '88.00',
        total_stock: 10,
        remaining_stock: 5,
        status: 'on_sale',
        product_brief: { id: 5001, title: '一口价翡翠', cover_image: '/fp.jpg' },
      }],
      byId: {},
      socket: fixedPriceSocket,
    });
    mockedPurchase.mockResolvedValue({
      order_id: 88,
      item_id: 7001,
      price: '88.00',
      remaining_stock: 4,
      status: 'success',
    });

    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    expect(await screen.findByText('一口价翡翠')).toBeInTheDocument();
    expect(mockedUseFixedPriceItems).toHaveBeenCalledWith(3);

    fireEvent.click(screen.getByRole('button', { name: /立即抢/ }));
    expect(await screen.findByRole('dialog', { name: /确认抢购/ })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));
    await waitFor(() => expect(mockedPurchase).toHaveBeenCalledWith({
      itemId: 7001,
      idempotencyKey: 'idem-live-page-001',
    }));
    expect(mockNavigate).not.toHaveBeenCalledWith('/order/88');
    await waitFor(() => expect(screen.queryByRole('dialog', { name: /确认抢购/ })).not.toBeInTheDocument());
    const purchasedButton = screen.getByRole('button', { name: /已购买/ });
    expect(purchasedButton).toBeDisabled();
    fireEvent.click(purchasedButton);
    expect(mockedPurchase).toHaveBeenCalledTimes(1);

    expect(fixedPriceSocket.on).toHaveBeenCalledWith('fixed_price_flair', expect.any(Function));
    act(() => {
      fixedPriceFlairHandler?.({
        buyer_nickname: 'Alice',
        product_title: '一口价翡翠',
        price: '88.00',
      });
    });
    expect(await screen.findByText('Alice')).toBeInTheDocument();
  });

  it('hydrates purchased fixed-price items on page load to prevent repeat purchase requests', async () => {
    mockedUseFixedPriceItems.mockReturnValue({
      items: [{
        id: 7001,
        product_id: 5001,
        price: '88.00',
        total_stock: 10,
        remaining_stock: 5,
        status: 'on_sale',
        product_brief: { id: 5001, title: '一口价翡翠', cover_image: '/fp.jpg' },
      }],
      byId: {},
      socket: null,
    });
    mockedFetchMyPurchase.mockResolvedValue({
      i_bought: true,
      purchase_id: 88,
      price: '88.00',
      created_at: '2026-06-04T10:00:00Z',
    });

    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    expect(await screen.findByText('一口价翡翠')).toBeInTheDocument();
    await waitFor(() => expect(mockedFetchMyPurchase).toHaveBeenCalledWith(7001));
    const purchasedButton = await screen.findByRole('button', { name: /已购买/ });
    expect(purchasedButton).toBeDisabled();

    fireEvent.click(purchasedButton);
    expect(mockedPurchase).not.toHaveBeenCalled();
  });

  it('mounts ChatPanel, dispatches chat_message into store and sends via sendChat', async () => {
    useLiveChatStore.getState().reset();
    const chatHandlers: Record<string, (data: any) => void> = {};
    mockWebSocketInstance.on.mockImplementation((type: string, handler: (data: any) => void) => {
      chatHandlers[type] = handler;
    });

    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <DemoProvider>
          <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
        </DemoProvider>
      </MemoryRouter>
    );

    fireEvent.click(await screen.findByText('明代紫砂壶'));

    await waitFor(() => expect(chatHandlers['chat_message']).toBeDefined());

    act(() => {
      chatHandlers['chat_message']({
        live_stream_id: 3,
        user_id: 2,
        user_name: '王五',
        text: '主播好',
        sent_at: Date.now(),
      });
    });
    expect(await screen.findByText('王五')).toBeInTheDocument();
    expect(screen.getByText('主播好')).toBeInTheDocument();

    const input = screen.getByPlaceholderText(/说点什么/);
    fireEvent.change(input, { target: { value: '出价加油' } });
    fireEvent.click(screen.getByRole('button', { name: /发送/ }));
    expect(mockWebSocketInstance.sendChat).toHaveBeenCalledWith('出价加油', expect.any(String));
  });
});
