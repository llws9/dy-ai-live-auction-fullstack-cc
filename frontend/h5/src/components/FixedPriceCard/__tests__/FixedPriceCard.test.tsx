import { fireEvent, render, screen } from '@testing-library/react';
import FixedPriceCard from '../index';
import type { FixedPriceItem } from '../../../api/fixedPrice';

const item: FixedPriceItem = {
  id: 7001,
  product_id: 5001,
  price: '99.00',
  total_stock: 100,
  remaining_stock: 87,
  status: 'live',
  product_brief: {
    id: 5001,
    title: '翡翠手镯',
    cover_image: 'cdn://a.jpg',
  },
};

describe('FixedPriceCard', () => {
  it('显示商品标题、价格和剩余库存', () => {
    render(<FixedPriceCard item={item} onPurchase={() => {}} />);

    expect(screen.getByText('翡翠手镯')).toBeInTheDocument();
    expect(screen.getByText('¥99.00')).toBeInTheDocument();
    expect(screen.getByText(/剩.*87.*100/)).toBeInTheDocument();
  });

  it('点击 live 状态按钮触发 onPurchase', () => {
    const onPurchase = jest.fn();

    render(<FixedPriceCard item={item} onPurchase={onPurchase} />);
    fireEvent.click(screen.getByRole('button', { name: /立即抢/ }));

    expect(onPurchase).toHaveBeenCalledWith(7001);
  });

  it('sold_out 状态按钮禁用并显示已售罄', () => {
    render(
      <FixedPriceCard
        item={{ ...item, status: 'sold_out', remaining_stock: 0 }}
        onPurchase={() => {}}
      />
    );

    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent('已售罄');
  });

  it('offline 状态按钮禁用并显示已下架', () => {
    render(<FixedPriceCard item={{ ...item, status: 'offline' }} onPurchase={() => {}} />);

    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent('已下架');
  });

  it('缺少封面图时使用商品标题作为可访问名称', () => {
    render(
      <FixedPriceCard
        item={{ ...item, product_brief: { id: 5001, title: '无图商品' } }}
        onPurchase={() => {}}
      />
    );

    expect(screen.getByRole('img', { name: '无图商品' })).toBeInTheDocument();
  });
});
