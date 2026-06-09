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
    const containerBlock = introCss.match(/\.container\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(containerBlock).toContain('position: absolute;');
    expect(containerBlock).not.toContain('position: fixed;');
    expect(keyframes).not.toMatch(/\b(top|left)\s*:/);
    expect(keyframes).not.toMatch(/\b(vw|vh)\b/);
    expect(keyframes).toMatch(/transform:/);
    expect(keyframes).toMatch(/opacity:/);
    expect(introCss).toContain('@media (prefers-reduced-motion: reduce)');
    expect(cardCss).toContain('@media (prefers-reduced-motion: reduce)');
  });

  it('holds the listed item around the product showcase area before flying to the fixed-price card', () => {
    const cardBlock = introCss.match(/\.card\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(cardBlock).toContain('top: 40%;');
    expect(cardBlock).toMatch(/slideDown[\s\S]*showcaseHold[\s\S]*flyToBottomRight/);
    expect(introCss).toContain('@keyframes showcaseHold');
    expect(introCss).toMatch(/showcaseHold[\s\S]*translate3d\(-50%, -50%, 0\)/);
  });
});
