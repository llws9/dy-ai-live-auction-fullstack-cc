import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import FixedPricePurchaseModal from '../index';
import * as fixedPriceApi from '../../../api/fixedPrice';
import { ToastProvider } from '../../Toast';

const mockNavigate = jest.fn();

jest.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
}));

jest.mock('../../../api/fixedPrice', () => ({
  generateIdempotencyKey: jest.fn(() => 'idem-fixed-001'),
  purchase: jest.fn(),
}));

const item: fixedPriceApi.FixedPriceItem = {
  id: 7001,
  product_id: 5001,
  price: '99.00',
  total_stock: 100,
  remaining_stock: 87,
  status: 'live',
  product_brief: {
    id: 5001,
    title: '翡翠手镯',
  },
};

function renderModal(props?: Partial<React.ComponentProps<typeof FixedPricePurchaseModal>>) {
  const onClose = jest.fn();
  const onSuccess = jest.fn();
  const onInsufficientBalance = jest.fn();

  render(
    <ToastProvider>
      <FixedPricePurchaseModal
        item={item}
        liveStreamId={1001}
        open={true}
        onClose={onClose}
        onSuccess={onSuccess}
        onInsufficientBalance={onInsufficientBalance}
        {...props}
      />
    </ToastProvider>,
  );

  return { onClose, onSuccess, onInsufficientBalance };
}

describe('FixedPricePurchaseModal', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockNavigate.mockClear();
  });

  it('成功路径：200 后提示、关闭弹窗并跳转订单详情', async () => {
    jest.mocked(fixedPriceApi.purchase).mockResolvedValue({
      order_id: 9,
      item_id: 7001,
      price: '99.00',
      remaining_stock: 86,
      status: 'success',
    });

    const { onClose, onSuccess } = renderModal();
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));

    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(9));
    expect(screen.getByText('抢到了！')).toBeInTheDocument();
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(mockNavigate).toHaveBeenCalledWith('/order/9');
  });

  it('余额不足 402：弹二级确认，点击去充值后触发回调并跳转充值页', async () => {
    jest.mocked(fixedPriceApi.purchase).mockRejectedValue({
      status: 402,
      code: 'INSUFFICIENT_BALANCE',
    });

    const { onInsufficientBalance } = renderModal();
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));

    await screen.findByText('余额不足，去充值');
    expect(onInsufficientBalance).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: /去充值/ }));
    expect(onInsufficientBalance).toHaveBeenCalledTimes(1);
    expect(mockNavigate).toHaveBeenCalledWith('/wallet/recharge');
  });

  it('409 SOLD_OUT：提示已售罄并关闭弹窗', async () => {
    jest.mocked(fixedPriceApi.purchase).mockRejectedValue({
      status: 409,
      code: 'SOLD_OUT',
    });

    const { onClose } = renderModal();
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));

    await screen.findByText('已售罄');
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('409 FP_ALREADY_BOUGHT：提示您已购买过并关闭弹窗', async () => {
    jest.mocked(fixedPriceApi.purchase).mockRejectedValue({
      status: 409,
      code: 'FP_ALREADY_BOUGHT',
    });

    const { onClose } = renderModal();
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));

    await screen.findByText('您已购买过');
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('网络异常：复用同一个 idempotencyKey 自动重试 1 次后提示失败', async () => {
    jest.mocked(fixedPriceApi.purchase).mockRejectedValue(new Error('Network'));

    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));

    await waitFor(() => expect(fixedPriceApi.purchase).toHaveBeenCalledTimes(2));
    expect(fixedPriceApi.purchase).toHaveBeenNthCalledWith(1, {
      itemId: 7001,
      idempotencyKey: 'idem-fixed-001',
    });
    expect(fixedPriceApi.purchase).toHaveBeenNthCalledWith(2, {
      itemId: 7001,
      idempotencyKey: 'idem-fixed-001',
    });
    expect(screen.getByText('网络异常，请稍后重试')).toBeInTheDocument();
  });
});
