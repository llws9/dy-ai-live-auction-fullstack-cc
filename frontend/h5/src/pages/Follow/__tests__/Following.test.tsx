import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import FollowPage from '../index';
import { followApi } from '../../../services/api';

const mockNavigate = jest.fn();

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
}));

jest.mock('../../../services/api', () => ({
  followApi: {
    getFollowedLiveStreams: jest.fn(),
    unfollowLiveStream: jest.fn(),
  },
}));

jest.mock('../../../store/authContext', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    user: { id: 9, name: '测试用户', role: 0 },
    token: 'token-1',
    loading: false,
  }),
}));

jest.mock('../../../components/ThemeToggle', () => ({
  __esModule: true,
  default: () => null,
}));

const mockedFollowApi = followApi as jest.Mocked<typeof followApi>;

describe('Following migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    mockedFollowApi.getFollowedLiveStreams.mockResolvedValue({
      list: [
        {
          id: 31,
          name: '苏州玉器直播间',
          title: '苏州玉器专场',
          description: '今晚 8 点开拍',
          creator_name: '林掌柜',
          host_name: '林掌柜',
          status: 'active',
          current_auctions_count: 3,
          followers_count: 128,
          viewer_count: 456,
        },
        {
          id: 32,
          name: '海派古董直播间',
          title: '海派古董夜拍',
          description: '精选古董拍卖',
          creator_name: '陈老师',
          status: 'inactive',
          current_auctions_count: 0,
          followers_count: 64,
        },
      ],
    });
    mockedFollowApi.unfollowLiveStream.mockResolvedValue({ success: true });
  });

  it('loads followed live streams from followApi instead of all live streams', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <FollowPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('我的收藏')).toBeInTheDocument();
    expect(screen.getByText('苏州玉器专场')).toBeInTheDocument();
    expect(screen.getByText('林掌柜')).toBeInTheDocument();
    expect(screen.getByText('456 观看')).toBeInTheDocument();
    expect(screen.getByText('2 个收藏')).toBeInTheDocument();

    await waitFor(() => expect(mockedFollowApi.getFollowedLiveStreams).toHaveBeenCalledWith(1, 20));
  });

  it('unfollows a stream in place and enters live room with the new query route', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <FollowPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('苏州玉器专场')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '取消收藏 苏州玉器专场' }));

    await waitFor(() => expect(mockedFollowApi.unfollowLiveStream).toHaveBeenCalledWith(31));
    expect(screen.queryByText('苏州玉器专场')).not.toBeInTheDocument();
    expect(screen.getByText('1 个收藏')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '进入直播间 海派古董夜拍' }));

    expect(mockNavigate).toHaveBeenCalledWith('/live?id=32');
  });

  it('renders backend followed stream fields without visible undefined suffixes', async () => {
    mockedFollowApi.getFollowedLiveStreams.mockResolvedValueOnce({
      items: [
        {
          live_stream_id: 880301,
          live_stream_name: '主播直播间',
          status: 1,
          viewer_count: 0,
          auction_count: 0,
        },
      ],
    });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <FollowPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('主播直播间')).toBeInTheDocument();
    expect(screen.queryByText(/#undefined/)).not.toBeInTheDocument();

    expect(screen.getByRole('button', { name: '进入直播间 主播直播间' })).toHaveTextContent(/^进入直播间$/);
    expect(screen.getByRole('button', { name: '取消收藏 主播直播间' })).toHaveTextContent(/^取消收藏$/);

    fireEvent.click(screen.getByRole('button', { name: '进入直播间 主播直播间' }));
    expect(mockNavigate).toHaveBeenCalledWith('/live?id=880301');
  });
});
