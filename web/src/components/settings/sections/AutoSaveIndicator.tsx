import { SaveStatus } from "../../../types/settings";

interface AutoSaveIndicatorProps {
  status: SaveStatus;
}

export function AutoSaveIndicator({ status }: AutoSaveIndicatorProps) {
  if (status === "idle") return null;
  return (
    <span
      className={`text-xs ml-2 ${
        status === "saving"
          ? "text-text-muted"
          : status === "saved"
            ? "text-status-success"
            : "text-status-error"
      }`}
    >
      {status === "saving"
        ? "Saving..."
        : status === "saved"
          ? "Saved"
          : "Error"}
    </span>
  );
}
