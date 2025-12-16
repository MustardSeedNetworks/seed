import type { Meta, StoryObj } from "@storybook/react-vite";
import { FAB } from "./FAB";
import { useEffect } from "react";
import { spacing } from "../../styles/theme";

/**
 * The Floating Action Button (FAB) provides quick access to running all diagnostic tests.
 * It's positioned in the bottom-right corner and shows a loading spinner during execution.
 */
const meta: Meta<typeof FAB> = {
  title: "UI/FAB",
  component: FAB,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <div className="relative h-96 bg-surface-base">
        <div className={`${spacing.pad.default}`}>
          <p className="text-text-secondary">
            The FAB is fixed in the bottom-right corner. Click to trigger tests.
          </p>
        </div>
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof FAB>;

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

    return <FAB />;
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
