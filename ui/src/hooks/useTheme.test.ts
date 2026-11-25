import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import html from '../../index.html?raw';
import { initThemeFromStorage, useTheme } from './useTheme';

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

  it.each([
    [true, true],
    [false, false],
  ])(
    'follows the OS (dark=%s) when nothing, or an unknown value, is stored',
    (systemDark, expected) => {
      stubSystemDark(systemDark);
      initThemeFromStorage();
      expect(document.documentElement.classList.contains('dark')).toBe(expected);
      document.documentElement.classList.remove('dark');
      localStorage.setItem('seed-theme', 'sepia');
      initThemeFromStorage();
      expect(document.documentElement.classList.contains('dark')).toBe(expected);
    },
  );

  it('follows the OS when storage throws', () => {
    stubSystemDark(false);
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError');
    });
    initThemeFromStorage();
    expect(document.documentElement.classList.contains('dark')).toBe(false);
  });
});

describe('useTheme store', () => {
  it('shares one state between the rail toggle and the Settings control', () => {
    stubSystemDark(false);
    initThemeFromStorage();
    const rail = renderHook(() => useTheme());
    const settings = renderHook(() => useTheme());
    expect(settings.result.current.theme).toBe('system');
    act(() => rail.result.current.toggleTheme());
    expect(settings.result.current.theme).toBe('dark');
    expect(settings.result.current.isDark).toBe(true);
    act(() => settings.result.current.setTheme('light'));
    expect(rail.result.current.isDark).toBe(false);
    expect(localStorage.getItem('seed-theme')).toBe('light');
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
