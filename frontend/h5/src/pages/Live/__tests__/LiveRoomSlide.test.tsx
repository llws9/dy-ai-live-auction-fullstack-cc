import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import LiveRoomSlide from '../LiveRoomSlide';
import { auctionApi, bidApi, followApi, liveStreamApi, productApi } from '../../../services/api';
import WebSocketService from '../../../services/websocket';
import { useFixedPriceItems } from '../../../hooks/useFixedPriceItems';

const mockShowGlobalToast = jest.fn();
const mockNavigate = jest.fn();
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
const mockedUseFixedPriceItems = useFixedPriceItems as jest.MockedFunction<typeof useFixedPriceItems>;

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

describe('LiveRoomSlide', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockWebSocketInstance.connect.mockResolvedValue(undefined);
    mockWebSocketInstance.onNotification.mockReturnValue(jest.fn());
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

  it('places a bid and refreshes ranking', async () => {
    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    fireEvent.click(screen.getByRole('button', { name: /立即出价/ }));
    await waitFor(() => expect(mockedBidApi.placeBid).toHaveBeenCalledWith(5, 1300));
    // 出价成功后 sheet 自动收起，重新展开以校验排行已刷新
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('测试用户')).toBeInTheDocument();
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

    const bidHandler = mockWebSocketInstance.on.mock.calls.find((c) => c[0] === 'bid_placed')?.[1] as
      | ((data: any) => void)
      | undefined;
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

  it('disables bid actions when an active-status auction has reached end_time', async () => {
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(Date.now() - 1000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('已结束')).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: '已结束' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: '出价' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('当前最高价 ¥1,200').closest('div')!);
    expect(await screen.findByText('00:00')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '竞拍已结束' })).toBeDisabled();
    expect(mockedBidApi.placeBid).not.toHaveBeenCalled();
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

    fireEvent.click(screen.getByText('明代紫砂壶'));

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(1);
    expect(screen.getAllByText('名家手作孤品').length).toBeGreaterThan(1);
  });
});
