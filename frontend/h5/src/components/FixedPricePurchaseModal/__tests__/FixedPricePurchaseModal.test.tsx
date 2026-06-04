import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { readFileSync } from 'fs';
import { join } from 'path';
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

  it('成功路径：200 后只提示并回调成功，关闭和跳转由页面层负责', async () => {
    jest.mocked(fixedPriceApi.purchase).mockResolvedValue({
      order_id: 9,
      item_id: 7001,
      price: '99.00',
      remaining_stock: 86,
      status: 'success',
    });

    const { onClose, onSuccess } = renderModal();
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));

    await waitFor(() => expect(onSuccess).toHaveBeenCalledTimes(1));
    expect(screen.getByText('抢到了！')).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
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

  it('成功购买后按钮不再处于 submitting 状态', async () => {
    jest.mocked(fixedPriceApi.purchase).mockResolvedValue({
      order_id: 9,
      item_id: 7001,
      price: '99.00',
      remaining_stock: 86,
      status: 'success',
    });

    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));

    await waitFor(() => expect(screen.getByText('抢到了！')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: /确认抢购/ })).not.toBeDisabled();
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

  it('不在抢购弹窗中泄露直播间 ID', () => {
    renderModal();

    expect(screen.getByText('直播间限时一口价')).toBeInTheDocument();
    expect(screen.queryByText(/1001/)).not.toBeInTheDocument();
  });

  it('在抢购弹窗中展示商品图片', () => {
    renderModal({
      item: {
        ...item,
        product_brief: {
          ...item.product_brief!,
          cover_image: '/jade-bracelet.jpg',
        },
      },
    });

    const image = screen.getByRole('img', { name: '翡翠手镯' });
    expect(image).toHaveAttribute('src', '/jade-bracelet.jpg');
  });

  it('商品无图片时显示兜底图块', () => {
    renderModal();

    expect(screen.getByRole('img', { name: '翡翠手镯' })).toHaveTextContent('无图');
  });

  it('关键抢购信息使用高对比视觉样式', () => {
    const css = readFileSync(join(__dirname, '..', 'index.module.css'), 'utf8');
    const modalCss = css.match(/\.modal\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const productPreviewCss = css.match(/\.productPreview\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const coverCss = css.match(/\.productImage,\n\.productImageFallback\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const titleCss = css.match(/\.title\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const productNameCss = css.match(/\.productName\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const priceLabelCss = css.match(/\.priceLabel\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const purchaseButtonCss = css.match(/\.purchaseButton\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(modalCss).toContain('animation: purchaseModalPop 220ms ease-out;');
    expect(productPreviewCss).toContain('display: grid;');
    expect(productPreviewCss).toContain('grid-template-columns: 72px minmax(0, 1fr);');
    expect(coverCss).toContain('width: 72px;');
    expect(coverCss).toContain('height: 72px;');
    expect(titleCss).toContain('color: #111827;');
    expect(titleCss).toContain('text-shadow: 0 1px 0 rgba(255, 255, 255, 0.75);');
    expect(productNameCss).toContain('color: #7c2d12;');
    expect(productNameCss).toContain('font-weight: 800;');
    expect(priceLabelCss).toContain('color: #9a3412;');
    expect(priceLabelCss).toContain('font-weight: 900;');
    expect(purchaseButtonCss).toContain('min-height: 52px;');
  });
});
