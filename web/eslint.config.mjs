// For more info, see https://github.com/storybookjs/eslint-plugin-storybook#configuration-flat-config-format
import storybook from "eslint-plugin-storybook";

/**
 * ESLint Configuration
 * 
 * Purpose: Configures ESLint rules and plugins for code quality in LuminetIQ frontend.
 * Enforces TypeScript, React, and accessibility best practices across the codebase.
 * 
 * Configuration:
 * - Base: JavaScript recommended config with TypeScript ESLint support
 * - Language: ECMAScript 2024 with JSX/TSX support
 * - Parser: typescript-eslint for TypeScript support
 * - Plugins:
 *   - react-hooks: Validates Hook usage (exhaustive-deps, rules of hooks)
 *   - react-refresh: Warns if exports might break Fast Refresh
 *   - typescript-eslint: Type-aware linting rules
 * 
 * Key Rules:
 * - react-hooks/rules-of-hooks: Enforces rules of hooks (no hooks in loops)
 * - react-hooks/exhaustive-deps: Validates dependency arrays
 * - react-refresh/only-export-components: Exports should be components (with exceptions)
 * - TypeScript: Recommends stricter type checking rules
 * 
 * Ignored Directories:
 * - dist/ - Build output
 * - node_modules/ - Dependencies
 * - coverage/ - Test coverage reports
 * 
 * Usage:
 * ```bash
 * npm run lint              # Run ESLint check
 * npm run lint:fix          # Auto-fix lint issues
 * npm run lint -- --max-warnings 0  # Treat warnings as errors in CI/CD
 * ```
 * 
 * Configuration Format:
 * - Uses flat config format (ESLint v9+)
 * - Module type: ES modules (.mjs extension)
 * 
 * Dependencies: eslint, typescript-eslint, eslint-plugin-react-hooks, eslint-plugin-react-refresh
 * IDE Integration: Automatically applies rules in VS Code with ESLint extension
 */

import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import jsdoc from "eslint-plugin-jsdoc";
import tseslint from "typescript-eslint";

export default tseslint.config({ ignores: ["dist", "node_modules", "coverage", "storybook-static", "playwright-report"] }, {
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
    jsdoc: jsdoc,
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
    // Naming conventions for TypeScript
    "@typescript-eslint/naming-convention": [
      "warn",
      // Variables: camelCase or UPPER_CASE (for constants), PascalCase for React components
      {
        selector: "variable",
        format: ["camelCase", "UPPER_CASE", "PascalCase"],
        leadingUnderscore: "allow",
        // Allow __dirname, __filename (Node.js special variables)
        filter: { regex: "^__", match: false },
      },
      // Functions: camelCase (React components can be PascalCase)
      {
        selector: "function",
        format: ["camelCase", "PascalCase"],
      },
      // Parameters: camelCase (allow PascalCase for React component params like Story)
      {
        selector: "parameter",
        format: ["camelCase", "PascalCase"],
        leadingUnderscore: "allow",
      },
      // Types, interfaces, classes: PascalCase
      {
        selector: "typeLike",
        format: ["PascalCase"],
      },
      // Enum members: PascalCase or UPPER_CASE
      {
        selector: "enumMember",
        format: ["PascalCase", "UPPER_CASE"],
      },
      // Object properties: flexible for API compatibility
      {
        selector: "property",
        format: null, // Allow any format for properties (HTTP headers, API responses, etc.)
        leadingUnderscore: "allow",
      },
      // Type properties: flexible for API types
      {
        selector: "typeProperty",
        format: null, // Allow any format (WebSocket __ws, API snake_case, etc.)
        leadingUnderscore: "allow",
      },
    ],
    "no-console": ["warn", { allow: ["warn", "error"] }],
    // JSDoc documentation rules (warn only - encourage documentation)
    "jsdoc/require-description": ["warn", { contexts: ["FunctionDeclaration", "ClassDeclaration"] }],
    "jsdoc/require-jsdoc": [
      "warn",
      {
        publicOnly: true,
        require: {
          FunctionDeclaration: true,
          ClassDeclaration: true,
          MethodDefinition: false,
          ArrowFunctionExpression: false,
        },
        contexts: [
          // Require JSDoc on exported function declarations
          "ExportNamedDeclaration > FunctionDeclaration",
          // Require JSDoc on exported arrow functions assigned to variables
          "ExportNamedDeclaration > VariableDeclaration > VariableDeclarator > ArrowFunctionExpression",
        ],
      },
    ],
    "jsdoc/check-alignment": "warn",
    "jsdoc/check-indentation": "warn",
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
}, storybook.configs["flat/recommended"]);
