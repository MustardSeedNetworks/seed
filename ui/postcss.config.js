/**
 * PostCSS Configuration
 *
 * Purpose: Configures PostCSS and Tailwind CSS processing for The Seed frontend.
 * Processes CSS files during build to generate utility classes and optimize output.
 *
 * Plugin: @tailwindcss/postcss
 * - Automatically generates Tailwind utility classes from usage
 * - Processes @tailwind directives in CSS files
 * - Applies PurgeCSS-like tree-shaking (unused classes removed in production)
 * - Handles theme configuration from tailwind.config.js
 *
 * Build Pipeline:
 * 1. Source CSS imports @tailwind directives
 * 2. PostCSS processes files and loads Tailwind plugin
 * 3. Tailwind scans source files for class usage
 * 4. Generates only used utility classes
 * 5. Output is minified for production
 *
 * CSS Source Files:
 * - web/src/index.css - Main CSS file with @tailwind directives
 * - Component styles via Tailwind utility classes in className props
 *
 * Configuration Files:
 * - tailwind.config.js - Customizes theme, colors, spacing, etc.
 * - web/src/styles/theme.ts - Exports design tokens matching Tailwind config
 *
 * Usage:
 * ```css
 * @tailwind base;
 * @tailwind components;
 * @tailwind utilities;
 * ```
 *
 * Dependencies: postcss, @tailwindcss/postcss, tailwindcss
 */

export default {
  plugins: {
    '@tailwindcss/postcss': {},
  },
};
