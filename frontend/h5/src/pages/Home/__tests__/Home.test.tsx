import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import HomePage from '../index';
import { auctionApi, followApi, productApi, productReminderApi } from '../../../services/api';
import { notificationApi } from '../../../services/notification';
import { useAuth } from '../../../store/authContext';
import { trackEvent } from '../../../utils/trackEvent';

jest.mock('../../../services/api', () => ({
  auctionApi: {
    list: jest.fn(),
  },
  productApi: {
    get: jest.fn(),
    listCategories: jest.fn(),
  },
  followApi: {
    getFollowedLiveStreams: jest.fn(),
  },
  productReminderApi: {
    subscribe: jest.fn(),
    list: jest.fn(),
  },
}));

jest.mock('../../../services/notification', () => ({
  notificationApi: {
    getUnreadCount: jest.fn(),
    getTouchpointSummary: jest.fn(),
    hotPull: jest.fn(),
  },
}));

jest.mock('../../../store/authContext', () => ({
  useAuth: jest.fn(),
}));

jest.mock('../../../components/ThemeToggle', () => ({
  __esModule: true,
  default: () => null,
}));

jest.mock('../../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
  getCountBucket: (count: number) =>
    count <= 0 ? '0' : count === 1 ? '1' : count <= 5 ? '2_5' : count <= 10 ? '6_10' : '10_plus',
}));

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;
const mockedFollowApi = followApi as jest.Mocked<typeof followApi>;
const mockedProductReminderApi = productReminderApi as jest.Mocked<typeof productReminderApi>;
const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;
const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockTrackEvent = trackEvent as jest.MockedFunction<typeof trackEvent>;

const renderHome = () =>
  render(
    <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <HomePage />
    </MemoryRouter>
  );

const createDeferred = <T,>() => {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
};

const mockAuthAuthenticated = () => {
  mockedUseAuth.mockReturnValue({
    isAuthenticated: true,
    loading: false,
    user: { id: 1, role: 'user' },
    token: 'token-1',
    login: jest.fn(),
    logout: jest.fn(),
  } as unknown as ReturnType<typeof useAuth>);
};

const mockAuthAnonymous = () => {
  mockedUseAuth.mockReturnValue({
    isAuthenticated: false,
    loading: false,
    user: null,
    token: null,
    login: jest.fn(),
    logout: jest.fn(),
  } as unknown as ReturnType<typeof useAuth>);
};

const emptyTouchpointSummary = {
  unreadTotal: 0,
  pendingPayment: 0,
  wonNotPaid: 0,
  outbid: 0,
  endingSoon: 0,
};

