import React from 'react';
import { act, render, screen, fireEvent } from '@testing-library/react';
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

  it('渲染 myBidStatus 状态胶囊', () => {
    const { rerender } = render(<BidDock {...baseProps} myBidStatus="leading" />);
    expect(screen.getByText('当前领先')).toBeInTheDocument();

    rerender(<BidDock {...baseProps} myBidStatus="outbid" />);
    expect(screen.getByText('被超越')).toBeInTheDocument();
  });

  it('默认态不显示放大价格，点击出价打开出价态', () => {
    const onOpen = jest.fn();
    render(<BidDock {...baseProps} onOpen={onOpen} />);

    fireEvent.click(screen.getByRole('button', { name: '出价' }));

    expect(onOpen).toHaveBeenCalledWith('bid');
  });

  it('商品图加载失败时切换到稳定兜底图，避免显示浏览器破图', () => {
    render(<BidDock {...baseProps} productImage="https://example.com/broken.jpg" />);

    const image = screen.getByRole('img', { name: '明代紫砂壶' }) as HTMLImageElement;
    fireEvent.error(image);

    expect(image.src).toContain('/api/ide/v1/text_to_image');
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

  it('打开抽屉时把 topAddon 渲染为抽屉顶部坞，而不是塞进滚动内容', () => {
    render(
      <BidDock {...baseProps} sheet="bid" topAddon={<section aria-label="竞拍战况热度">热度条</section>}>
        <div>出价表单</div>
      </BidDock>
    );

    const addon = screen.getByTestId('bid-dock-top-addon');
    const heatBar = screen.getByLabelText('竞拍战况热度');
    const sheet = screen.getByLabelText('收起竞拍面板').parentElement;

    expect(addon).toHaveClass('sheetDockAddon');
    expect(addon).toContainElement(heatBar);
    expect(sheet).not.toContainElement(heatBar);
  });

  it('打开抽屉时先挂载闭合态，再进入滑入态', () => {
    const rafCallbacks: FrameRequestCallback[] = [];
    const requestAnimationFrameSpy = jest
      .spyOn(window, 'requestAnimationFrame')
      .mockImplementation((callback) => {
        rafCallbacks.push(callback);
        return rafCallbacks.length;
      });

    render(
      <BidDock {...baseProps} sheet="bid">
        <div>出价表单</div>
      </BidDock>
    );

    const sheet = screen.getByLabelText('收起竞拍面板').parentElement;
    expect(sheet).toHaveClass('sheet');
    expect(sheet).not.toHaveClass('sheetOpen');

    act(() => {
      rafCallbacks.forEach((callback) => callback(16));
    });

    expect(sheet).toHaveClass('sheetOpen');
    requestAnimationFrameSpy.mockRestore();
  });

  it('关闭抽屉时先播放下滑动画，再卸载 DOM', () => {
    jest.useFakeTimers();
    const requestAnimationFrameSpy = jest
      .spyOn(window, 'requestAnimationFrame')
      .mockImplementation((callback) => {
        callback(16);
        return 1;
      });

    const { rerender } = render(
      <BidDock {...baseProps} sheet="bid">
        <div>出价表单</div>
      </BidDock>
    );

    const sheet = screen.getByLabelText('收起竞拍面板').parentElement;
    expect(sheet).toHaveClass('sheetOpen');

    rerender(
      <BidDock {...baseProps} sheet={null}>
        <div>出价表单</div>
      </BidDock>
    );

    expect(screen.getByLabelText('收起竞拍面板').parentElement).not.toHaveClass('sheetOpen');

    act(() => {
      jest.advanceTimersByTime(350);
    });

    expect(screen.queryByLabelText('收起竞拍面板')).not.toBeInTheDocument();
    requestAnimationFrameSpy.mockRestore();
    jest.useRealTimers();
  });
});
