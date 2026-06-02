import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import BidDock from '../BidDock';

const baseProps = {
  product: { name: '明代紫砂壶' },
  productImage: '/product.jpg',
  roomName: '瓷器珍藏夜场',
  currentPrice: 1200,
  sheet: null as 'bid' | 'info' | null,
  isAuthenticated: true,
  onOpen: jest.fn(),
  onClose: jest.fn(),
  onRequireLogin: jest.fn(),
};

describe('BidDock', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('默认态不显示放大价格，点击出价打开出价态', () => {
    const onOpen = jest.fn();
    render(<BidDock {...baseProps} onOpen={onOpen} />);

    fireEvent.click(screen.getByRole('button', { name: '出价' }));

    expect(onOpen).toHaveBeenCalledWith('bid');
  });

  it('未登录点击出价触发登录引导而非打开抽屉', () => {
    const onOpen = jest.fn();
    const onRequireLogin = jest.fn();
    render(
      <BidDock
        {...baseProps}
        isAuthenticated={false}
        onOpen={onOpen}
        onRequireLogin={onRequireLogin}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '出价' }));

    expect(onRequireLogin).toHaveBeenCalledTimes(1);
    expect(onOpen).not.toHaveBeenCalled();
  });

  it('点击遮罩收起抽屉', () => {
    const onClose = jest.fn();
    render(
      <BidDock {...baseProps} sheet="bid" onClose={onClose}>
        <div>出价表单</div>
      </BidDock>
    );

    fireEvent.click(screen.getByTestId('bid-dock-mask'));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
