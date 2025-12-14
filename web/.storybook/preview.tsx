import React from "react";
import type { Preview } from "@storybook/react-vite";
import "../src/index.css";

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    backgrounds: {
      default: "dark",
      values: [
        { name: "dark", value: "var(--color-surface-base, #0f172a)" },
        { name: "light", value: "var(--color-surface-base-light, #f8fafc)" },
      ],
    },
    layout: "centered",
  },
  decorators: [
    (Story) => (
      <div className="dark p-4">
        <Story />
      </div>
    ),
  ],
};

export default preview;
