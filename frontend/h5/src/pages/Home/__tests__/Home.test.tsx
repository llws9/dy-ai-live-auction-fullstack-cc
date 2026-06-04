import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import HomePage from '../index';
import { auctionApi, followApi, productApi } from '../../../services/api';
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
}));

jest.mock('../../../services/notification', () => ({
  notificationApi: {
    getUnreadCount: jest.fn(),
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
const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;
const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockTrackEvent = trackEvent as jest.MockedFunction<typeof trackEvent>;

const renderHome = () =>
  render(
    <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <HomePage />
    </MemoryRouter>
  );

const mockAuthAuthenticated = () => {
  mockedUseAuth.mockReturnValue({
    isAuthenticated: true,
    loading: false,
    user: { id: 1, role: 'user' },
    login: jest.fn(),
    logout: jest.fn(),
  } as unknown as ReturnType<typeof useAuth>);
};

const mockAuthAnonymous = () => {
  mockedUseAuth.mockReturnValue({
    isAuthenticated: false,
    loading: false,
    user: null,
    login: jest.fn(),
    logout: jest.fn(),
  } as unknown as ReturnType<typeof useAuth>);
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
});

describe('HomePage 未读消息红点 (T3.6 / F-D2)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0 });
    mockedProductApi.listCategories.mockResolvedValue([]);
    mockedFollowApi.getFollowedLiveStreams.mockResolvedValue({ list: [] });
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 0 });
    mockAuthAuthenticated();
  });

  it('登录后 mount 调用 getUnreadCount 一次', async () => {
    renderHome();
    await waitFor(() => expect(mockedNotificationApi.getUnreadCount).toHaveBeenCalledTimes(1));
  });

  it('未登录时不调用 getUnreadCount，且不渲染 BadgeDot', async () => {
    mockAuthAnonymous();
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 5 });

    renderHome();

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());
    expect(mockedNotificationApi.getUnreadCount).not.toHaveBeenCalled();
    expect(screen.queryByLabelText(/条待处理提醒/)).not.toBeInTheDocument();
  });

  it('count>0 时通知图标右上角渲染未读数红点', async () => {
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 8 });

    renderHome();

    expect(await screen.findByLabelText('8 条待处理提醒')).toBeInTheDocument();
  });

  it('count=0 时不渲染红点', async () => {
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 0 });

    renderHome();

    await waitFor(() => expect(mockedNotificationApi.getUnreadCount).toHaveBeenCalled());
    expect(screen.queryByLabelText(/条待处理提醒/)).not.toBeInTheDocument();
  });

  it('getUnreadCount 失败时仅 console.warn，不渲染红点（容错）', async () => {
    const warnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    mockedNotificationApi.getUnreadCount.mockRejectedValueOnce(new Error('network down'));

    renderHome();

    await waitFor(() => expect(mockedNotificationApi.getUnreadCount).toHaveBeenCalled());
    expect(screen.queryByLabelText(/条待处理提醒/)).not.toBeInTheDocument();
    expect(warnSpy).toHaveBeenCalled();
    warnSpy.mockRestore();
  });

  it('页面回到前台（visibilitychange → visible）时再次拉取未读数', async () => {
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 1 });

    renderHome();
    await waitFor(() => expect(mockedNotificationApi.getUnreadCount).toHaveBeenCalledTimes(1));

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true });
    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'));
    });

    await waitFor(() => expect(mockedNotificationApi.getUnreadCount).toHaveBeenCalledTimes(2));
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
