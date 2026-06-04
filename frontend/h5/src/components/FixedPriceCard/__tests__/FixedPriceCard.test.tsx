import { fireEvent, render, screen } from '@testing-library/react';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
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

  it('优先展示接口返回的 product_title，避免降级成泛化标题', () => {
    render(
      <FixedPriceCard
        item={{
          ...item,
          product_brief: undefined,
          product: undefined,
          product_title: '青花瓷茶杯限量装',
        }}
        onPurchase={() => {}}
      />
    );

    expect(screen.getByRole('heading', { name: '青花瓷茶杯限量装' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '一口价商品' })).not.toBeInTheDocument();
  });

  it('标题样式有足够视觉权重', () => {
    const css = readFileSync(join(__dirname, '..', 'index.module.css'), 'utf8');

    expect(css).toMatch(/\.title\s*\{[\s\S]*?color:\s*#111827;/);
    expect(css).toMatch(/\.title\s*\{[\s\S]*?font-size:\s*15px;/);
    expect(css).toMatch(/\.title\s*\{[\s\S]*?font-weight:\s*900;/);
    expect(css).toMatch(/\.title\s*\{[\s\S]*?text-shadow:/);
  });

  it('使用紧凑的直播右下角商品卡布局', () => {
    const css = readFileSync(join(__dirname, '..', 'index.module.css'), 'utf8');
    const cardCss = css.match(/\.card\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const coverCss = css.match(/\.cover,\n\.coverFallback\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const badgeCss = css.match(/\.badge\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const buttonCss = css.match(/\.purchaseButton\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const metaCss = css.match(/\.meta\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const stockCss = css.match(/\.stock\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(cardCss).toContain('grid-template-columns: 46px minmax(0, 1fr);');
    expect(cardCss).toContain('"badge badge"');
    expect(cardCss).toContain('width: 100%;');
    expect(cardCss).toContain('box-sizing: border-box;');
    expect(coverCss).toContain('width: 46px;');
    expect(coverCss).toContain('height: 46px;');
    expect(badgeCss).toContain('grid-area: badge;');
    expect(badgeCss).toContain('width: 100%;');
    expect(badgeCss).toContain('text-align: center;');
    expect(metaCss).toContain('display: grid;');
    expect(stockCss).toContain('max-width: 100%;');
    expect(stockCss).toContain('overflow: hidden;');
    expect(stockCss).toContain('text-overflow: ellipsis;');
    expect(buttonCss).toContain('min-height: 34px;');
    expect(buttonCss).toContain('font-size: 13px;');
  });
});
