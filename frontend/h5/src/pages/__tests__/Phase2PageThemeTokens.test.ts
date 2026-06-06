import { readFileSync } from 'fs';
import { join } from 'path';

function readPageCss(relativePath: string) {
  return readFileSync(join(__dirname, '..', relativePath), 'utf8');
}

describe('phase 2 scoped page theme tokens', () => {
  it('tokenizes Addresses page surfaces and actions', () => {
    const css = readPageCss('Addresses/Addresses.module.css');

    expect(css).toContain('background: var(--page-gradient-profile);');
    expect(css).toContain('border: 1px solid var(--card-border-accent);');
    expect(css).toContain('background: var(--surface-glass);');
    expect(css).toContain('color: var(--danger-text);');
    expect(css).not.toContain('var(--bg-page-start, #1a1a1a)');
    expect(css).not.toContain('background: rgba(255, 255, 255, 0.06);');
    expect(css).not.toContain('color: #ffb3b3;');
  });

  it('tokenizes Order detail page surfaces and inline feedback', () => {
    const css = readPageCss('Order/Detail.module.css');

    expect(css).toContain('border: 1px solid var(--card-border-accent);');
    expect(css).toContain('background: var(--bg-surface);');
    expect(css).toContain('background: var(--surface-glass);');
    expect(css).toContain('background: var(--item-subtle-bg);');
    expect(css).not.toContain('background: rgba(0, 0, 0, 0.78);');
    expect(css).not.toContain('color: #fff;');
  });

  it.each([
    ['Notifications/Notifications.module.css'],
    ['Follow/Following.module.css'],
    ['History/AuctionHistory.module.css'],
    ['Order/List.module.css'],
  ])('keeps %s on semantic theme surfaces', (relativePath) => {
    const css = readPageCss(relativePath);

    expect(css).toContain('var(--bg-page)');
    expect(css).toContain('var(--bg-surface)');
    expect(css).toContain('var(--border-subtle)');
    expect(css).not.toContain('var(--bg-primary)');
    expect(css).not.toContain('var(--bg-secondary)');
    expect(css).not.toContain('var(--border-light)');
  });

  it('keeps Following inactive and active avatar borders visually distinct', () => {
    const css = readPageCss('Follow/Following.module.css');

    expect(css).toContain('border: 2px solid var(--border-subtle);');
    expect(css).toContain('border-color: var(--avatar-border);');
    expect(css).not.toContain('border: 2px solid var(--avatar-border);');
  });
});
