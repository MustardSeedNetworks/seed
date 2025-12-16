/**
 * Vite Build Configuration
 *
 * Purpose: Configures the Vite development server and build process for the The Seed web frontend.
 * Handles bundling, module resolution, and development server settings.
 *
 * Configuration:
 * - React plugin: Enables JSX/TSX transformation and fast refresh during development
 * - Path alias: @ resolves to src/ directory for cleaner imports
 * - Dev server: Runs on port 3000 with HMR support
 * - Build output: Compiled to dist/ directory with source maps for debugging
 * - Embedding: Compiled frontend is embedded in Go binary via //go:embed directive
 *
 * Build Process:
 * 1. TypeScript compilation and bundling
 * 2. CSS processing and minification
 * 3. Asset optimization and tree-shaking
 * 4. Source map generation for production debugging
 * 5. Output to dist/ for Go embedding
 *
 * Usage:
 * ```bash
 * npm run dev     # Start dev server on port 3000
 * npm run build   # Build for production
 * npm run preview # Preview production build locally
 * ```
 *
 * Dependencies: vite, @vitejs/plugin-react
 * See: web/embed.go for how dist/ is embedded in the Go binary
 */
declare const _default: import("vite").UserConfig;
export default _default;
