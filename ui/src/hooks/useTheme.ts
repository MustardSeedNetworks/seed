/**
 * Theme Management Hook
 *
 * Manages application theme with support for light, dark, and system modes.
 *
 * Features:
 * - Light and dark theme modes
 * - Automatic system theme detection (the default: with nothing stored, the
 *   theme follows the OS)
 * - One store: the Settings control and the rail toggle read and write the
 *   same state, so neither can show a stale choice
 * - Persistent theme storage in localStorage
 * - Automatic system theme change detection
 * - Theme toggling functionality
 *
 * The theme is applied by adding/removing the 'dark' class on the document root element,
 * which Tailwind CSS uses for dark mode styling.
 *
 * Persistence — deliberate exception (#458). Every other setting persists to
 * the backend via `/api/v1/settings`; theme does not, and stays per-browser in
 * localStorage. Seed is a portable diagnostic driven from whatever laptop or
 * phone is to hand, so a light/dark choice made on a phone in a dim IDF should
 * not follow the operator to a bright office monitor. Cross-device sync would
 * make that behaviour worse, not better. Do not "fix" this by moving it into
 * the settings API without revisiting that decision.
 *
 * Usage:
 * ```typescript
 * const { theme, isDark, toggleTheme, setTheme } = useTheme();
 *
 * // Toggle between light and dark
 * <button onClick={toggleTheme}>Toggle Theme</button>
 *
 * // Set specific theme
 * <button onClick={() => setTheme('system')}>Use System Theme</button>
 * ```
 */

import { useSyncExternalStore } from 'react';

/** Theme mode options */
export type Theme = 'light' | 'dark' | 'system';

/** localStorage key for theme persistence */
const STORAGE_KEY = 'seed-theme';

/**
 * Detects the system's preferred color scheme.
 *
 * @returns 'dark' if system prefers dark mode, 'light' otherwise
 */
function getSystemTheme(): 'light' | 'dark' {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'dark'; // Default to dark if unable to detect
}

function resolve(theme: Theme): 'light' | 'dark' {
  return theme === 'system' ? getSystemTheme() : theme;
}

/** Applies the resolved theme to the document root (Tailwind's 'dark' class). */
function applyTheme(theme: Theme): void {
  document.documentElement.classList.toggle('dark', resolve(theme) === 'dark');
}

/**
 * Reads the persisted theme. Anything unreadable (storage blocked, or a value
 * this build never wrote) falls back to system, the default for a first visit.
 */
function readStoredTheme(): Theme {
  try {
    const stored: string | null = localStorage.getItem(STORAGE_KEY);
    return stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system';
  } catch {
    return 'system';
  }
}

interface ThemeSnapshot {
  theme: Theme;
  effectiveTheme: 'light' | 'dark';
}

let snapshot: ThemeSnapshot | undefined;
const listeners = new Set<() => void>();
let systemQuery: MediaQueryList | undefined;

function current(): ThemeSnapshot {
  if (!snapshot) {
    const theme = readStoredTheme();
    snapshot = { theme, effectiveTheme: resolve(theme) };
  }
  return snapshot;
}

function commit(theme: Theme): void {
  snapshot = { theme, effectiveTheme: resolve(theme) };
  applyTheme(theme);
  for (const listener of listeners) {
    listener();
  }
}

function onSystemChange(): void {
  if (current().theme === 'system') {
    commit('system');
  }
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  if (listeners.size === 1 && typeof window !== 'undefined' && window.matchMedia) {
    systemQuery = window.matchMedia('(prefers-color-scheme: dark)');
    systemQuery.addEventListener('change', onSystemChange);
  }
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) {
      systemQuery?.removeEventListener('change', onSystemChange);
      systemQuery = undefined;
    }
  };
}

/**
 * Applies the persisted theme to <html>. Called from main.tsx before anything
 * else loads, so the first paint already has the right palette. index.html
 * cannot do it: the CSP is `script-src 'self'`, which forbids an inline
 * script, and the markup carries no theme class of its own.
 */
export function initThemeFromStorage(): void {
  snapshot = undefined;
  commit(readStoredTheme());
}

function setTheme(newTheme: Theme): void {
  try {
    localStorage.setItem(STORAGE_KEY, newTheme);
  } catch {
    // Storage blocked: the choice still applies for this page's lifetime.
  }
  commit(newTheme);
}

function toggleTheme(): void {
  setTheme(current().effectiveTheme === 'dark' ? 'light' : 'dark');
}

/**
 * Theme state and controls. Every caller shares one store.
 *
 * @returns Theme state and control functions
 */
export function useTheme(): {
  theme: Theme;
  effectiveTheme: 'light' | 'dark';
  setTheme: (newTheme: Theme) => void;
  toggleTheme: () => void;
  isDark: boolean;
} {
  const { theme, effectiveTheme } = useSyncExternalStore(subscribe, current, current);
  return { theme, effectiveTheme, setTheme, toggleTheme, isDark: effectiveTheme === 'dark' };
}
