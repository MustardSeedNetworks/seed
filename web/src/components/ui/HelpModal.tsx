import { ReactNode } from "react";

interface HelpModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

export function HelpModal({
  isOpen,
  onClose,
  title,
  children,
}: HelpModalProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal */}
      <div
        className="relative bg-surface-raised border border-surface-border rounded-lg shadow-xl max-w-2xl w-full max-h-modal overflow-hidden flex flex-col"
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-modal-title"
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-surface-border bg-surface-raised shrink-0">
          <h2
            id="help-modal-title"
            className="text-lg font-semibold text-text-primary"
          >
            {title}
          </h2>
          <button
            onClick={onClose}
            className="p-1 text-text-muted hover:text-text-primary transition-colors rounded hover:bg-surface-base"
            aria-label="Close help"
          >
            <svg className="w-5 h-5" viewBox="0 0 20 20" fill="currentColor">
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-4 overflow-y-auto flex-1">{children}</div>
      </div>
    </div>
  );
}

interface HelpSectionProps {
  title: string;
  children: ReactNode;
}

export function HelpSection({ title, children }: HelpSectionProps) {
  return (
    <div className="mb-6 last:mb-0">
      <h3 className="text-sm font-semibold text-text-primary mb-3 pb-1 border-b border-surface-border">
        {title}
      </h3>
      <div className="space-y-2">{children}</div>
    </div>
  );
}

interface HelpItemProps {
  term: string;
  description: string;
  color?: string;
}

export function HelpItem({ term, description, color }: HelpItemProps) {
  return (
    <div className="flex gap-3 text-sm">
      <div className="flex items-center gap-2 shrink-0 w-24">
        {color && (
          <span className={`inline-block w-2.5 h-2.5 rounded-full ${color}`} />
        )}
        <span className="font-medium text-text-primary">{term}</span>
      </div>
      <span className="text-text-muted">{description}</span>
    </div>
  );
}
