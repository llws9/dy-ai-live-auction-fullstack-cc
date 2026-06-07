import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveRoomCard, { LiveRoomItem } from '../LiveRoomCard';

const renderCard = (room: LiveRoomItem, onSubscribe = jest.fn(), onEnter = jest.fn()) =>
  render(
    <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <LiveRoomCard room={room} onSubscribe={onSubscribe} onEnter={onEnter} subscribedProductIds={new Set()} />
    </MemoryRouter>,
  );

test('有 current_auction 时显示直播中并以进入直播为主操作', () => {
  const onEnter = jest.fn();
  renderCard({ id: 1, name: '瑾瑜珠宝', status: 1, current_auction_id: 11, current_product_id: 8, current_price: '1200.00', recent_deals: [] }, jest.fn(), onEnter);
  expect(screen.getByText('直播中')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '进入直播间' }));
  expect(onEnter).toHaveBeenCalledWith(1, 11);
});

test('无 current 有 next 时显示即将开始并以预约为主操作', () => {
  const onSubscribe = jest.fn();
  renderCard(
    { id: 2, name: '云裳阁', status: 0, current_auction_id: null, next_auction: { auction_id: 21, product_id: 8, product_name: '翡翠手镯', start_price: '300.00', start_time: '2026-06-08T10:00:00Z' }, recent_deals: [] },
    onSubscribe,
  );
  expect(screen.getByText('即将开始')).toBeInTheDocument();
  expect(screen.getByText('翡翠手镯')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '预约开拍提醒' }));
  expect(onSubscribe).toHaveBeenCalledWith(8, 21);
});

test('渲染最近成交氛围信息', () => {
  renderCard({ id: 3, name: 'X', status: 1, current_auction_id: 5, recent_deals: [{ product_name: '和田玉牌', final_price: '500.00' }] });
  expect(screen.getByText(/和田玉牌/)).toBeInTheDocument();
});
