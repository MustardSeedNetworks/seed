import type { Meta, StoryFn, StoryObj } from "@storybook/react-vite";
import type React from "react";
import { button, cn, layout, radius } from "../../styles/theme";
import { ToastProvider } from "./Toast";
import { useToast } from "./useToast";

/**
 * Toast notifications provide non-modal feedback to users for actions
 * like success messages, errors, warnings, and informational updates.
 *
 * Toasts appear in the bottom-right corner and auto-dismiss after 5 seconds by default.
 */
const meta: Meta = {
  title: "UI/Toast",
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <ToastProvider>
        <StoryComponent />
      </ToastProvider>
    ),
  ],
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;

function _toastDemo({
  type,
  message,
}: {
  type: "success" | "error" | "warning" | "info";
  message: string;
}): React.JSX.Element {
  const { addToast } = useToast();

  return (
    <button
      type="button"
      onClick={() => addToast(message, type)}
      class={cn(
        button.size.md,
        "bg-surface-raised hover:bg-surface-hover border border-surface-border text-text-primary",
        radius.lg,
      )}
    >
      Show {type} toast
    </button>
  );
}

export const Success: StoryObj = {
  render: () => <toastDemo type="success" message="Operation completed successfully!" />,
};

export const ErrorToast: StoryObj = {
  render: () => <toastDemo type="error" message="An error occurred. Please try again." />,
};

export const Warning: StoryObj = {
  render: () => <toastDemo type="warning" message="This action cannot be undone." />,
};

export const Info: StoryObj = {
  render: () => <toastDemo type="info" message="Network scan is in progress..." />,
};

function _allToastsDemo(): React.JSX.Element {
  const { addToast } = useToast();

  return (
    <div class={layout.stack.default}>
      <button
        type="button"
        onClick={() => {
          addToast("Success message", "success", 3000);
          setTimeout(() => addToast("Error message", "error", 3000), 500);
          setTimeout(() => addToast("Warning message", "warning", 3000), 1000);
          setTimeout(() => addToast("Info message", "info", 3000), 1500);
        }}
        class={cn(
          button.size.md,
          "bg-status-info hover:bg-status-info/80 text-text-inverse",
          radius.lg,
        )}
      >
        Show all toast types
      </button>
    </div>
  );
}

export const AllTypes: StoryObj = {
  render: () => <allToastsDemo />,
  parameters: {
    docs: {
      description: {
        story: "Click to see all toast types in sequence.",
      },
    },
  },
};
