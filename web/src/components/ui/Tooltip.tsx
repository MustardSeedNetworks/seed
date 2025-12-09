import { ReactNode, useState } from "react";

interface TooltipProps {
  content: string;
  children: ReactNode;
  position?: "top" | "bottom";
}

export function Tooltip({ content, children, position = "top" }: TooltipProps) {
  const [show, setShow] = useState(false);

  const positionClasses =
    position === "top"
      ? "bottom-full left-1/2 -translate-x-1/2 mb-2"
      : "top-full left-1/2 -translate-x-1/2 mt-2";

  return (
    <div className="relative inline-flex items-center">
      <div
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        onFocus={() => setShow(true)}
        onBlur={() => setShow(false)}
        className="cursor-help"
      >
        {children}
      </div>
      {show && (
        <div
          className={`absolute z-50 ${positionClasses} px-2 py-1.5 text-xs bg-surface-raised border border-surface-border rounded shadow-lg max-w-xs text-text-primary`}
          role="tooltip"
        >
          {content}
        </div>
      )}
    </div>
  );
}