describe('HomePage 分类联动 (T2.10)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0 });
    mockedProductApi.listCategories.mockResolvedValue([
      { id: 1, name: '珠宝腕表' },
      { id: 2, name: '艺术品' },
    ]);
    mockedFollowApi.getFollowedLiveStreams.mockResolvedValue({ list: [] });
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 0 });
    mockedNotificationApi.getTouchpointSummary.mockResolvedValue(emptyTouchpointSummary);
    mockedNotificationApi.hotPull.mockResolvedValue({ notifications: [], has_more: false });
    mockedProductReminderApi.list.mockResolvedValue({ items: [] });
    mockAuthAuthenticated();
  });

  it('mount 时不传 category_id，渲染从后端拉取的分类 tabs', async () => {
    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());
    const firstCall = mockedAuctionApi.list.mock.calls[0][0] as Record<string, unknown> | undefined;
    expect(firstCall).toEqual(expect.objectContaining({ page: 1, page_size: 20 }));
    expect(firstCall).not.toHaveProperty('category_id');

    await waitFor(() => expect(mockedProductApi.listCategories).toHaveBeenCalled());
    expect(await screen.findByRole('button', { name: '珠宝腕表' })).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: '艺术品' })).toBeInTheDocument();
  });

  it('点击分类 tab 时透传 category_id 调用 auctionApi.list', async () => {
    renderHome();

    const tab = await screen.findByRole('button', { name: '珠宝腕表' });
    fireEvent.click(tab);

    await waitFor(() =>
      expect(mockedAuctionApi.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ category_id: 1, page: 1, page_size: 20 })
      )
    );
  });

  it('listCategories 失败时不阻塞首屏，仍能渲染「全部」tab', async () => {
    mockedProductApi.listCategories.mockRejectedValueOnce(new Error('boom'));

    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());
    expect(screen.getByRole('button', { name: '全部' })).toBeInTheDocument();
  });

  it('过滤与固定 tab 重名的后端分类，避免「收藏」右侧重复渲染「全部」', async () => {
    mockedProductApi.listCategories.mockResolvedValueOnce([
      { id: 0, name: '全部' },
      { id: 1, name: '珠宝腕表' },
    ]);

    renderHome();

    await waitFor(() => expect(mockedProductApi.listCategories).toHaveBeenCalled());
    expect(screen.getAllByRole('button', { name: '全部' })).toHaveLength(1);
    expect(await screen.findByRole('button', { name: '珠宝腕表' })).toBeInTheDocument();
  });

  it('首页头部快捷入口使用 SVG 图形图标而不是文字占位', async () => {
    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());

    const searchAction = screen.getByLabelText('搜索暂未开放');
    const followAction = screen.getByLabelText('我的收藏');
    const notificationAction = screen.getByLabelText('消息通知');

    expect(searchAction.querySelector('svg')).toBeInTheDocument();
    expect(followAction.querySelector('svg')).toBeInTheDocument();
    expect(notificationAction.querySelector('svg')).toBeInTheDocument();
    expect(searchAction).not.toHaveTextContent('搜');
    expect(followAction).not.toHaveTextContent('收');
    expect(notificationAction).not.toHaveTextContent('铃');
  });

  it('修复首页竞拍卡片中的 cp1252 风格中文乱码', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 9,
          product_id: 13,
          live_stream_id: 5,
          status: 1,
          current_price: 3400,
          product: {
            id: 13,
            name: 'è€èœœèœ¡æ‰‹ä¸²',
          },
        },
      ],
      total: 1,
    });

    renderHome();

    await waitFor(() => expect(screen.getByRole('heading', { name: '老蜜蜡手串' })).toBeTruthy());
    expect(screen.queryByText('è€èœœèœ¡æ‰‹ä¸²')).toBeNull();
  });

  it('首页竞拍卡片兼容后端 list 返回的 product.image 首图字段', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 10,
          product_id: 99,
          live_stream_id: 5,
          status: 1,
          current_price: 6800,
          product: {
            id: 99,
            name: '高冰飘花翡翠吊坠',
            image: 'https://example.com/jade-pendant.jpg',
          },
        },
      ],
      total: 1,
    });

    renderHome();

    const image = await screen.findByRole('img', { name: '高冰飘花翡翠吊坠' });
    expect(image).toHaveAttribute('src', 'https://example.com/jade-pendant.jpg');
    expect(screen.queryByText('暂无图片')).not.toBeInTheDocument();
  });

  it('首页无出价竞拍使用起拍价展示价格，不显示 0', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 12,
          product_id: 100,
          live_stream_id: 5,
          status: 1,
          current_price: 0,
          start_price: 100,
          bid_count: 0,
          product: {
            id: 100,
            name: '起拍价拍品',
            images: ['/auction.jpg'],
          },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '起拍价拍品' });
    expect(screen.getByText('暂无出价')).toBeInTheDocument();
    expect(screen.getByText('¥100')).toBeInTheDocument();
    expect(screen.queryByText('¥0')).not.toBeInTheDocument();
  });

  it('首页竞拍卡片在商品无图时使用兜底图片', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 11,
          product_id: 101,
          live_stream_id: 5,
          status: 3,
          current_price: 32112,
          product: {
            id: 101,
            name: '压测拍品 1780733852',
            images: [],
          },
        },
      ],
      total: 1,
    });

    renderHome();

    const image = await screen.findByRole('img', { name: '压测拍品 1780733852' });
    expect(image).toHaveAttribute('src', '/assets/default-auction-cover.svg');
    expect(image).not.toHaveAttribute('src', expect.stringContaining('copilot-cn.bytedance.net'));
    expect(screen.queryByText('暂无图片')).not.toBeInTheDocument();
  });

  it('首页将已过 end_time 的 active 竞拍展示为已结束并隐藏进入直播', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 9,
          product_id: 13,
          live_stream_id: 5,
          status: 1,
          end_time: new Date(Date.now() - 1000).toISOString(),
          current_price: 3400,
          product: {
            id: 13,
            name: '青花瓷摆件',
          },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '青花瓷摆件' });
    expect(await screen.findByText('已结束')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '查看结果' })).toHaveAttribute('href', '/result?id=9');
    expect(screen.queryByRole('link', { name: '进入直播' })).not.toBeInTheDocument();
  });

  it('首页按直播状态优先级展示：直播中 > 即将开始 > 已结束', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 9,
          status: 3,
          current_price: 100,
          end_time: new Date(Date.now() - 60_000).toISOString(),
          product: { id: 9, name: '已结束拍品' },
        },
        {
          id: 8,
          status: 0,
          current_price: 200,
          end_time: new Date(Date.now() + 7_200_000).toISOString(),
          product: { id: 8, name: '即将开始拍品' },
        },
        {
          id: 7,
          status: 1,
          current_price: 300,
          end_time: new Date(Date.now() + 3_600_000).toISOString(),
          product: { id: 7, name: '直播中拍品' },
        },
      ],
      total: 3,
    });

    renderHome();

    const headings = await screen.findAllByRole('heading', { level: 2 });
    expect(headings.map((heading) => heading.textContent)).toEqual([
      '直播中拍品',
      '即将开始拍品',
      '已结束拍品',
    ]);
  });

  it('即将开始竞拍展示详情和订阅，不展示进入直播', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 8,
          product_id: 88,
          live_stream_id: 5,
          status: 0,
          current_price: 399,
          start_time: '2026-06-04T18:39:00+08:00',
          product: { id: 88, name: '手作陶瓷茶具套装' },
        },
      ],
      total: 1,
    });
    mockedProductReminderApi.subscribe.mockResolvedValue({ product_id: 88 });

    renderHome();

    await screen.findByRole('heading', { name: '手作陶瓷茶具套装' });
    expect(await screen.findByText(/^开拍/)).toBeInTheDocument();
    expect(screen.queryByText('0次出价')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: '详情' })).toHaveAttribute('href', '/detail?id=8');
    expect(screen.queryByRole('link', { name: '进入直播' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '订阅' }));

    await waitFor(() => expect(mockedProductReminderApi.subscribe).toHaveBeenCalledWith(88));
    expect(screen.getByRole('button', { name: '已订阅' })).toBeDisabled();
  });

  it('刷新后根据我的商品提醒列表回填已订阅状态', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 8,
          product_id: 88,
          live_stream_id: 5,
          status: 0,
          current_price: 399,
          start_time: '2026-06-05T01:40:00+08:00',
          product: { id: 88, name: '手作陶瓷茶具套装' },
        },
      ],
      total: 1,
    });
    mockedProductReminderApi.list.mockResolvedValueOnce({
      items: [{ product_id: 88 }],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '手作陶瓷茶具套装' });
    expect(await screen.findByRole('button', { name: '已订阅' })).toBeDisabled();
    expect(mockedProductReminderApi.subscribe).not.toHaveBeenCalled();
  });

  it('进行中竞拍继续展示出价次数，已结束竞拍展示成交时间', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 7,
          status: 1,
          current_price: 300,
          bid_count: 2,
          end_time: new Date(Date.now() + 3_600_000).toISOString(),
          product: { id: 7, name: '直播中拍品' },
        },
        {
          id: 9,
          status: 3,
          current_price: 100,
          end_time: '2026-06-04T18:39:00+08:00',
          product: { id: 9, name: '已结束拍品' },
        },
      ],
      total: 2,
    });

    renderHome();

    await screen.findByRole('heading', { name: '直播中拍品' });
    expect(await screen.findByText('2次出价')).toBeInTheDocument();
    expect(screen.getByText(/成交时间/)).toBeInTheDocument();
    expect(screen.queryByText('0次成交')).not.toBeInTheDocument();
  });

  it('已结束且无人中标的竞拍在首页展示流拍而不是成交', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 993566,
          status: 3,
          current_price: '0',
          winner_id: null,
          end_time: '2026-06-08T22:45:39+08:00',
          rules: { start_price: '100' },
          product: { id: 993510, name: '无人出价拍品' },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '无人出价拍品' });
    expect(screen.getByText(/结束时间/)).toBeInTheDocument();
    expect(screen.getByText('流拍')).toBeInTheDocument();
    expect(screen.queryByText('成交')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: '查看结果' })).toHaveAttribute('href', '/result?id=993566');
  });

  it('点击收藏 tab 时复用我的收藏接口渲染收藏直播间', async () => {
    mockedFollowApi.getFollowedLiveStreams.mockResolvedValue({
      list: [
        {
          id: 88,
          title: '翡翠手镯专场',
          host_name: '主播',
          status: 'live',
          cover_image: 'https://example.com/cover.jpg',
          viewer_count: 12,
          followers_count: 1,
        },
      ],
      total: 1,
    });

    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());
    fireEvent.click(screen.getByRole('button', { name: '收藏' }));

    await waitFor(() => expect(mockedFollowApi.getFollowedLiveStreams).toHaveBeenCalledWith(1, 20));
    expect(await screen.findByRole('heading', { name: '翡翠手镯专场' })).toBeInTheDocument();
    expect(screen.queryByText('收藏接口待后端开放后接入。')).not.toBeInTheDocument();
  });

  it('收藏 tab 将没有有效竞拍的直播间展示为已结束并禁止进入直播', async () => {
    mockedFollowApi.getFollowedLiveStreams.mockResolvedValue({
      list: [
        {
          live_stream_id: 88,
          live_stream_name: '翡翠手镯专场',
          host_name: '主播',
          status: 'live',
          auction_count: 0,
          cover_image: 'https://example.com/cover.jpg',
          viewer_count: 12,
          followers_count: 1,
        },
      ],
      total: 1,
    });

    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());
    fireEvent.click(screen.getByRole('button', { name: '收藏' }));

    expect(await screen.findByText('已结束')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '进入直播' })).not.toBeInTheDocument();
  });

  it('收藏 tab 使用后端 live_stream_id/live_stream_name 渲染并进入直播', async () => {
    mockedFollowApi.getFollowedLiveStreams.mockResolvedValue({
      list: [
        {
          live_stream_id: 89,
          live_stream_name: '沉香手串专场',
          host_name: '主播',
          status: 'live',
          auction_count: 1,
          viewer_count: 8,
          followers_count: 2,
        },
      ],
      total: 1,
    });

    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());
    fireEvent.click(screen.getByRole('button', { name: '收藏' }));

    expect(await screen.findByRole('heading', { name: '沉香手串专场' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '进入直播' })).toHaveAttribute('href', '/live?id=89');
  });

  it('点击最热胶囊后以 sort=hot 调用列表接口', async () => {
    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('button', { name: '最热' }));

    await waitFor(() =>
      expect(mockedAuctionApi.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ sort: 'hot' })
      )
    );
  });

  it('忽略过期的筛选请求，避免旧列表覆盖最新筛选结果', async () => {
    const defaultRequest = createDeferred<{ list: unknown[]; total: number }>();
    const hotRequest = createDeferred<{ list: unknown[]; total: number }>();
    mockedAuctionApi.list
      .mockImplementationOnce(() => defaultRequest.promise)
      .mockImplementationOnce(() => hotRequest.promise);

    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole('button', { name: '最热' }));

    await waitFor(() =>
      expect(mockedAuctionApi.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ sort: 'hot' })
      )
    );

    await act(async () => {
      hotRequest.resolve({
        list: [
          {
            id: 2,
            status: 1,
            current_price: 200,
            product: { id: 2, name: '热度结果' },
          },
        ],
        total: 1,
      });
      await hotRequest.promise;
    });

    expect(await screen.findByRole('heading', { name: '热度结果' })).toBeInTheDocument();

    await act(async () => {
      defaultRequest.resolve({
        list: [
          {
            id: 1,
            status: 1,
            current_price: 100,
            product: { id: 1, name: '过期结果' },
          },
        ],
        total: 1,
      });
      await defaultRequest.promise;
    });

    expect(screen.getByRole('heading', { name: '热度结果' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '过期结果' })).not.toBeInTheDocument();
  });

  it('进行中竞拍卡片在封面右下角展示真实观看人数', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 30, product_id: 300, live_stream_id: 5, status: 1,
          current_price: 1000, viewer_count: 128,
          end_time: new Date(Date.now() + 3_600_000).toISOString(),
          product: { id: 300, name: '在线人数拍品', images: ['/a.jpg'] },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '在线人数拍品' });
    expect(screen.getByText(/128\s*观看/)).toBeInTheDocument();
  });

  it('进行中竞拍 viewer_count 为 0 时展示 0 观看', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 31, product_id: 301, live_stream_id: 5, status: 1,
          current_price: 1000, viewer_count: 0,
          end_time: new Date(Date.now() + 3_600_000).toISOString(),
          product: { id: 301, name: '降级拍品', images: ['/a.jpg'] },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '降级拍品' });
    expect(screen.getByText(/0\s*观看/)).toBeInTheDocument();
  });

  it('已结束竞拍即使带 viewer_count 也不展示观看人数', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 32, product_id: 302, status: 3,
          current_price: 1000, viewer_count: 99,
          end_time: new Date(Date.now() - 1000).toISOString(),
          product: { id: 302, name: '结束拍品', images: ['/a.jpg'] },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '结束拍品' });
    expect(screen.queryByText(/观看/)).not.toBeInTheDocument();
  });
});

