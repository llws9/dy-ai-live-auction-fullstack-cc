import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { act, render, screen } from '@testing-library/react';
import { BidSuccessAnimation } from '../BidSuccessAnimation';

describe('BidSuccessAnimation', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    act(() => {
      jest.runOnlyPendingTimers();
    });
    jest.useRealTimers();
  });

  it('动画展示完成后触发关闭，即使父组件重渲染替换回调', () => {
    const firstEnd = jest.fn();
    const latestEnd = jest.fn();
    const { rerender } = render(
      <BidSuccessAnimation productName="明代紫砂壶" price={1300} onAnimationEnd={firstEnd} />
    );

    expect(screen.getByTestId('bid-success-animation')).toHaveTextContent('明代紫砂壶');

    act(() => {
      jest.advanceTimersByTime(1000);
    });
    rerender(<BidSuccessAnimation productName="明代紫砂壶" price={1300} onAnimationEnd={latestEnd} />);

    act(() => {
      jest.advanceTimersByTime(2000);
    });

    expect(firstEnd).not.toHaveBeenCalled();
    expect(latestEnd).toHaveBeenCalledTimes(1);
  });

  it('成交动画覆盖层绑定到手机容器而不是浏览器视口', () => {
    const css = readFileSync(join(__dirname, '..', 'bid-success-animation.css'), 'utf8');
    const shakeTriggerCss = css.match(/\.shake-trigger\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const gavelWrapperCss = css.match(/\.gavel-wrapper\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const cardContainerCss = css.match(/\.card-container\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const auctionCardCss = css.match(/\.auction-card\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(shakeTriggerCss).toContain('position: absolute;');
    expect(shakeTriggerCss).not.toContain('position: fixed;');
    expect(gavelWrapperCss).toContain('left: 50%;');
    expect(gavelWrapperCss).toContain('top: 50%;');
    expect(gavelWrapperCss).not.toContain('bottom right');
    expect(css).toContain('translate(36px, 20px) rotate(-45deg) scale(1.5)');
    expect(cardContainerCss).toContain('width: 100%;');
    expect(cardContainerCss).toContain('justify-content: center;');
    expect(auctionCardCss).toContain('box-sizing: border-box;');
    expect(auctionCardCss).toContain('width: min(360px, 100%);');
    expect(auctionCardCss).not.toContain('100vw');
  });
});
