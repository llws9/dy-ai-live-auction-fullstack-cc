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
});
