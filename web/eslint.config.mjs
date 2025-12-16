// For more info, see https://github.com/storybookjs/eslint-plugin-storybook#configuration-flat-config-format
import storybook from "eslint-plugin-storybook";

/**
 * ESLint Configuration
 *
 * Purpose: Configures ESLint rules and plugins for code quality in The Seed frontend.
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
import security from "eslint-plugin-security";
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
    security: security,
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
    // Security rules - prevent common vulnerabilities
    "security/detect-object-injection": "warn",       // Dynamic property access
    "security/detect-non-literal-regexp": "warn",     // Dynamic regex construction
    "security/detect-unsafe-regex": "warn",           // ReDoS vulnerable patterns
    "security/detect-buffer-noassert": "error",       // Buffer without bounds check
    "security/detect-eval-with-expression": "error",  // eval() with variables
    "security/detect-no-csrf-before-method-override": "error",
    "security/detect-possible-timing-attacks": "warn", // Timing side channels
    "security/detect-child-process": "warn",          // Child process execution
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
      // Enforce theme tokens for icon sizes instead of hardcoded w-N h-N patterns
      {
        selector: "Literal[value=/className.*\\bw-3\\s+h-3\\b/]",
        message: "Use iconTokens.size.xs from theme.ts instead of 'w-3 h-3'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\bw-4\\s+h-4\\b/]",
        message: "Use iconTokens.size.sm from theme.ts instead of 'w-4 h-4'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\bw-5\\s+h-5\\b/]",
        message: "Use iconTokens.size.md from theme.ts instead of 'w-5 h-5'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\bw-6\\s+h-6\\b/]",
        message: "Use iconTokens.size.lg from theme.ts instead of 'w-6 h-6'. See web/THEMING.md",
      },
      // Enforce theme tokens for border radius instead of hardcoded rounded-* patterns
      {
        selector: "Literal[value=/className.*\\brounded-sm\\b/]",
        message: "Use radius.sm from theme.ts instead of 'rounded-sm'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\brounded-md\\b/]",
        message: "Use radius.md from theme.ts instead of 'rounded-md'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\brounded-lg\\b/]",
        message: "Use radius.lg from theme.ts instead of 'rounded-lg'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\brounded-xl\\b/]",
        message: "Use radius.xl from theme.ts instead of 'rounded-xl'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\brounded-full\\b/]",
        message: "Use radius.full from theme.ts instead of 'rounded-full'. See web/THEMING.md",
      },
      // Enforce theme tokens for large padding values (p-4 and above)
      // Small values (p-1, p-2, p-3) are allowed for granular spacing
      {
        selector: "Literal[value=/className.*\\bp-[4-9]\\b/]",
        message: "Use 'pad' or 'pad-lg' from theme.ts instead of 'p-4+'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\bp-1[0-9]\\b/]",
        message: "Use 'pad-lg' from theme.ts instead of 'p-10+'. See web/THEMING.md",
      },
      // Enforce theme tokens for large gap values (gap-4 and above)
      // Small gaps (gap-1, gap-2, gap-3) are allowed for tight layouts
      {
        selector: "Literal[value=/className.*\\bgap-[4-9]\\b/]",
        message: "Use spacing.gap.comfortable or spacing.gap.spacious from theme.ts instead of 'gap-4+'. See web/THEMING.md",
      },
      // Enforce button size tokens for button padding patterns
      {
        selector: "Literal[value=/className.*\\bpx-4\\s+py-2\\b/]",
        message: "Use button.size.md from theme.ts instead of 'px-4 py-2'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\bpx-3\\s+py-1\\.?5?\\b/]",
        message: "Use button.size.sm from theme.ts instead of 'px-3 py-1.5'. See web/THEMING.md",
      },
      {
        selector: "Literal[value=/className.*\\bpx-2\\s+py-1\\b/]",
        message: "Use button.size.xs from theme.ts instead of 'px-2 py-1'. See web/THEMING.md",
      },
      // i18n: Detect hardcoded English text in common UI patterns
      // This catches common patterns like <span>Settings</span> or <button>Save</button>
      // Note: This is a heuristic - some false positives may occur for technical terms
      {
        selector: "JSXElement[openingElement.name.name=/^(span|button|label|h1|h2|h3|h4|h5|h6|p|li|th|td|option)$/] > JSXText[value=/[A-Z][a-z]{2,}/]",
        message: "Possible hardcoded UI text detected. Consider using t() from useTranslation for i18n support. See i18n documentation.",
      },
    ],
  },
},
// Storybook configuration
storybook.configs["flat/recommended"],
// Test file overrides - relax rules for test files
{
  files: ["**/*.test.ts", "**/*.test.tsx", "**/*.spec.ts", "**/test/**/*.ts", "**/e2e/**/*.ts"],
  rules: {
    // Disable JSDoc requirements in tests
    "jsdoc/require-jsdoc": "off",
    "jsdoc/require-description": "off",
    "jsdoc/check-indentation": "off",
    // Disable i18n warnings in tests (test strings don't need translation)
    "no-restricted-syntax": "off",
    // Relax security rules in tests (test mocks often use dynamic patterns)
    "security/detect-object-injection": "off",
    "security/detect-unsafe-regex": "off",
    "security/detect-non-literal-regexp": "off",
    // Allow any in test mocks
    "@typescript-eslint/no-explicit-any": "off",
  },
},
// Storybook stories overrides
{
  files: ["**/*.stories.tsx", "**/*.stories.ts"],
  rules: {
    // Disable i18n warnings in stories (story descriptions don't need translation)
    "no-restricted-syntax": "off",
    // Stories often use helper components that aren't exported
    "react-refresh/only-export-components": "off",
    // Relax JSDoc in stories
    "jsdoc/require-jsdoc": "off",
    "jsdoc/require-description": "off",
    // Safe dynamic access in story demos
    "security/detect-object-injection": "off",
  },
},
// Storybook preview overrides
{
  files: ["**/.storybook/**/*.tsx", "**/.storybook/**/*.ts"],
  rules: {
    // Preview file has helper components
    "react-refresh/only-export-components": "off",
    "jsdoc/require-jsdoc": "off",
  },
},
// Test setup and utilities
{
  files: ["**/test/setup.ts", "**/test/**/*.ts"],
  rules: {
    "jsdoc/require-jsdoc": "off",
    "jsdoc/require-description": "off",
    "security/detect-object-injection": "off",
  },
},
);
