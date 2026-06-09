import type { SyntheticEvent } from 'react';

export const DEFAULT_AUCTION_IMAGE =
  'https://copilot-cn.bytedance.net/api/ide/v1/text_to_image?prompt=professional%20auction%20catalog%20photo%20of%20premium%20antique%20collectible%2C%20warm%20studio%20lighting%2C%20realistic%20product%20photography%2C%20mobile%20ecommerce%20card%20cover&image_size=landscape_4_3';

export function replaceBrokenImageWithFallback(event: SyntheticEvent<HTMLImageElement>) {
  const image = event.currentTarget;
  if (image.dataset.fallbackApplied === 'true') {
    return;
  }

  image.dataset.fallbackApplied = 'true';
  image.src = DEFAULT_AUCTION_IMAGE;
}
