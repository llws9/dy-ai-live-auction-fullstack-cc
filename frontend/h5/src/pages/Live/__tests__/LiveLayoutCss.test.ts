import { readFileSync } from 'node:fs';
import { join } from 'node:path';

describe('Live layout css', () => {
  const readLiveCss = () => readFileSync(join(__dirname, '..', 'Live.module.css'), 'utf8');
  const getClassBlock = (css: string, className: string) => css.match(new RegExp(`\\.${className}\\s*\\{[\\s\\S]*?\\n\\}`))?.[0] ?? '';
  const getDeclaration = (block: string, property: string) =>
    block.match(new RegExp(`${property}:\\s*([^;]+);`))?.[1] ?? '';

  it('places fixed-price cards at the lower-right above the bid dock and away from chat input', () => {
    const css = readLiveCss();
    const fixedPriceListCss = getClassBlock(css, 'fixedPriceList');

    expect(fixedPriceListCss).toContain('right: var(--spacing-4);');
    expect(fixedPriceListCss).toContain('bottom: calc(96px + env(safe-area-inset-bottom, 0px));');
    expect(fixedPriceListCss).toContain('width: min(40vw, 156px);');
    expect(fixedPriceListCss).toContain('overflow: hidden;');
    expect(fixedPriceListCss).toContain('overflow-x: hidden;');
    expect(fixedPriceListCss).not.toContain('left: var(--spacing-4);');
  });

  it('shortens the live chat overlay so it does not collide with the fixed-price card', () => {
    const css = readLiveCss();
    const liveChatOverlayCss = getClassBlock(css, 'liveChatOverlay');

    expect(liveChatOverlayCss).toContain('width: min(50vw, 210px);');
  });

  it('keeps live empty upcoming state colors bound to theme tokens', () => {
    const css = readLiveCss();
    const liveEmptyPageCss = getClassBlock(css, 'liveEmptyPage');
    const liveEmptyTitleCss = getClassBlock(css, 'liveEmptyTitle');
    const liveEmptyPrimaryLinkCss = getClassBlock(css, 'liveEmptyPrimaryLink');
    const upcomingCardCss = getClassBlock(css, 'upcomingCard');

    expect(liveEmptyPageCss).toContain('background: var(--bg-page);');
    expect(liveEmptyPageCss).toContain('color: var(--text-primary);');
    expect(liveEmptyTitleCss).toContain('color: var(--text-primary);');
    expect(getDeclaration(liveEmptyPrimaryLinkCss, 'background')).toContain('var(');
    expect(getDeclaration(liveEmptyPrimaryLinkCss, 'color')).toContain('var(');
    expect(upcomingCardCss).toContain('background: var(--bg-elevated);');
  });
});
