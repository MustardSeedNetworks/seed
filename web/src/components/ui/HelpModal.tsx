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

import { ReactNode } from "react";
import { icon as iconTokens, layout, radius, modal, spacing } from "../../styles/theme";

interface HelpModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

/**
 * Modal dialog for displaying help and documentation content.
 */
export function HelpModal({ isOpen, onClose, title, children }: HelpModalProps) {
  if (!isOpen) return null;

  return (
    <div className={`fixed inset-0 z-50 flex items-center justify-center ${spacing.pad.default}`}>
      {/* Backdrop */}
      <div
        className={`absolute inset-0 ${modal.overlay} backdrop-blur-sm`}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal */}
      <div
        className={`relative bg-surface-raised border border-surface-border ${radius.lg} shadow-xl max-w-2xl w-full max-h-modal overflow-hidden flex flex-col`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-modal-title"
      >
        {/* Header */}
        <div
          className={`${layout.flex.between} ${spacing.pad.default} border-b border-surface-border bg-surface-raised shrink-0`}
        >
          <h2 id="help-modal-title" className="heading-3">
            {title}
          </h2>
          <button
            onClick={onClose}
            className={`p-1 text-text-muted hover:text-text-primary transition-colors ${radius.default} hover:bg-surface-base`}
            aria-label="Close help"
          >
            <svg className={iconTokens.size.md} viewBox="0 0 20 20" fill="currentColor">
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className={`${spacing.pad.default} overflow-y-auto flex-1`}>{children}</div>
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
export function HelpSection({ title, children }: HelpSectionProps) {
  return (
    <div className={`${spacing.margin.bottom.section} last:mb-0`}>
      <h3
        className={`heading-4 ${spacing.margin.bottom.heading} pb-1 border-b border-surface-border`}
      >
        {title}
      </h3>
      <div className="stack-sm">{children}</div>
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
export function HelpItem({ term, description, color }: HelpItemProps) {
  return (
    <div className={`flex ${spacing.gap.default} body-small`}>
      <div className="flex items-center gap-2 shrink-0 w-24">
        {color && <span className={`inline-block w-2.5 h-2.5 ${radius.full} ${color}`} />}
        <span className="font-medium text-text-primary">{term}</span>
      </div>
      <span className="text-text-muted">{description}</span>
    </div>
  );
}
