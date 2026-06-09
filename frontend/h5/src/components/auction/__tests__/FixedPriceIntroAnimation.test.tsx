import { render, screen, waitFor } from '@testing-library/react';
import { FixedPriceIntroAnimation } from '../FixedPriceIntroAnimation';

describe('FixedPriceIntroAnimation', () => {
  const item = {
    id: 7001,
    auction_id: 5,
    price: '99.00',
    product_brief: {
      id: 8001,
      title: '一口价翡翠',
      cover_image: '/fp.jpg',
    },
  } as any;

  it('computes the fly-away target from its live-room container instead of the browser viewport', async () => {
    const rectSpy = jest.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function () {
      if ((this as HTMLElement).dataset.testid === 'fixed-price-intro-container') {
        return {
          width: 390,
          height: 844,
          top: 100,
          left: 300,
          right: 690,
          bottom: 944,
          x: 300,
          y: 100,
          toJSON: () => ({}),
        } as DOMRect;
      }

      return {
        width: 0,
        height: 0,
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      } as DOMRect;
    });

    try {
      render(<FixedPriceIntroAnimation item={item} onComplete={jest.fn()} />);

      const card = screen.getByTestId('fixed-price-intro-card');

      await waitFor(() => {
        expect(card).toHaveStyle('--fly-to-x: 115px');
        expect(card).toHaveStyle('--fly-to-y: 386.4px');
      });
    } finally {
      rectSpy.mockRestore();
    }
  });
});
