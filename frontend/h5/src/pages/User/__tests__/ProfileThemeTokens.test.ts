import { readFileSync } from 'fs';
import { join } from 'path';

const css = readFileSync(join(__dirname, '..', 'Profile.module.css'), 'utf8');

describe('Profile phase 2 theme tokens', () => {
  it('uses profile atmosphere and semantic surface tokens', () => {
    expect(css).toContain('background: var(--page-gradient-profile);');
    expect(css).toContain('border: 2px solid var(--avatar-border);');
    expect(css).toContain('background: var(--avatar-bg);');
    expect(css).toContain('box-shadow: var(--avatar-shadow);');
    expect(css).toContain('background: var(--chip-bg);');
    expect(css).toContain('border: 1px solid var(--card-border-accent);');
    expect(css).toContain('background: var(--item-subtle-bg);');
    expect(css).toContain('background: var(--icon-tile-bg);');
    expect(css).toContain('color: var(--danger-text);');
  });

  it('does not keep dark-only values in theme-sensitive Profile blocks', () => {
    expect(css).not.toContain('linear-gradient(180deg, #242424 0%, #171717 42%, #101010 100%)');
    expect(css).not.toContain('background: rgba(44, 44, 44, 0.82);');
    expect(css).not.toContain('background: rgba(26, 26, 26, 0.64);');
    expect(css).not.toContain('color: #f87171;');
  });
});
