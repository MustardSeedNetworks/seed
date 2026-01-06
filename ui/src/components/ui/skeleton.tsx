/**
 * Skeleton Component
 *
 * Purpose: Provides reusable loading placeholder components for data that hasn't loaded yet.
 * Uses CSS animation to create a pulsing skeleton effect while content is being fetched.
 *
 * Key Features:
 * - Multiple variants: text (rounded), circular (for avatars), rectangular (for images/blocks)
 * - Flexible sizing via width/height props (accepts px numbers or string values)
 * - CardSkeleton: Pre-configured skeleton for card layouts with title, rows, and value skeletons
 * - Accessible: Uses aria-hidden="true" to hide from screen readers during loading
 *
 * Usage:
 * ```typescript
 * // Text skeleton (for paragraphs)
 * <Skeleton variant="text" className="h-4 w-32" />
 *
 * // Circular skeleton (for avatars)
 * <Skeleton variant="circular" width={40} height={40} />
 *
 * // Rectangular skeleton (for images)
 * <Skeleton variant="rectangular" width={200} height={150} />
 *
 * // Full card skeleton
 * <CardSkeleton />
 * ```
 *
 * Dependencies: theme utilities (cn, radius, card, layout), React
 * State: None - purely presentational component
 */

import { card, cn, layout, radius, spacing } from "../../styles/theme";

interface SkeletonProps {
  className?: string;
  variant?: "text" | "circular" | "rectangular";
  width?: string | number;
  height?: string | number;
}

/**
 * Animated placeholder component for loading states with configurable shape.
 */
export function Skeleton({ className = "", variant = "text", width, height }: SkeletonProps) {
  const baseClasses = "animate-pulse bg-surface-hover";

  // Type-safe variant class getter
  function getVariantClass(v: typeof variant) {
    switch (v) {
      case "text":
        return radius.default;
      case "circular":
        return radius.full;
      case "rectangular":
        return radius.lg;
    }
  }

  const sizeClasses = [
    width ? (typeof width === "number" ? `w-[${width}px]` : `w-[${width}]`) : "",
    height ? (typeof height === "number" ? `h-[${height}px]` : `h-[${height}]`) : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      className={cn(baseClasses, getVariantClass(variant), sizeClasses, className)}
      aria-hidden="true"
    />
  );
}

/**
 * Pre-configured skeleton matching the Card component layout.
 */
export function CardSkeleton() {
  return (
    <div className={cn(card.base, card.variant.default, card.padding.md)}>
      <div className={cn(layout.flex.between, spacing.margin.bottom.heading)}>
        <Skeleton className="h-4 w-24" />
        <Skeleton variant="circular" className="h-3 w-3" />
      </div>
      <Skeleton className={cn("h-8 w-32", spacing.margin.bottom.inline)} />
      <div className={cn("stack-sm", spacing.margin.top.content)}>
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-16" />
        </div>
      </div>
    </div>
  );
}

/**
 * Multi-line text placeholder for paragraph loading states.
 */
export function TextSkeleton({ lines = 3 }: { lines?: number }) {
  // Generate stable unique IDs for skeleton lines
  // Each line has a unique ID based on its position descriptor (line-1-of-3, line-2-of-3, etc.)
  // This avoids using array index directly as key while maintaining stable identity
  const lineConfigs = Array.from({ length: lines }, (_, i) => ({
    id: `line-${i + 1}-of-${lines}`,
    width: i === lines - 1 ? "60%" : "100%",
  }));

  return (
    <div className="stack-sm">
      {lineConfigs.map((config) => (
        <Skeleton key={config.id} className="h-4" width={config.width} />
      ))}
    </div>
  );
}
