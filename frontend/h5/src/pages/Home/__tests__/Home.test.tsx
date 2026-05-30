import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import HomePage from '../index';
import { auctionApi, productApi } from '../../../services/api';

jest.mock('../../../services/api', () => ({
  auctionApi: {
    list: jest.fn(),
  },
  productApi: {
    get: jest.fn(),
    listCategories: jest.fn(),
  },
}));

jest.mock('../../../components/ThemeToggle', () => ({
  __esModule: true,
  default: () => null,
}));

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedProductApi = productApi as jest.Mocked<typeof productApi>;

const renderHome = () =>
  render(
    <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <HomePage />
    </MemoryRouter>
  );

describe('HomePage 分类联动 (T2.10)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0 });
    mockedProductApi.listCategories.mockResolvedValue([
      { id: 1, name: '珠宝腕表' },
      { id: 2, name: '艺术品' },
    ]);
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
});
