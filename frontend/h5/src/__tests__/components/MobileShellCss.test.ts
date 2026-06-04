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
});
