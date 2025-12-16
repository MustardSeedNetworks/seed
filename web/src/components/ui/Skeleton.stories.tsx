import type { Meta, StoryObj } from "@storybook/react-vite";
import { Skeleton, CardSkeleton, TextSkeleton } from "./Skeleton";
import { spacing, layout } from "../../styles/theme";

/**
 * Skeleton components provide visual loading placeholders while content is being fetched.
 * They improve perceived performance by showing the expected layout structure.
 */
const meta: Meta<typeof Skeleton> = {
  title: "UI/Skeleton",
  component: Skeleton,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    variant: {
      control: "radio",
      options: ["text", "circular", "rectangular"],
    },
  },
};

export default meta;
type Story = StoryObj<typeof Skeleton>;

export const Text: Story = {
  args: {
    variant: "text",
    className: "h-4 w-48",
  },
};

export const Circular: Story = {
  args: {
    variant: "circular",
    className: "h-12 w-12",
  },
};

export const Rectangular: Story = {
  args: {
    variant: "rectangular",
    className: "h-24 w-48",
  },
};

export const TextLines: StoryObj<typeof TextSkeleton> = {
  render: () => <TextSkeleton lines={4} />,
  parameters: {
    docs: {
      description: {
        story: "Multiple text lines with the last line shorter, simulating a paragraph.",
      },
    },
  },
};

export const Card: StoryObj<typeof CardSkeleton> = {
  render: () => <CardSkeleton />,
  parameters: {
    docs: {
      description: {
        story: "Pre-configured card skeleton matching the dashboard card layout.",
      },
    },
  },
};

export const DashboardGrid: Story = {
  render: () => (
    <div className={`grid grid-cols-2 ${spacing.gap.comfortable} w-[600px]`}>
      <CardSkeleton />
      <CardSkeleton />
      <CardSkeleton />
      <CardSkeleton />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Grid of card skeletons matching the dashboard layout during initial load.",
      },
    },
  },
};

export const UserProfile: Story = {
  render: () => (
    <div className={`${layout.inline.spacious} ${spacing.pad.default} bg-surface-base rounded-lg`}>
      <Skeleton variant="circular" className="h-16 w-16" />
      <div className={`${layout.stack.default}`}>
        <Skeleton variant="text" className="h-5 w-32" />
        <Skeleton variant="text" className="h-4 w-48" />
        <Skeleton variant="text" className="h-3 w-24" />
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Example of a user profile loading state with avatar and text.",
      },
    },
  },
};

export const TableRows: Story = {
  render: () => (
    <div className={`w-[500px] bg-surface-base rounded-lg ${spacing.pad.default}`}>
      <div className={`${layout.flex.between} ${spacing.margin.bottom.content}`}>
        <Skeleton variant="text" className="h-5 w-20" />
        <Skeleton variant="text" className="h-5 w-24" />
        <Skeleton variant="text" className="h-5 w-16" />
      </div>
      {[1, 2, 3, 4].map((i) => (
        <div
          key={i}
          className={`${layout.flex.between} ${spacing.row.pyLg} border-t border-surface-border`}
        >
          {" "}
          {/* spacing.row.pyLg for table row spacing */}
          <Skeleton variant="text" className="h-4 w-24" />
          <Skeleton variant="text" className="h-4 w-32" />
          <Skeleton variant="text" className="h-4 w-16" />
        </div>
      ))}
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Table loading state with header and row skeletons.",
      },
    },
  },
};
