import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Addresses from '../index';
import { addressApi } from '../../../services/api';

jest.mock('../../../services/api', () => ({
  addressApi: {
    list: jest.fn(),
    setDefault: jest.fn(),
  },
}));

jest.mock('../../../components/ThemeToggle', () => ({
  __esModule: true,
  default: () => null,
}));

const mockedAddressApi = addressApi as jest.Mocked<typeof addressApi>;

const sampleList = [
  {
    id: 1,
    recipient_name: '林见山',
    phone: '13800138000',
    province: '上海市',
    city: '上海市',
    district: '徐汇区',
    detail: '某路 1 号',
    is_default: true,
  },
  {
    id: 2,
    recipient_name: '陈老师',
    phone: '13900139000',
    province: '北京市',
    city: '北京市',
    district: '海淀区',
    detail: '某路 2 号',
    is_default: false,
  },
];

describe('Addresses page', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders address list with default badge', async () => {
    mockedAddressApi.list.mockResolvedValue({ items: sampleList, total: 2 });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Addresses />
      </MemoryRouter>
    );

    expect(await screen.findByText('林见山')).toBeInTheDocument();
    expect(screen.getByText('陈老师')).toBeInTheDocument();
    expect(screen.getByText('默认')).toBeInTheDocument();
    expect(screen.getByText(/上海市 上海市 徐汇区 某路 1 号/)).toBeInTheDocument();
  });

  it('shows empty state when list is empty', async () => {
    mockedAddressApi.list.mockResolvedValue({ items: [], total: 0 });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Addresses />
      </MemoryRouter>
    );

    expect(await screen.findByText('暂无收货地址')).toBeInTheDocument();
  });

  it('calls setDefault when clicking on a non-default address', async () => {
    mockedAddressApi.list.mockResolvedValue({ items: sampleList, total: 2 });
    mockedAddressApi.setDefault.mockResolvedValue({});

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Addresses />
      </MemoryRouter>
    );

    const setDefaultBtn = await screen.findByRole('button', { name: '设为默认' });
    fireEvent.click(setDefaultBtn);

    await waitFor(() => expect(mockedAddressApi.setDefault).toHaveBeenCalledWith(2));
    // reload triggered after success
    await waitFor(() => expect(mockedAddressApi.list).toHaveBeenCalledTimes(2));
  });
});
