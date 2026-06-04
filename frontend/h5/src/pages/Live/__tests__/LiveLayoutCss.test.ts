import { readFileSync } from 'node:fs';
import { join } from 'node:path';

describe('Live layout css', () => {
  it('places fixed-price cards at the lower-right above the bid dock and away from chat input', () => {
    const css = readFileSync(join(__dirname, '..', 'Live.module.css'), 'utf8');
    const fixedPriceListCss = css.match(/\.fixedPriceList\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(fixedPriceListCss).toContain('right: var(--spacing-4);');
    expect(fixedPriceListCss).toContain('bottom: calc(96px + env(safe-area-inset-bottom, 0px));');
    expect(fixedPriceListCss).toContain('width: min(40vw, 156px);');
    expect(fixedPriceListCss).toContain('overflow: hidden;');
    expect(fixedPriceListCss).toContain('overflow-x: hidden;');
    expect(fixedPriceListCss).not.toContain('left: var(--spacing-4);');
  });

  it('shortens the live chat overlay so it does not collide with the fixed-price card', () => {
    const css = readFileSync(join(__dirname, '..', 'Live.module.css'), 'utf8');
    const liveChatOverlayCss = css.match(/\.liveChatOverlay\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(liveChatOverlayCss).toContain('width: min(50vw, 210px);');
  });
});
