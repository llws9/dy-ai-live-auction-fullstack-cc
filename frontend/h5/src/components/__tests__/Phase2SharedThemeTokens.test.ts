import { readFileSync } from 'fs';
import { join } from 'path';

const componentRoot = join(__dirname, '..');

function readComponentCss(relativePath: string) {
  return readFileSync(join(componentRoot, relativePath), 'utf8');
}

const oldLightOnlyTokenPatterns = [
  /var\(--bg-primary\b/,
  /var\(--bg-secondary\b/,
  /var\(--bg-tertiary\b/,
  /var\(--border-light\b/,
  /var\(--border-default\b/,
];

describe('phase 2 shared component theme tokens', () => {
  it.each([
    ['shared/Input.module.css'],
    ['shared/Loading.module.css'],
    ['shared/Skeleton.module.css'],
    ['LiveReminderModal/LiveReminderModal.module.css'],
  ])('removes old light-only tokens from %s', (relativePath) => {
    const css = readComponentCss(relativePath);

    for (const tokenPattern of oldLightOnlyTokenPatterns) {
      expect(css).not.toMatch(tokenPattern);
    }
  });

  it('uses phase 2 loading and skeleton surfaces', () => {
    const loadingCss = readComponentCss('shared/Loading.module.css');
    const skeletonCss = readComponentCss('shared/Skeleton.module.css');

    expect(loadingCss).toContain('background: var(--surface-glass);');
    expect(loadingCss).toContain('border: 3px solid var(--skeleton-bg);');
    expect(loadingCss).toContain('border-top-color: var(--text-brand);');
    expect(skeletonCss).toContain('background: var(--skeleton-bg);');
    expect(skeletonCss).toContain('var(--skeleton-wave)');
  });

  it('uses phase 2 danger tokens in BadgeDot fallback styling', () => {
    const css = readComponentCss('BadgeDot/BadgeDot.module.css');

    expect(css).toContain('var(--danger-text)');
    expect(css).toContain('var(--bg-surface)');
  });

  it('uses theme-aware hover states in shared Button', () => {
    const css = readComponentCss('shared/Button.module.css');

    expect(css).toContain('background: var(--surface-muted);');
    expect(css).toContain('outline: 2px solid var(--focus-ring);');
  });
});
