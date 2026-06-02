import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveFeedPage from '../LiveFeedPage';
import { liveStreamApi } from '@/services/api';

jest.mock('@/services/api', () => ({
  liveStreamApi: {
    list: jest.fn(),
  },
}));

const mockedLiveStreamApi = liveStreamApi as jest.Mocked<typeof liveStreamApi>;

const renderFeed = (entry: string) =>
  render(
    <MemoryRouter initialEntries={[entry]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <LiveFeedPage />
    </MemoryRouter>
  );

describe('LiveFeedPage feed 骨架', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedLiveStreamApi.list.mockResolvedValue({
      list: [
        { id: 3, name: '房间A', current_auction_id: 11 },
        { id: 4, name: '房间B', current_auction_id: 12 },
      ],
      total: 2,
      page: 1,
      page_size: 20,
    });
  });

  it('按 URL id 初始定位到对应房间（id=4 → 房间B）', async () => {
    renderFeed('/live?id=4');
    expect(await screen.findByText('房间B')).toBeInTheDocument();
  });

  it('无 id 时展示第一个房间（房间A）', async () => {
    renderFeed('/live');
    expect(await screen.findByText('房间A')).toBeInTheDocument();
  });

  it('list 为空时展示空态文案', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 });
    renderFeed('/live');
    await waitFor(() => expect(screen.getByText('暂无直播中房间')).toBeInTheDocument());
  });
});
