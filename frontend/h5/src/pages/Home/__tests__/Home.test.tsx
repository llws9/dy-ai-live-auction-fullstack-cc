import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import HomePage from '../index';
import { auctionApi, productApi } from '../../../services/api';
import { notificationApi } from '../../../services/notification';
import { useAuth } from '../../../store/authContext';

jest.mock('../../../services/api', () => ({
  auctionApi: {
    list: jest.fn(),
  },
  productApi: {
    get: jest.fn(),
    listCategories: jest.fn(),
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

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;
const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;
const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

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
});

describe('HomePage 未读消息红点 (T3.6 / F-D2)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0 });
    mockedProductApi.listCategories.mockResolvedValue([]);
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
});
