import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import html from '../../index.html?raw';
import { initThemeFromStorage } from './useTheme';

function stubSystemDark(matches: boolean): void {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockReturnValue({
      matches,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }),
  );
}

beforeEach(() => {
  localStorage.clear();
  document.documentElement.classList.remove('dark');
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('initThemeFromStorage', () => {
  it('leaves <html> light when light is stored', () => {
    document.documentElement.classList.add('dark');
    localStorage.setItem('seed-theme', 'light');
    initThemeFromStorage();
    expect(document.documentElement.classList.contains('dark')).toBe(false);
  });

  it('adds dark when dark is stored', () => {
    localStorage.setItem('seed-theme', 'dark');
    initThemeFromStorage();
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it.each([
    [true, true],
    [false, false],
  ])('follows prefers-color-scheme (dark=%s) when system is stored', (systemDark, expected) => {
    stubSystemDark(systemDark);
    localStorage.setItem('seed-theme', 'system');
    initThemeFromStorage();
    expect(document.documentElement.classList.contains('dark')).toBe(expected);
  });

  it('defaults to dark when nothing, or an unknown value, is stored', () => {
    initThemeFromStorage();
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    document.documentElement.classList.remove('dark');
    localStorage.setItem('seed-theme', 'sepia');
    initThemeFromStorage();
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it('defaults to dark when storage throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError');
    });
    initThemeFromStorage();
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });
});

describe('index.html', () => {
  it('does not hard-code a theme class on <html>', () => {
    expect(html).not.toMatch(/<html[^>]*class=/);
  });

  it('has no inline script, which the script-src self CSP would block', () => {
    expect(html).not.toMatch(/<script(?![^>]*\bsrc=)/);
  });
});
