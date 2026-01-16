/**
 * HelpModal Component
 *
 * Purpose: Reusable modal dialog for displaying contextual help content.
 * Provides a standard layout with title, content area, close button, and backdrop.
 *
 * Key Features:
 * - Controlled visibility: isOpen prop controls display
 * - Backdrop: semi-transparent backdrop with blur effect
 * - Close handling: onClose callback, close button, backdrop click
 * - Accessibility: role="dialog", aria-modal, aria-labelledby for screen readers
 * - Fixed positioning: appears centered on screen with z-layer management
 * - Customizable content: ReactNode children for flexible content
 * - Semantic HTML: uses button with aria-label for close action
 *
 * Usage:
 * ```typescript
 * const [showHelp, setShowHelp] = useState(false);
 *
 * <HelpModal
 *   isOpen={showHelp}
 *   onClose={() => setShowHelp(false)}
 *   title="How to run a scan"
 * >
 *   <p>Click the start button to begin...</p>
 * </HelpModal>
 * ```
 *
 * Dependencies: React, theme utilities (icon tokens)
 * State: Receives visibility state from parent component
 */

import type React from "react";
import type { ReactNode } from "react";
import { cn, icon as iconTokens, layout, modal, radius, spacing } from "../../styles/theme";

interface HelpModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

/**
 * Modal dialog for displaying help and documentation content.
 */
export function HelpModal({
  isOpen,
  onClose,
  title,
  children,
}: HelpModalProps): React.JSX.Element | null {
  if (!isOpen) {
    return null;
  }

  return (
    <div class={modal.overlay}>
      {/* Backdrop */}
      <div class={modal.backdrop} onClick={onClose} aria-hidden="true" />

      {/* Modal */}
      <div
        class={cn(
          "relative",
          modal.content,
          modal.size.md,
          radius.lg,
          "flex",
          "flex-col",
          "overflow-hidden",
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-modal-title"
      >
        {/* Header */}
        <div
          class={cn(
            layout.flex.between,
            spacing.pad.default,
            "border-b",
            "border-surface-border",
            "bg-surface-raised",
            "shrink-0",
          )}
        >
          <h2 id="help-modal-title" class="heading-3">
            {title}
          </h2>
          <button
            type="button"
            onClick={onClose}
            class={cn(
              spacing.iconBtn.sm,
              "text-text-muted",
              "hover:text-text-primary",
              "transition-colors",
              radius.default,
              "hover:bg-surface-base",
            )}
            aria-label="Close help"
          >
            <svg
              class={iconTokens.size.md}
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div class={cn(spacing.pad.default, "overflow-y-auto", "flex-1")}>{children}</div>
      </div>
    </div>
  );
}

interface HelpSectionProps {
  title: string;
  children: ReactNode;
}

/**
 * Section container with title for organizing help content into logical groups.
 */
export function HelpSection({ title, children }: HelpSectionProps): React.JSX.Element {
  return (
    <div class={cn(spacing.margin.bottom.section, "last:mb-0")}>
      <h3
        class={cn(
          "heading-4",
          spacing.margin.bottom.heading,
          spacing.padding.bottom.tight,
          "border-b",
          "border-surface-border",
        )}
      >
        {title}
      </h3>
      <div class="stack-sm">{children}</div>
    </div>
  );
}

interface HelpItemProps {
  term: string;
  description: string;
  color?: string;
}

/**
 * Term-description pair with optional color indicator for help documentation.
 */
export function HelpItem({ term, description, color }: HelpItemProps): React.JSX.Element {
  return (
    <div class={cn("flex", spacing.gap.default, "body-small")}>
      <div class={cn("flex", "items-center", spacing.gap.compact, "shrink-0", "w-24")}>
        {color ? <span class={cn("inline-block", "w-2.5", "h-2.5", radius.full, color)} /> : null}{" "}
        {/* w-2.5 h-2.5 for status dot */}
        <span class="font-medium text-text-primary">{term}</span>
      </div>
      <span class="text-text-muted">{description}</span>
    </div>
  );
}
