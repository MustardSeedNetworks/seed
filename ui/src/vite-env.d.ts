/**
 * Vite Environment Type Definitions
 *
 * Purpose: Defines TypeScript types for Vite environment variables accessed via import.meta.env.
 * Provides type safety for build-time configuration and environment-specific settings.
 *
 * Environment Variables:
 * - VITE_API_BASE: Base URL for API requests (backend server address)
 * Set during build: `VITE_API_BASE=http://localhost:3000 npm run build`
 * Default runtime fallback: "" (same origin)
 *
 * Usage:
 * ```typescript
 * const apiBase = import.meta.env.VITE_API_BASE || "";
 * const apiUrl = `${apiBase}/api/config`;
 * ```
 *
 * Configuration Files:
 * - .env: Default environment variables
 * - .env.local: Local overrides (git-ignored)
 * - .env.production: Production-specific variables
 *
 * Dependencies: Vite client types
 */

/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
