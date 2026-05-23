import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import AuctionPage from '@/pages/Auction';

// Mock the WebSocket service
jest.mock('../../services/websocket', () => {
  return jest.fn().mockImplementation(() => ({
    connect: jest.fn().mockResolvedValue(undefined),
    disconnect: jest.fn(),
    on: jest.fn(),
    requestSync: jest.fn(),
  }));
});

// Mock fetch
global.fetch = jest.fn();

const mockAuction = {
  id: 1,
  product_id: 1,
  status: 1,
  current_price: 150,
  winner_id: null,
  start_time: new Date().toISOString(),
  end_time: new Date(Date.now() + 3600000).toISOString(),
  delay_used: 0,
};

describe('Auction Page Integration', () => {
  beforeEach(() => {
    (fetch as jest.Mock).mockClear();
  });

  it('shows loading state initially', async () => {
    (fetch as jest.Mock).mockImplementation(() =>
      new Promise((resolve) =>
        setTimeout(() =>
          resolve({
            ok: true,
            json: () => Promise.resolve(mockAuction),
          }),
          100
        )
      )
    );

    render(
      <BrowserRouter>
        <Routes>
          <Route path="/auction/:id" element={<AuctionPage />} />
        </Routes>
      </BrowserRouter>
    );

    // Navigate to auction page
    window.history.pushState({}, '', '/auction/1');

    render(
      <BrowserRouter>
        <Routes>
          <Route path="/auction/:id" element={<AuctionPage />} />
        </Routes>
      </BrowserRouter>
    );

    expect(document.querySelector('.loadingContainer')).toBeInTheDocument();
  });

  it('loads and displays auction details', async () => {
    (fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockAuction),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ bids: [] }),
      });

    window.history.pushState({}, '', '/auction/1');

    render(
      <BrowserRouter>
        <Routes>
          <Route path="/auction/:id" element={<AuctionPage />} />
        </Routes>
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('💰 出价竞拍')).toBeInTheDocument();
    });

    expect(screen.getByText('¥150')).toBeInTheDocument();
    expect(screen.getByText('进行中')).toBeInTheDocument();
  });

  it('displays auction info section', async () => {
    (fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockAuction),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ bids: [] }),
      });

    window.history.pushState({}, '', '/auction/1');

    render(
      <BrowserRouter>
        <Routes>
          <Route path="/auction/:id" element={<AuctionPage />} />
        </Routes>
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('📋 竞拍详情')).toBeInTheDocument();
    });

    expect(screen.getByText('竞拍ID')).toBeInTheDocument();
    expect(screen.getByText('状态')).toBeInTheDocument();
    expect(screen.getByText('开始时间')).toBeInTheDocument();
    expect(screen.getByText('结束时间')).toBeInTheDocument();
  });

  it('displays bid records when available', async () => {
    const mockBids = [
      { id: 1, user_id: 2, user_name: '用户A', amount: 150, created_at: new Date().toISOString() },
      { id: 2, user_id: 3, user_name: '用户B', amount: 140, created_at: new Date().toISOString() },
    ];

    (fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockAuction),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ bids: mockBids }),
      });

    window.history.pushState({}, '', '/auction/1');

    render(
      <BrowserRouter>
        <Routes>
          <Route path="/auction/:id" element={<AuctionPage />} />
        </Routes>
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('📊 出价排行')).toBeInTheDocument();
    });

    expect(screen.getByText('用户A')).toBeInTheDocument();
    expect(screen.getByText('用户B')).toBeInTheDocument();
  });

  it('shows error state when auction not found', async () => {
    (fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      status: 404,
    });

    window.history.pushState({}, '', '/auction/999');

    render(
      <BrowserRouter>
        <Routes>
          <Route path="/auction/:id" element={<AuctionPage />} />
        </Routes>
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('竞拍不存在')).toBeInTheDocument();
    });
  });
});
