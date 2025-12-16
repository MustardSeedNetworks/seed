import type { Meta, StoryObj } from "@storybook/react-vite";
import { ToastProvider } from "./Toast";
import { useToast } from "./useToast";
import { button, radius, layout } from "../../styles/theme";

/**
 * Toast notifications provide non-modal feedback to users for actions
 * like success messages, errors, warnings, and informational updates.
 *
 * Toasts appear in the bottom-right corner and auto-dismiss after 5 seconds by default.
 */
const meta: Meta = {
  title: "UI/Toast",
  decorators: [
    (Story) => (
      <ToastProvider>
        <Story />
      </ToastProvider>
    ),
  ],
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;

function ToastDemo({
  type,
  message,
}: {
  type: "success" | "error" | "warning" | "info";
  message: string;
}) {
  const { addToast } = useToast();

  return (
    <button
      onClick={() => addToast(message, type)}
      className={`${button.size.md} bg-surface-raised hover:bg-surface-hover ${radius.lg} border border-surface-border text-text-primary`}
    >
      Show {type} toast
    </button>
  );
}

export const Success: StoryObj = {
  render: () => <ToastDemo type="success" message="Operation completed successfully!" />,
};

export const Error: StoryObj = {
  render: () => <ToastDemo type="error" message="An error occurred. Please try again." />,
};

export const Warning: StoryObj = {
  render: () => <ToastDemo type="warning" message="This action cannot be undone." />,
};

export const Info: StoryObj = {
  render: () => <ToastDemo type="info" message="Network scan is in progress..." />,
};

function AllToastsDemo() {
  const { addToast } = useToast();

  return (
    <div className={`${layout.stack.default}`}>
      <button
        onClick={() => {
          addToast("Success message", "success", 3000);
          setTimeout(() => addToast("Error message", "error", 3000), 500);
          setTimeout(() => addToast("Warning message", "warning", 3000), 1000);
          setTimeout(() => addToast("Info message", "info", 3000), 1500);
        }}
        className={`${button.size.md} bg-status-info hover:bg-status-info/80 ${radius.lg} text-text-inverse`}
      >
        Show all toast types
      </button>
    </div>
  );
}

export const AllTypes: StoryObj = {
  render: () => <AllToastsDemo />,
  parameters: {
    docs: {
      description: {
        story: "Click to see all toast types in sequence.",
      },
    },
  },
};
