/**
 * Type Shims
 *
 * Purpose: Provides TypeScript type declarations for modules that don't have built-in type definitions.
 * Prevents TypeScript errors when importing modules without @types packages.
 *
 * Module Declarations:
 * - react-dom/client: React 18+ client rendering API type definitions
 *
 * Usage: Automatically applied by TypeScript compiler, no explicit imports needed
 *
 * Note: This file is useful for third-party modules or build outputs that lack type definitions.
 * Consider using @types packages when available for better type coverage.
 */

declare module "react-dom/client";
