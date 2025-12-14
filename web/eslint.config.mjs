import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage"] },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: {
      ecmaVersion: 2024,
      globals: globals.browser,
      parser: tseslint.parser, // Explicitly set the parser
      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
      },
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/no-explicit-any": "warn",
      "no-console": ["warn", { allow: ["warn", "error"] }],
      // Enforce design system colors - prevent hardcoded grayscale/black/white in className
      // Note: Colored variants (blue, green, red, etc.) are allowed when used semantically
      // (e.g., discovery method badges, category icons) but should use dark: variants
      "no-restricted-syntax": [
        "warn",
        {
          selector: "Literal[value=/\\btext-white\\b/]",
          message: "Use 'text-text-inverse' instead of 'text-white'. See web/THEMING.md",
        },
        {
          selector: "Literal[value=/\\btext-black\\b/]",
          message: "Use 'text-text-primary' instead of 'text-black'. See web/THEMING.md",
        },
        {
          // Flag bg-white followed by space or end of string (not opacity variants like bg-white/20)
          selector: "Literal[value=/\\bbg-white(\\s|$)/]",
          message: "Use 'bg-surface-raised' or 'bg-surface-base' instead of 'bg-white'. Opacity variants like 'bg-white/20' are allowed for hover effects. See web/THEMING.md",
        },
        {
          // Flag bg-black followed by space or end of string (not opacity variants like bg-black/50)
          selector: "Literal[value=/\\bbg-black(\\s|$)/]",
          message: "Use design system tokens instead of 'bg-black'. Opacity variants like 'bg-black/50' are allowed for overlays. See web/THEMING.md",
        },
        {
          selector: "Literal[value=/\\btext-gray-\\d/]",
          message: "Use design system tokens (text-text-primary, text-text-secondary, text-text-muted) instead of text-gray-*. See web/THEMING.md",
        },
        {
          selector: "Literal[value=/\\bbg-gray-\\d/]",
          message: "Use design system tokens (bg-surface-base, bg-surface-raised, bg-surface-hover) instead of bg-gray-*. See web/THEMING.md",
        },
      ],
    },
  },
);
