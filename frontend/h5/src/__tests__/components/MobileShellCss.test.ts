import { readFileSync } from 'node:fs';
import { join } from 'node:path';

describe('MobileShell css', () => {
  const css = readFileSync(
    join(__dirname, '..', '..', 'components', 'MobileShell', 'MobileShell.module.css'),
    'utf8',
  );

  it('keeps the desktop preview nav pinned to the phone frame while long pages scroll internally', () => {
    const desktopMediaCss = css.match(
      /@media \(min-width: 431px\) and \(hover: hover\) and \(pointer: fine\) \{[\s\S]*?\n\}/,
    )?.[0] ?? '';

    expect(desktopMediaCss).toContain('height: min(100vh, 812px);');
    expect(desktopMediaCss).toContain('overflow: hidden;');
    expect(desktopMediaCss).toContain('overflow-y: auto;');
  });

  it('uses a nav-level shared indicator instead of per-tab active capsules', () => {
    expect(css).toContain('.navIndicator');
    expect(css).toContain('.navIndicatorLine');
    expect(css).toContain('--nav-indicator-width: 72px;');
    expect(css).toContain('width: var(--nav-indicator-width);');
    expect(css).toContain('transform: translate3d(var(--nav-indicator-x), 0, 0);');
    expect(css).toContain('z-index: 0;');
    expect(css).toContain('z-index: 1;');
    expect(css).not.toContain('.navItem::before');
  });

  it('disables shared indicator transitions for reduced motion users', () => {
    const reducedMotionCss = css.match(/@media \(prefers-reduced-motion: reduce\) \{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(reducedMotionCss).toContain('.navIndicator');
    expect(reducedMotionCss).toContain('.navIndicatorLine');
    expect(reducedMotionCss).toContain('transition: none;');
  });
});
