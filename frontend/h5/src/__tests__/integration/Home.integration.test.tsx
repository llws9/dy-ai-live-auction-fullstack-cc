import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import HomePage from '@/pages/Home';

// Mock the fetch function
global.fetch = jest.fn();

const mockAuctions = [
  {
    id: 1,
    product_id: 1,
    product_name: '测试商品1',
    product_image: 'https://example.com/image1.jpg',
    status: 1,
    current_price: 100,
    end_time: new Date(Date.now() + 3600000).toISOString(),
    start_time: new Date().toISOString(),
    bidder_count: 10,
  },
  {
    id: 2,
    product_id: 2,
    product_name: '测试商品2',
    product_image: 'https://example.com/image2.jpg',
    status: 1,
    current_price: 200,
    end_time: new Date(Date.now() + 1800000).toISOString(),
    start_time: new Date().toISOString(),
    bidder_count: 5,
  },
];

describe('Home Page Integration', () => {
  beforeEach(() => {
    (fetch as jest.Mock).mockClear();
  });

  it('shows loading state initially', async () => {
    (fetch as jest.Mock).mockImplementation(() =>
      new Promise((resolve) =>
        setTimeout(() =>
          resolve({
            ok: true,
            json: () => Promise.resolve({ auctions: mockAuctions }),
          }),
          100
        )
      )
    );

    render(
      <BrowserRouter>
        <HomePage />
      </BrowserRouter>
    );

    // Should show skeleton loaders
    expect(document.querySelector('.skeleton')).toBeInTheDocument();
  });

  it('loads and displays auction list', async () => {
    (fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ auctions: mockAuctions }),
    });

    render(
      <BrowserRouter>
        <HomePage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('测试商品1')).toBeInTheDocument();
    });

    expect(screen.getByText('测试商品2')).toBeInTheDocument();
    expect(screen.getByText('¥100')).toBeInTheDocument();
    expect(screen.getByText('¥200')).toBeInTheDocument();
  });

  it('displays error state when fetch fails', async () => {
    (fetch as jest.Mock).mockRejectedValue(new Error('Network error'));

    render(
      <BrowserRouter>
        <HomePage />
      </BrowserRouter>
    );

    await waitFor(() => {
      // Should fall back to mock data
      expect(screen.getByText('限定款奢侈品包包')).toBeInTheDocument();
    });
  });

  it('filters auctions by tab', async () => {
    (fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ auctions: mockAuctions }),
    });

    render(
      <BrowserRouter>
        <HomePage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('测试商品1')).toBeInTheDocument();
    });

    // Click "进行中" tab
    const ongoingTab = screen.getByRole('button', { name: '进行中' });
    ongoingTab.click();

    // Both items should still be visible as they are ongoing
    expect(screen.getByText('测试商品1')).toBeInTheDocument();
    expect(screen.getByText('测试商品2')).toBeInTheDocument();
  });

  it('displays empty state when no auctions', async () => {
    (fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ auctions: [] }),
    });

    render(
      <BrowserRouter>
        <HomePage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('暂无竞拍商品')).toBeInTheDocument();
    });
  });

  it('renders navigation elements', async () => {
    (fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ auctions: mockAuctions }),
    });

    render(
      <BrowserRouter>
        <HomePage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('直播竞拍')).toBeInTheDocument();
    });

    expect(screen.getByText('直播间')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /关注/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /历史/ })).toBeInTheDocument();
  });
});
