/**
 * Theme Management Hook
 *
 * Manages application theme with support for light, dark, and system modes.
 *
 * Features:
 * - Light and dark theme modes
 * - Automatic system theme detection
 * - Persistent theme storage in localStorage
 * - Automatic system theme change detection
 * - Theme toggling functionality
 *
 * The theme is applied by adding/removing the 'dark' class on the document root element,
 * which Tailwind CSS uses for dark mode styling.
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

import { useState, useEffect, useCallback } from "react";

/** Theme mode options */
type Theme = "light" | "dark" | "system";

/** localStorage key for theme persistence */
const STORAGE_KEY = "seed-theme";

/**
 * Detects the system's preferred color scheme.
 *
 * @returns 'dark' if system prefers dark mode, 'light' otherwise
 */
function getSystemTheme(): "light" | "dark" {
  if (typeof window !== "undefined" && window.matchMedia) {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  return "dark"; // Default to dark if unable to detect
}

/**
 * Applies the theme to the document root element.
 * Resolves 'system' theme to actual light/dark preference.
 *
 * @param theme - Theme to apply ('light', 'dark', or 'system')
 */
function applyTheme(theme: Theme) {
  const root = document.documentElement;
  const effectiveTheme = theme === "system" ? getSystemTheme() : theme;

  // Tailwind dark mode uses the 'dark' class on root element
  if (effectiveTheme === "dark") {
    root.classList.add("dark");
  } else {
    root.classList.remove("dark");
  }
}

/**
 * Custom hook for managing application theme.
 *
 * Handles theme persistence, system theme detection, and automatic updates
 * when system theme changes.
 *
 * @returns Theme state and control functions
 */
export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(() => {
    if (typeof window !== "undefined") {
      const stored = localStorage.getItem(STORAGE_KEY) as Theme | null;
      return stored || "dark";
    }
    return "dark";
  });

  const [effectiveTheme, setEffectiveTheme] = useState<"light" | "dark">(() => {
    return theme === "system" ? getSystemTheme() : theme;
  });

  const setTheme = useCallback((newTheme: Theme) => {
    setThemeState(newTheme);
    localStorage.setItem(STORAGE_KEY, newTheme);
    applyTheme(newTheme);
    setEffectiveTheme(newTheme === "system" ? getSystemTheme() : newTheme);
  }, []);

  const toggleTheme = useCallback(() => {
    const newTheme = effectiveTheme === "dark" ? "light" : "dark";
    setTheme(newTheme);
  }, [effectiveTheme, setTheme]);

  // Apply theme on mount and listen for system theme changes
  useEffect(() => {
    applyTheme(theme);

    if (theme === "system") {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      const handler = (e: MediaQueryListEvent) => {
        setEffectiveTheme(e.matches ? "dark" : "light");
        applyTheme("system");
      };
      mediaQuery.addEventListener("change", handler);
      return () => mediaQuery.removeEventListener("change", handler);
    }
  }, [theme]);

  return {
    theme,
    effectiveTheme,
    setTheme,
    toggleTheme,
    isDark: effectiveTheme === "dark",
  };
}
