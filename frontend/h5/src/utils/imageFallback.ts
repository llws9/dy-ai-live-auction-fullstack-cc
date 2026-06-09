import type { SyntheticEvent } from 'react';

export const DEFAULT_AUCTION_IMAGE = '/assets/default-auction-cover.svg';

export function replaceBrokenImageWithFallback(event: SyntheticEvent<HTMLImageElement>) {
  const image = event.currentTarget;
  if (image.dataset.fallbackApplied === 'true') {
    return;
  }

  image.dataset.fallbackApplied = 'true';
  image.src = DEFAULT_AUCTION_IMAGE;
}
