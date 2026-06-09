import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Wallet from '../Index';
import { userApi } from '../../../services/api';

const mockNavigate = jest.fn();

jest.mock('react-router-dom', () => {
  const actual = jest.requireActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

jest.mock('../../../services/api', () => ({
  userApi: {
    getBalance: jest.fn(),
  },
}));

const mockedUserApi = userApi as jest.Mocked<typeof userApi>;

describe('Wallet ledger page', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedUserApi.getBalance.mockResolvedValue({
      balance: '12288',
      frozen_amount: '600',
    });
  });

  it('renders balance, frozen amount and derived ledger rows', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Wallet />
      </MemoryRouter>,
    );

    expect(await screen.findByRole('heading', { name: '钱包' })).toBeInTheDocument();
    expect(screen.getAllByText('¥12,288').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('¥600').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('最近流水')).toBeInTheDocument();
    expect(screen.getByText('订单支付')).toBeInTheDocument();
    expect(screen.getByText('竞拍冻结')).toBeInTheDocument();
    expect(screen.getByText('冻结释放')).toBeInTheDocument();
    expect(screen.getByText('前端派生演示流水')).toBeInTheDocument();
    expect(mockedUserApi.getBalance).toHaveBeenCalledTimes(1);
  });

  it('uses the backend available_amount field as the primary wallet balance', async () => {
    mockedUserApi.getBalance.mockResolvedValueOnce({
      available_amount: '9520.50',
      frozen_amount: '320.00',
      currency: 'CNY',
    });

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Wallet />
      </MemoryRouter>,
    );

    await waitFor(() => expect(screen.getAllByText('¥9,520.5').length).toBeGreaterThanOrEqual(1));
    expect(screen.getAllByText('¥320').length).toBeGreaterThanOrEqual(1);
  });

  it('navigates back when tapping the back button', async () => {
    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Wallet />
      </MemoryRouter>,
    );

    expect(await screen.findByRole('heading', { name: '钱包' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '返回' }));
    expect(mockNavigate).toHaveBeenCalledWith(-1);
  });

  it('shows a retry action when balance loading fails', async () => {
    mockedUserApi.getBalance.mockRejectedValueOnce(new Error('network'));

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Wallet />
      </MemoryRouter>,
    );

    expect(await screen.findByText('钱包信息加载失败')).toBeInTheDocument();

    mockedUserApi.getBalance.mockResolvedValueOnce({
      available: 300,
      frozen_amount: 0,
    });
    fireEvent.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(screen.getAllByText('¥300').length).toBeGreaterThanOrEqual(1));
    expect(mockedUserApi.getBalance).toHaveBeenCalledTimes(2);
  });
});
