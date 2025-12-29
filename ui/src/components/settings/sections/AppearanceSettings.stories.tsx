/**
 * AppearanceSettings Storybook Stories
 *
 * Demonstrates the appearance/theme settings component allowing users to
 * customize the visual theme of the application.
 *
 * Variants:
 * - Light theme selected
 * - Dark theme selected
 * - System theme selected (follows OS preference)
 * - Interactive theme toggle
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { cn, spacing } from "../../../styles/theme";
import { AppearanceSettings } from "./AppearanceSettings";

const meta: Meta<typeof AppearanceSettings> = {
  title: "Settings/AppearanceSettings",
  component: AppearanceSettings,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Theme selection settings allowing users to choose between light, dark, or system-preferred themes. Includes a quick toggle button for easy switching.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    theme: {
      control: "select",
      options: ["light", "dark", "system"],
      description: "Current theme selection",
    },
    isDark: {
      control: "boolean",
      description: "Whether dark mode is currently active",
    },
  },
  decorators: [
    (Story) => (
      <div className="w-[400px]">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Light theme selected
 */
export const LightTheme: Story = {
  args: {
    theme: "light",
    isDark: false,
    setTheme: () => {
      // intentionally empty - story placeholder callback
    },
  },
};

/**
 * Dark theme selected
 */
export const DarkTheme: Story = {
  args: {
    theme: "dark",
    isDark: true,
    setTheme: () => {
      // intentionally empty - story placeholder callback
    },
  },
};

/**
 * System theme selected (follows OS preference)
 */
export const SystemTheme: Story = {
  args: {
    theme: "system",
    isDark: false,
    setTheme: () => {
      // intentionally empty - story placeholder callback
    },
  },
};

/**
 * System theme with dark mode active
 */
export const SystemThemeDark: Story = {
  args: {
    theme: "system",
    isDark: true,
    setTheme: () => {
      // intentionally empty - story placeholder callback
    },
  },
};

/**
 * Interactive theme selector - fully functional
 */
export const Interactive: Story = {
  render: function InteractiveStory() {
    const [theme, setTheme] = useState<"light" | "dark" | "system">("light");
    const [isDark, setIsDark] = useState(false);

    const handleSetTheme = (newTheme: "light" | "dark" | "system") => {
      setTheme(newTheme);
      if (newTheme === "light") {
        setIsDark(false);
      } else if (newTheme === "dark") {
        setIsDark(true);
      }
      // For "system", we'd normally detect OS preference
    };

    return <AppearanceSettings theme={theme} setTheme={handleSetTheme} isDark={isDark} />;
  },
};

/**
 * Multiple appearance sections showing different states
 */
// No-op function for story event handlers
const noopSetTheme = () => {
  // intentionally empty - story placeholder callback
};

export const Comparison: Story = {
  render: () => (
    <div className="stack-lg">
      <div>
        <p className={cn("caption text-text-muted", spacing.margin.bottom.inline)}>Light Theme</p>
        <AppearanceSettings theme="light" setTheme={noopSetTheme} isDark={false} />
      </div>
      <div>
        <p className={cn("caption text-text-muted", spacing.margin.bottom.inline)}>Dark Theme</p>
        <AppearanceSettings theme="dark" setTheme={noopSetTheme} isDark={true} />
      </div>
      <div>
        <p className={cn("caption text-text-muted", spacing.margin.bottom.inline)}>System Theme</p>
        <AppearanceSettings theme="system" setTheme={noopSetTheme} isDark={false} />
      </div>
    </div>
  ),
};
