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

  it('keeps the opened bid sheet to half height so the live view remains prominent', () => {
    const css = readLiveCss();
    const videoAreaCompactCss = getClassBlock(css, 'videoAreaCompact');
    const sheetCss = getClassBlock(css, 'sheet');

    expect(videoAreaCompactCss).toContain('height: 50%;');
    expect(sheetCss).toContain('height: 50dvh;');
    expect(sheetCss).toContain('box-sizing: border-box;');
  });

  it('uses warm glass styling for the ranking block in dark mode', () => {
    const css = readLiveCss();
    const darkRankingBlockCss = css.match(/:global\(:root\[data-theme='dark'\]\) \.rankingBlock,[\s\S]*?\n\}/)?.[0] ?? '';
    const darkRankingGlowCss = css.match(/:global\(:root\[data-theme='dark'\]\) \.rankingGlow,[\s\S]*?\n\}/)?.[0] ?? '';
    const darkMyBidCardCss = css.match(/:global\(:root\[data-theme='dark'\]\) \.myBidCard,[\s\S]*?\n\}/)?.[0] ?? '';

    expect(darkRankingBlockCss).toContain(":global(:root:not([data-theme])) .rankingBlock");
    expect(darkRankingBlockCss).toContain('linear-gradient(145deg, rgba(25, 18, 10, 0.98) 0%, rgba(8, 12, 20, 0.96) 100%)');
    expect(darkRankingBlockCss).toContain('rgba(245, 158, 11, 0.38)');
    expect(darkRankingGlowCss).toContain('rgba(245, 158, 11, 0.32)');
    expect(darkMyBidCardCss).toContain('rgba(245, 158, 11, 0.22)');
  });

  it('keeps live empty upcoming state colors bound to theme tokens', () => {
    const css = readLiveCss();
    const liveEmptyPageCss = getClassBlock(css, 'liveEmptyPage');
    const liveEmptyTitleCss = getClassBlock(css, 'liveEmptyTitle');
    const liveEmptyIconRingCss = getClassBlock(css, 'liveEmptyIconRing');
    const liveEmptyPrimaryLinkCss = getClassBlock(css, 'liveEmptyPrimaryLink');
    const upcomingCardCss = getClassBlock(css, 'upcomingCard');

    expect(liveEmptyPageCss).toContain('background: var(--bg-page);');
    expect(liveEmptyPageCss).toContain('color: var(--text-primary);');
    expect(liveEmptyTitleCss).toContain('color: var(--text-primary);');
    expect(getDeclaration(liveEmptyIconRingCss, 'border')).toContain('var(');
    expect(getDeclaration(liveEmptyIconRingCss, 'background')).toContain('var(');
    expect(getDeclaration(liveEmptyPrimaryLinkCss, 'background')).toContain('var(');
    expect(getDeclaration(liveEmptyPrimaryLinkCss, 'color')).toContain('var(');
    expect(getDeclaration(liveEmptyPrimaryLinkCss, 'box-shadow')).toContain('var(');
    expect(upcomingCardCss).toContain('background: var(--bg-elevated);');
  });

  it('keeps live empty state anchored near the top instead of vertically centered', () => {
    const css = readLiveCss();
    const liveEmptyPageCss = getClassBlock(css, 'liveEmptyPage');

    expect(liveEmptyPageCss).toContain('align-items: flex-start;');
  });

  it('prevents long product names from overflowing the auction card', () => {
    const css = readLiveCss();
    const productCardContentCss = css.match(/\.productCard\s*>\s*div\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const productTitleCss = getClassBlock(css, 'productCard h1');

    expect(productCardContentCss).toContain('min-width: 0;');
    expect(productTitleCss).toContain('overflow: hidden;');
    expect(productTitleCss).toContain('text-overflow: ellipsis;');
    expect(productTitleCss).toContain('white-space: nowrap;');
  });

  it('keeps the auction ended summary in the luxury shimmer treatment', () => {
    const css = readLiveCss();
    const endedSummaryCss = getClassBlock(css, 'endedSummary');
    const endedSummaryBeforeCss = css.match(/\.endedSummary::before\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const endedSummaryAfterCss = css.match(/\.endedSummary::after\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const endedProductNameCss = getClassBlock(css, 'endedSummary p');
    const endedPriceCss = getClassBlock(css, 'endedSummary strong');

    expect(endedSummaryCss).toContain('overflow: hidden;');
    expect(endedSummaryCss).toContain('text-align: center;');
    expect(endedSummaryCss).toContain('max-width: 360px;');
    expect(endedSummaryCss).toContain('margin: 0 auto;');
    expect(endedSummaryBeforeCss).toContain('animation: endedSummaryShimmer 2.5s infinite;');
    expect(endedSummaryAfterCss).toContain("content: 'SOLD';");
    expect(endedProductNameCss).toContain('overflow: hidden;');
    expect(endedProductNameCss).toContain('text-overflow: ellipsis;');
    expect(endedProductNameCss).toContain('white-space: nowrap;');
    expect(endedPriceCss).toContain('font-size: 42px;');
  });
});
