/**
 * pillText.test.ts — a pill's text must be legible on its own wash.
 *
 * A pill is a hue at low alpha (`bg-status-error/10`) under text of the same
 * hue. The wash pulls the ground toward the text, so every hue that clears
 * 4.5:1 on a plain surface drops under it on its own wash (.github#74:
 * 3.45–3.71 at /20 on the worst surface). No alpha fixes that, so each pill
 * hue has a `-strong` text token measured against its wash (owner 2026-09-22,
 * UI-FLEET-3), and a pill uses it instead of the bare hue.
 *
 * The grounds here are the five surface tokens. The page body also carries a
 * faint brand gradient (index.css), which only Storybook axe sees.
 */
import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const src = resolve(dirname(fileURLToPath(import.meta.url)), '..');

// The strongest wash a pill may use. Every site in the tree is at /20 or below.
const MAX_WASH = 0.2;
const SURFACES = ['base', 'raised', 'hover', 'sunken', 'deep'] as const;
const STATUS = ['status-success', 'status-warning', 'status-error', 'status-info'];
const CATS = ['cat-1', 'cat-2', 'cat-3', 'cat-4', 'cat-5', 'cat-6', 'cat-7', 'cat-8'];
const PILL_HUES = [...STATUS, 'brand-primary', 'severity-high', ...CATS];

type Mode = 'light' | 'dark';

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

function palette(mode: Mode): Map<string, string> {
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

type Rgb = [number, number, number];

function rgb(hex: string): Rgb {
  return [1, 3, 5].map((i) => Number.parseInt(hex.slice(i, i + 2), 16)) as Rgb;
}

function luminance([r, g, b]: Rgb): number {
  const linear = (v: number) => {
    const s = v / 255;
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * linear(r) + 0.7152 * linear(g) + 0.0722 * linear(b);
}

function contrast(a: Rgb, b: Rgb): number {
  const la = luminance(a);
  const lb = luminance(b);
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
}

function wash([hr, hg, hb]: Rgb, [gr, gg, gb]: Rgb, alpha: number): Rgb {
  const mix = (h: number, g: number) => Math.round(h * alpha + g * (1 - alpha));
  return [mix(hr, gr), mix(hg, gg), mix(hb, gb)];
}

describe('pill text tokens', () => {
  for (const mode of ['light', 'dark'] as const) {
    const tokens = palette(mode);
    it.each(PILL_HUES)(`${mode}: text-%s-strong is 4.5:1 on its wash over every surface`, (hue) => {
      const text = tokens.get(`${hue}-strong`);
      const base = tokens.get(hue);
      expect(text, `--color-${hue}-strong is not defined in ${mode} mode`).toBeDefined();
      expect(base).toBeDefined();
      const failing = SURFACES.flatMap((surface) => {
        const ground = tokens.get(`surface-${surface}`);
        if (!(text && base && ground)) return [`surface-${surface} missing`];
        for (let alpha = 0.05; alpha <= MAX_WASH + 1e-9; alpha += 0.05) {
          const ratio = contrast(rgb(text), wash(rgb(base), rgb(ground), alpha));
          if (ratio < 4.5) return [`${surface} /${Math.round(alpha * 100)}: ${ratio.toFixed(2)}`];
        }
        return [];
      });
      expect(failing).toEqual([]);
    });
  }
});

// Brand pills share one family: brand-accent text on a brand wash has the same
// problem as brand-primary text on it.
const FAMILY: Record<string, string> = { 'brand-accent': 'brand-primary' };
const HUE = `(status-(?:success|warning|error|info)|brand-(?:primary|accent)|severity-high|cat-[1-8])`;

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) return sourceFiles(path);
    return /\.tsx?$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name) ? [path] : [];
  });
}

describe('pill markup', () => {
  it('same-hue text on a wash uses the -strong token', () => {
    const washRe = new RegExp(String.raw`(?<![\w-])(?:[\w-]+:)*bg-${HUE}/(\d+)(?![\w-])`, 'g');
    const bare = new RegExp(String.raw`(?<![\w-])text-${HUE}(?![\w-])`, 'g');
    const offenders: string[] = [];
    for (const file of sourceFiles(src)) {
      readFileSync(file, 'utf8')
        .split('\n')
        .forEach((line, i) => {
          for (const [, washHue = '', alpha = ''] of line.matchAll(washRe)) {
            // /50 and up is a fill, whose text is an on-* token.
            if (Number(alpha) >= 50) continue;
            const family = FAMILY[washHue] ?? washHue;
            const where = `${relative(src, file)}:${i + 1}`;
            for (const [, textHue = ''] of line.matchAll(bare)) {
              if ((FAMILY[textHue] ?? textHue) === family) {
                offenders.push(`${where} text-${textHue} on bg-${washHue}/${alpha}`);
              }
            }
            if (Number(alpha) > MAX_WASH * 100 && line.includes(`text-${family}-strong`)) {
              offenders.push(`${where} bg-${washHue}/${alpha} is stronger than the measured /20`);
            }
          }
        });
    }
    expect([...new Set(offenders)]).toEqual([]);
  });
});
