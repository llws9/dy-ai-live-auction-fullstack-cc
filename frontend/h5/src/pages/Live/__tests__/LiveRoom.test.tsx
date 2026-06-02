import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveRoomSlide from '../LiveRoomSlide';
import { auctionApi, bidApi, followApi, liveStreamApi, productApi } from '../../../services/api';
import WebSocketService from '../../../services/websocket';

const mockShowGlobalToast = jest.fn();
const mockNavigate = jest.fn();
const mockWebSocketInstance = {
  on: jest.fn(),
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
}));

jest.mock('../../../services/websocket', () => ({
  __esModule: true,
  default: jest.fn(() => mockWebSocketInstance),
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
const MockedWebSocketService = WebSocketService as jest.MockedClass<typeof WebSocketService>;

describe('LiveRoom migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockWebSocketInstance.connect.mockResolvedValue(undefined);
    mockWebSocketInstance.onNotification.mockReturnValue(jest.fn());

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
  });

  it('loads live auction data and wires bid/follow actions', async () => {
    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
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
    expect(await screen.findByText('测试用户')).toBeInTheDocument();
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
        <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
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
        <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
      </MemoryRouter>
    );

    await waitFor(() => expect(MockedWebSocketService).toHaveBeenCalledWith(5, 'token-1'));
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
        <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
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
        <LiveRoomSlide liveStreamId={3} currentAuctionId={5} active />
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
});
