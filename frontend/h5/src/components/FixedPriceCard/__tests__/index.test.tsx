import React from 'react';
import { render } from '@testing-library/react';
import '@testing-library/jest-dom';
import FixedPriceCard from '../index';

describe('FixedPriceCard', () => {
  const mockItem = {
    id: 1,
    product_id: 101,
    product_title: 'Test Product',
    price: '99.00',
    total_stock: 10,
    remaining_stock: 5,
    status: 'online' as const,
    product: {
      id: 101,
      title: 'Test Product',
      cover_image: 'test.jpg'
    }
  };

  it('should apply pulsing class when isPulsing is true', () => {
    const { container } = render(
      <FixedPriceCard item={mockItem} isPulsing={true} onPurchase={jest.fn()} />
    );
    const article = container.querySelector('article');
    expect(article).toHaveClass('pulsing');
  });

  it('should not apply pulsing class when isPulsing is false', () => {
    const { container } = render(
      <FixedPriceCard item={mockItem} isPulsing={false} onPurchase={jest.fn()} />
    );
    const article = container.querySelector('article');
    expect(article).not.toHaveClass('pulsing');
  });
});
