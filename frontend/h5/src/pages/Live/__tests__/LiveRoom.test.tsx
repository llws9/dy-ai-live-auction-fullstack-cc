import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveRoom from '../index';
import { auctionApi, bidApi, followApi, liveStreamApi, productApi } from '../../../services/api';

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
  default: jest.fn().mockImplementation(() => ({
    on: jest.fn(),
    connect: jest.fn().mockResolvedValue(undefined),
    requestSync: jest.fn(),
    disconnect: jest.fn(),
  })),
}));

jest.mock('../../../store/authContext', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    user: { id: 9, name: '测试用户', role: 0 },
    token: 'token-1',
    loading: false,
  }),
}));

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedBidApi = bidApi as jest.Mocked<typeof bidApi>;
const mockedFollowApi = followApi as jest.Mocked<typeof followApi>;
const mockedLiveStreamApi = liveStreamApi as jest.Mocked<typeof liveStreamApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;

describe('LiveRoom migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();

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
  });

  it('loads live auction data and wires bid/follow actions', async () => {
    render(
      <MemoryRouter
        initialEntries={['/live?id=3&auction_id=5']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <LiveRoom />
      </MemoryRouter>
    );

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    expect(screen.getByText('拍卖师王老师')).toBeInTheDocument();
    expect(screen.getByText('张三')).toBeInTheDocument();

    expect(mockedLiveStreamApi.get).toHaveBeenCalledWith(3);
    expect(mockedAuctionApi.get).toHaveBeenCalledWith(5);
    expect(mockedProductApi.get).toHaveBeenCalledWith(7);
    expect(mockedBidApi.getRanking).toHaveBeenCalledWith(5, 10);

    fireEvent.click(screen.getByRole('button', { name: /关注/ }));
    await waitFor(() => expect(mockedFollowApi.followLiveStream).toHaveBeenCalledWith(3));

    fireEvent.click(screen.getByRole('button', { name: /立即出价/ }));
    await waitFor(() => expect(mockedBidApi.placeBid).toHaveBeenCalledWith(5, 1300));
    expect(await screen.findByText('测试用户')).toBeInTheDocument();
  });
});
