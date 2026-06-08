import { readFileSync } from 'fs';
import { join } from 'path';

describe('FixedPriceIntroAnimation CSS', () => {
  const introCss = readFileSync(
    join(__dirname, '../FixedPriceIntroAnimation.module.css'),
    'utf8'
  );
  const cardCss = readFileSync(
    join(__dirname, '../../FixedPriceCard/index.module.css'),
    'utf8'
  );

  it('uses transform/opacity for motion and supports reduced motion', () => {
    const keyframes = introCss.match(/@keyframes[\s\S]+?(?=\.badge)/)?.[0] ?? '';

    expect(keyframes).not.toMatch(/\b(top|left)\s*:/);
    expect(keyframes).toMatch(/transform:/);
    expect(keyframes).toMatch(/opacity:/);
    expect(introCss).toContain('@media (prefers-reduced-motion: reduce)');
    expect(cardCss).toContain('@media (prefers-reduced-motion: reduce)');
  });
});
