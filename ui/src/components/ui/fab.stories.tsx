import type { Meta, StoryFn, StoryObj } from "@storybook/react-vite";
import type React from "react";
import { useEffect } from "react";
import { cn, spacing } from "../../styles/theme";
import { Fab } from "./Fab";

/**
 * The Floating Action Button (FAB) provides quick access to running all diagnostic tests.
 * It's positioned in the bottom-right corner and shows a loading spinner during execution.
 */
const meta: Meta<typeof Fab> = {
  title: "UI/FAB",
  component: Fab,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="relative h-96 bg-surface-base">
        <div class={cn(spacing.pad.default)}>
          <p class="text-text-secondary">
            The FAB is fixed in the bottom-right corner. Click to trigger tests.
          </p>
        </div>
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof Fab>;

export const Default: Story = {};

export const WithSimulatedTest: Story = {
  render: () => {
    // Simulate test completion after 3 seconds
    useEffect(() => {
      const handleRunTests = () => {
        setTimeout(() => {
          window.dispatchEvent(new CustomEvent("testsComplete"));
        }, 3000);
      };

      window.addEventListener("runAllTests", handleRunTests);
      return () => window.removeEventListener("runAllTests", handleRunTests);
    }, []);

    return <Fab />;
  },
  parameters: {
    docs: {
      description: {
        story: "Click the FAB to see it enter loading state. Tests complete after 3 seconds.",
      },
    },
  },
};

export const CustomPosition: Story = {
  args: {
    className: "bottom-20 right-20",
  },
  parameters: {
    docs: {
      description: {
        story: "The FAB position can be customized via className prop.",
      },
    },
  },
};
