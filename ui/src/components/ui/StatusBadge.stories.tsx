import type { Meta, StoryObj } from "@storybook/react-vite";
import { cn, layout, spacing } from "../../styles/theme";
import { StatusBadge } from "./StatusBadge";

const meta: Meta<typeof StatusBadge> = {
  title: "UI/StatusBadge",
  component: StatusBadge,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Displays system status with visual indicators (icon or dot) using consistent color-coding.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    status: {
      control: "select",
      options: ["success", "warning", "error", "unknown", "loading"],
      description: "Status type to display",
    },
    variant: {
      control: "radio",
      options: ["icon", "dot"],
      description: "Visual variant",
    },
    size: {
      control: "radio",
      options: ["sm", "md"],
      description: "Size of the badge",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Icon variants
export const SuccessIcon: Story = {
  args: {
    status: "success",
    variant: "icon",
    size: "md",
  },
};

export const WarningIcon: Story = {
  args: {
    status: "warning",
    variant: "icon",
    size: "md",
  },
};

export const ErrorIcon: Story = {
  args: {
    status: "error",
    variant: "icon",
    size: "md",
  },
};

export const LoadingIcon: Story = {
  args: {
    status: "loading",
    variant: "icon",
    size: "md",
  },
};

export const UnknownIcon: Story = {
  args: {
    status: "unknown",
    variant: "icon",
    size: "md",
  },
};

// Dot variants
export const SuccessDot: Story = {
  args: {
    status: "success",
    variant: "dot",
    size: "md",
  },
};

export const WarningDot: Story = {
  args: {
    status: "warning",
    variant: "dot",
    size: "md",
  },
};

export const ErrorDot: Story = {
  args: {
    status: "error",
    variant: "dot",
    size: "md",
  },
};

// Size comparison
export const SmallSize: Story = {
  args: {
    status: "success",
    variant: "icon",
    size: "sm",
  },
};

export const MediumSize: Story = {
  args: {
    status: "success",
    variant: "icon",
    size: "md",
  },
};

// All statuses gallery
export const AllStatuses: Story = {
  render: () => (
    <div className={layout.stack.spacious}>
      <div>
        <h3
          className={cn("body-small font-semibold text-text-muted", spacing.margin.bottom.inline)}
        >
          Icon Variant (Medium)
        </h3>
        <div className={layout.inline.spacious}>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="success" variant="icon" size="md" />
            <span className="caption text-text-muted">Success</span>
          </div>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="warning" variant="icon" size="md" />
            <span className="caption text-text-muted">Warning</span>
          </div>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="error" variant="icon" size="md" />
            <span className="caption text-text-muted">Error</span>
          </div>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="loading" variant="icon" size="md" />
            <span className="caption text-text-muted">Loading</span>
          </div>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="unknown" variant="icon" size="md" />
            <span className="caption text-text-muted">Unknown</span>
          </div>
        </div>
      </div>
      <div>
        <h3
          className={cn("body-small font-semibold text-text-muted", spacing.margin.bottom.inline)}
        >
          Dot Variant
        </h3>
        <div className={layout.inline.spacious}>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="success" variant="dot" size="md" />
            <span className="caption text-text-muted">Success</span>
          </div>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="warning" variant="dot" size="md" />
            <span className="caption text-text-muted">Warning</span>
          </div>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="error" variant="dot" size="md" />
            <span className="caption text-text-muted">Error</span>
          </div>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="loading" variant="dot" size="md" />
            <span className="caption text-text-muted">Loading</span>
          </div>
          <div className={cn(layout.stack.tight, "items-center")}>
            <StatusBadge status="unknown" variant="dot" size="md" />
            <span className="caption text-text-muted">Unknown</span>
          </div>
        </div>
      </div>
    </div>
  ),
};
