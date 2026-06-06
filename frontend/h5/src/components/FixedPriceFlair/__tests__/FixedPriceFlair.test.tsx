import { act, render, screen } from '@testing-library/react';
import FixedPriceFlair from '../index';

type FlairMessage = {
  payload?: {
    item_id?: number;
    buyer_id?: number;
    buyer_nickname: string;
    product_title: string;
    price: string;
  };
  data?: {
    item_id?: number;
    buyer_id?: number;
    buyer_nickname: string;
    product_title: string;
    price: string;
  };
};

type FlairHandler = (message: FlairMessage) => void;

function createSubscribeSocket() {
  const unsubscribe = jest.fn();
  const socket = {
    subscribe: jest.fn((_type: string, _handler: FlairHandler) => unsubscribe),
  };

  return { socket, unsubscribe };
}

function pushFlair(handler: FlairHandler, buyer: string, itemId: number) {
  handler({
    payload: {
      item_id: itemId,
      buyer_nickname: buyer,
      product_title: '翡翠手镯',
      price: '99.00',
    },
  });
}

const toUtf8Mojibake = (text: string) =>
  encodeURIComponent(text).replace(/%([0-9A-F]{2})/g, (_, hex: string) => String.fromCharCode(parseInt(hex, 16)));

describe('FixedPriceFlair', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    act(() => {
      jest.runOnlyPendingTimers();
    });
    jest.useRealTimers();
  });

  it('收到 fixed_price_flair 后渲染买家/商品/价格，并在 4s 后消失', () => {
    const { socket } = createSubscribeSocket();

    render(<FixedPriceFlair socket={socket} />);
    expect(socket.subscribe).toHaveBeenCalledWith('fixed_price_flair', expect.any(Function));

    const handler = socket.subscribe.mock.calls[0][1] as FlairHandler;
    act(() => pushFlair(handler, 'Alice', 7001));

    expect(screen.getByText(/Alice/)).toBeInTheDocument();
    expect(screen.getByText(/翡翠手镯/)).toBeInTheDocument();
    expect(screen.getByText(/¥99.00/)).toBeInTheDocument();

    act(() => {
      jest.advanceTimersByTime(4100);
    });

    expect(screen.queryByText(/Alice/)).not.toBeInTheDocument();
  });

  it('同时最多堆叠 3 条飘屏，保留最新消息', () => {
    const { socket } = createSubscribeSocket();

    const { container } = render(<FixedPriceFlair socket={socket} />);
    const handler = socket.subscribe.mock.calls[0][1] as FlairHandler;

    act(() => {
      for (let index = 0; index < 5; index += 1) {
        pushFlair(handler, `U${index}`, 7001 + index);
      }
    });

    expect(container.querySelectorAll('[data-flair]')).toHaveLength(3);
    expect(screen.queryByText(/U0/)).not.toBeInTheDocument();
    expect(screen.queryByText(/U1/)).not.toBeInTheDocument();
    expect(screen.getByText(/U2/)).toBeInTheDocument();
    expect(screen.getByText(/U4/)).toBeInTheDocument();
  });

  it('卸载时取消订阅，避免继续接收 WS 消息', () => {
    const { socket, unsubscribe } = createSubscribeSocket();

    const { unmount } = render(<FixedPriceFlair socket={socket} />);
    unmount();

    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('兼容 data 包裹的后端 WS 消息', () => {
    const { socket } = createSubscribeSocket();

    render(<FixedPriceFlair socket={socket} />);
    const handler = socket.subscribe.mock.calls[0][1] as FlairHandler;

    act(() => {
      handler({
        data: {
          item_id: 7002,
          buyer_nickname: 'DataUser',
          product_title: '南红手串',
          price: '188.00',
        },
      });
    });

    expect(screen.getByText(/DataUser/)).toBeInTheDocument();
    expect(screen.getByText(/南红手串/)).toBeInTheDocument();
  });

  it('兼容后端真实 fixed_price_flair payload 缺少昵称和商品名的场景', () => {
    const { socket } = createSubscribeSocket();

    render(<FixedPriceFlair socket={socket} />);
    const handler = socket.subscribe.mock.calls[0][1] as FlairHandler;

    act(() => {
      handler({
        data: {
          item_id: 7003,
          buyer_id: 1001,
          price: '88.00',
        } as any,
      });
    });

    expect(screen.getByText(/用户 #1001/)).toBeInTheDocument();
    expect(screen.getByText(/商品 #7003/)).toBeInTheDocument();
    expect(screen.getByText(/¥88.00/)).toBeInTheDocument();
  });

  it('修复 fixed_price_flair 中的买家昵称和商品名乱码', () => {
    const { socket } = createSubscribeSocket();

    render(<FixedPriceFlair socket={socket} />);
    const handler = socket.subscribe.mock.calls[0][1] as FlairHandler;

    act(() => {
      handler({
        data: {
          item_id: 7004,
          buyer_nickname: toUtf8Mojibake('测试买家'),
          product_title: toUtf8Mojibake('南红手串'),
          price: '188.00',
        },
      });
    });

    expect(screen.getByText(/测试买家/)).toBeInTheDocument();
    expect(screen.getByText(/南红手串/)).toBeInTheDocument();
    expect(screen.queryByText(/æ|å|ç|è/)).not.toBeInTheDocument();
  });

  it('兼容 WebSocketService 的 on/off 订阅形态', () => {
    const socket = {
      on: jest.fn(),
      off: jest.fn(),
    };

    const { unmount } = render(<FixedPriceFlair socket={socket} />);
    expect(socket.on).toHaveBeenCalledWith('fixed_price_flair', expect.any(Function));

    const handler = socket.on.mock.calls[0][1] as FlairHandler;
    act(() => pushFlair(handler, 'Bob', 7002));
    expect(screen.getByText(/Bob/)).toBeInTheDocument();

    unmount();
    expect(socket.off).toHaveBeenCalledWith('fixed_price_flair', handler);
  });
});
