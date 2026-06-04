import { readFileSync } from 'node:fs';
import { join } from 'node:path';

describe('ProductDetail css', () => {
  const css = readFileSync(join(__dirname, '..', 'ProductDetail.module.css'), 'utf8');

  it('keeps the upcoming status badge readable in light mode', () => {
    const statusBadgeCss = css.match(/\.statusBadge\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(statusBadgeCss).toContain('background: rgba(36, 31, 24, 0.84);');
    expect(statusBadgeCss).toContain('color: #fff4dc;');
    expect(statusBadgeCss).toContain('box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);');
  });

  it('renders upcoming reminder CTA as a wide in-content button', () => {
    const reminderCtaCss = css.match(/\.reminderCta\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const contentCss = css.match(/\.content\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(contentCss).toContain('padding-bottom: var(--spacing-5);');
    expect(reminderCtaCss).toContain('width: calc(100% - var(--spacing-5) * 2);');
    expect(reminderCtaCss).toContain('height: 52px;');
    expect(reminderCtaCss).toContain('margin: 0 var(--spacing-5) var(--spacing-6);');
  });
});
