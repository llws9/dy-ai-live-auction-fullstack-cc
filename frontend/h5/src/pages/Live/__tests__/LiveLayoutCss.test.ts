import { readFileSync } from 'node:fs';
import { join } from 'node:path';

describe('Live layout css', () => {
  const readLiveCss = () => readFileSync(join(__dirname, '..', 'Live.module.css'), 'utf8');
  const readTreasureCss = () => readFileSync(join(__dirname, '..', 'TreasureProgressBar.module.css'), 'utf8');
  const getClassBlock = (css: string, className: string) => css.match(new RegExp(`\\.${className}\\s*\\{[\\s\\S]*?\\n\\}`))?.[0] ?? '';
  const getDeclaration = (block: string, property: string) =>
    block.match(new RegExp(`${property}:\\s*([^;]+);`))?.[1] ?? '';

  it('anchors the treasure progress panel below the host pill', () => {
    const css = readTreasureCss();
    const containerCss = getClassBlock(css, 'container');
    const glassPanelCss = getClassBlock(css, 'glassPanel');

    expect(containerCss).toContain('position: absolute;');
    expect(containerCss).toContain('top: calc(var(--spacing-4) + env(safe-area-inset-top, 0px) + 58px);');
    expect(containerCss).toContain('left: var(--spacing-4);');
    expect(containerCss).toContain('width: min(calc(100vw - 132px), 260px);');
    expect(containerCss).toContain('z-index: 3;');
    expect(glassPanelCss).toContain('box-sizing: border-box;');
  });

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

  it('uses the framed live container height on desktop previews so bottom overlays stay visible', () => {
    const css = readLiveCss();
    const desktopLivePageCss = css.match(/@media \(min-width: 431px\) and \(hover: hover\) and \(pointer: fine\) \{[\s\S]*?\.page\s*\{[\s\S]*?\n  \}[\s\S]*?\n\}/)?.[0] ?? '';

    expect(desktopLivePageCss).toContain('.page');
    expect(desktopLivePageCss).toContain('height: 100%;');
    expect(desktopLivePageCss).toContain('min-height: 0;');
  });

  it('keeps the live feed wrapper height definite for percentage-based live room layout', () => {
    const css = readLiveCss();
    const feedShellCss = getClassBlock(css, 'feedShell');

    expect(feedShellCss).toContain('height: 100%;');
    expect(feedShellCss).toContain('min-height: 0;');
    expect(feedShellCss).toContain('overflow: hidden;');
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

  it('keeps the bid sheet focused on ranking after removing the product card', () => {
    const css = readLiveCss();
    const rankingBlockCss = getClassBlock(css, 'rankingBlock');

    expect(css).not.toContain('.productCard');
    expect(css).not.toContain('.productFallback');
    expect(css).not.toContain('.followRow');
    expect(css).not.toContain('.followButton');
    expect(rankingBlockCss).toContain('position: relative;');
    expect(rankingBlockCss).toContain('padding: 16px;');
  });

  it('keeps the online viewers pill compact without a close affordance', () => {
    const css = readLiveCss();
    const viewersRowCss = getClassBlock(css, 'viewersRow');

    expect(css).not.toContain('--live-header-pill-width');
    expect(viewersRowCss).not.toContain('width:');
    expect(viewersRowCss).toContain('height: 34px;');
    expect(viewersRowCss).toContain('box-sizing: border-box;');
    expect(css).not.toContain('.closeBtn');
  });

  it('groups viewer and like counts into a weak data island separate from primary actions', () => {
    const css = readLiveCss();
    const rightActionsCss = getClassBlock(css, 'rightActions');
    const dataIslandCss = getClassBlock(css, 'topDataIsland');
    const actionRowCss = getClassBlock(css, 'topActionRow');
    const viewersRowCss = getClassBlock(css, 'viewersRow');
    const likesPillCss = getClassBlock(css, 'likesPill');
    const likesPillIconCss = getClassBlock(css, 'likesPill span');

    expect(rightActionsCss).toContain('flex-direction: column;');
    expect(rightActionsCss).toContain('align-items: flex-end;');
    expect(dataIslandCss).toContain('display: flex;');
    expect(dataIslandCss).toContain('box-sizing: border-box;');
    expect(dataIslandCss).toContain('height: 44px;');
    expect(dataIslandCss).toContain('background: rgba(0, 0, 0, 0.2);');
    expect(dataIslandCss).toContain('backdrop-filter: blur(12px);');
    expect(viewersRowCss).toContain('height: 34px;');
    expect(likesPillCss).toContain('height: 34px;');
    expect(actionRowCss).toContain('display: flex;');
    expect(likesPillIconCss).toContain('color: #ef4444;');
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
