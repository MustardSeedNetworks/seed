/**
 * palette.ts — read the theme's colour tokens the way the cascade applies them,
 * and measure WCAG contrast between them. Shared by the theme tests.
 */
import { readFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export const src = resolve(dirname(fileURLToPath(import.meta.url)), '..');

export type Mode = 'light' | 'dark';

function blocks(css: string, selector: string): string[] {
  const out: string[] = [];
  const re = new RegExp(`(^|\\n)${selector.replace('.', '\\.')}\\s*\\{`, 'g');
  for (const m of css.matchAll(re)) {
    let depth = 1;
    let i = m.index + m[0].length;
    const start = i;
    while (depth > 0 && i < css.length) {
      if (css[i] === '{') depth++;
      if (css[i] === '}') depth--;
      i++;
    }
    out.push(css.slice(start, i - 1));
  }
  return out;
}

export function palette(mode: Mode): Map<string, string> {
  const css = ['theme/msn-shared.css', 'theme/product-seed.css', 'index.css'].map((file) =>
    readFileSync(join(src, file), 'utf8'),
  );
  // Every :root first, then every .dark, as the cascade applies them.
  const selectors = mode === 'light' ? [':root'] : [':root', '.dark'];
  const tokens = new Map<string, string>();
  for (const body of selectors.flatMap((selector) =>
    css.flatMap((file) => blocks(file, selector)),
  )) {
    for (const [, name = '', hex = ''] of body.matchAll(
      /--color-([\w-]+):\s*(#[0-9a-f]{6})\s*;/gi,
    )) {
      tokens.set(name, hex.toLowerCase());
    }
  }
  return tokens;
}

export type Rgb = [number, number, number];

export function rgb(hex: string): Rgb {
  return [1, 3, 5].map((i) => Number.parseInt(hex.slice(i, i + 2), 16)) as Rgb;
}

function luminance([r, g, b]: Rgb): number {
  const linear = (v: number) => {
    const s = v / 255;
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * linear(r) + 0.7152 * linear(g) + 0.0722 * linear(b);
}

export function contrast(a: Rgb, b: Rgb): number {
  const la = luminance(a);
  const lb = luminance(b);
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
}