describe('HomePage 未读消息红点 (T3.6 / F-D2)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0 });
    mockedProductApi.listCategories.mockResolvedValue([]);
    mockedFollowApi.getFollowedLiveStreams.mockResolvedValue({ list: [] });
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 0 });
    mockedNotificationApi.getTouchpointSummary.mockResolvedValue(emptyTouchpointSummary);
    mockedNotificationApi.hotPull.mockResolvedValue({ notifications: [], has_more: false });
    mockedProductReminderApi.list.mockResolvedValue({ items: [] });
    mockAuthAuthenticated();
  });

  it('登录后 mount 热拉商品提醒，并刷新共享通知汇总', async () => {
    mockedNotificationApi.getTouchpointSummary
      .mockResolvedValueOnce(emptyTouchpointSummary)
      .mockResolvedValueOnce({
        ...emptyTouchpointSummary,
        unreadTotal: 1,
      });

    renderHome();
    await waitFor(() => expect(mockedNotificationApi.hotPull).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(mockedNotificationApi.getTouchpointSummary).toHaveBeenCalledTimes(2));
    expect(mockedNotificationApi.getUnreadCount).not.toHaveBeenCalled();
    expect(await screen.findByLabelText('1 条待处理提醒')).toBeInTheDocument();
  });

  it('hot-pull 后使用共享汇总刷新首页通知红点，避免和底部导航不同步', async () => {
    mockedNotificationApi.getTouchpointSummary
      .mockResolvedValueOnce({
        ...emptyTouchpointSummary,
        unreadTotal: 83,
      })
      .mockResolvedValueOnce({
        ...emptyTouchpointSummary,
        unreadTotal: 84,
      });
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 84 });

    renderHome();

    await waitFor(() => expect(mockedNotificationApi.hotPull).toHaveBeenCalledTimes(1));
    expect(await screen.findByLabelText('84 条待处理提醒')).toBeInTheDocument();
    await waitFor(() => expect(mockedNotificationApi.getTouchpointSummary).toHaveBeenCalledTimes(2));
  });

  it('未登录时不调用 getUnreadCount，且不渲染 BadgeDot', async () => {
    mockAuthAnonymous();
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 5 });

    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());
    expect(mockedNotificationApi.getUnreadCount).not.toHaveBeenCalled();
    expect(screen.queryByLabelText(/条待处理提醒/)).not.toBeInTheDocument();
  });

  it('summary unreadTotal>0 时通知图标右上角渲染未读数红点', async () => {
    mockedNotificationApi.getTouchpointSummary.mockResolvedValue({
      ...emptyTouchpointSummary,
      unreadTotal: 8,
    });

    renderHome();

    expect(await screen.findByLabelText('8 条待处理提醒')).toBeInTheDocument();
  });

  it('summary unreadTotal=0 时不渲染红点', async () => {
    mockedNotificationApi.getTouchpointSummary.mockResolvedValue(emptyTouchpointSummary);

    renderHome();

    await waitFor(() => expect(mockedNotificationApi.getTouchpointSummary).toHaveBeenCalled());
    expect(screen.queryByLabelText(/条待处理提醒/)).not.toBeInTheDocument();
  });

  it('getTouchpointSummary 失败时不渲染红点（容错）', async () => {
    mockedNotificationApi.getTouchpointSummary.mockRejectedValueOnce(new Error('network down'));

    renderHome();

    await waitFor(() => expect(mockedNotificationApi.getTouchpointSummary).toHaveBeenCalled());
    expect(screen.queryByLabelText(/条待处理提醒/)).not.toBeInTheDocument();
  });

  it('页面回到前台（visibilitychange → visible）时再次热拉并刷新共享汇总', async () => {
    mockedNotificationApi.getTouchpointSummary
      .mockResolvedValueOnce({
        ...emptyTouchpointSummary,
        unreadTotal: 1,
      })
      .mockResolvedValueOnce({
        ...emptyTouchpointSummary,
        unreadTotal: 1,
      })
      .mockResolvedValueOnce({
        ...emptyTouchpointSummary,
        unreadTotal: 2,
      });

    renderHome();
    await waitFor(() => expect(mockedNotificationApi.hotPull).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(mockedNotificationApi.getTouchpointSummary).toHaveBeenCalledTimes(2));

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true });
    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'));
    });

    await waitFor(() => expect(mockedNotificationApi.hotPull).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(mockedNotificationApi.getTouchpointSummary).toHaveBeenCalledTimes(3));
    expect(await screen.findByLabelText('2 条待处理提醒')).toBeInTheDocument();
  });

  it('点击通知铃铛时记录首页入口点击埋点', async () => {
    renderHome();

    const notificationLink = await screen.findByRole('link', { name: '消息通知' });
    fireEvent.click(notificationLink);

    expect(mockTrackEvent).toHaveBeenCalledWith('entry_clicked', {
      source: 'home',
      entry: 'notification_bell',
      type: 'notification',
      result: 'clicked',
    });
  });
});
