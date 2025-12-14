import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      fontSize: {
        "2xs": ["0.6875rem", { lineHeight: "1rem" }], // 11px - replaces text-[11px]
      },
      maxWidth: {
        "8xl": "88rem", // 1408px - wider container for dashboard
      },
      maxHeight: {
        modal: "85vh", // replaces max-h-[85vh]
      },
      fontFamily: {
        display: [
          '"Inter"',
          "system-ui",
          "-apple-system",
          "Segoe UI",
          "sans-serif",
        ],
        body: [
          '"Inter"',
          "system-ui",
          "-apple-system",
          "Segoe UI",
          "sans-serif",
        ],
        mono: [
          '"JetBrains Mono"',
          "ui-monospace",
          "SFMono-Regular",
          "monospace",
        ],
      },
      colors: {
        // WiFi Vigilante color scheme
        brand: {
          primary: "var(--color-brand-primary)",
          accent: "var(--color-brand-accent)",
        },
        surface: {
          base: "var(--color-surface-base)",
          raised: "var(--color-surface-raised)",
          border: "var(--color-surface-border)",
          hover: "var(--color-surface-hover)",
        },
        text: {
          primary: "var(--color-text-primary)",
          secondary: "var(--color-text-secondary)",
          muted: "var(--color-text-muted)",
          accent: "var(--color-text-accent)",
          inverse: "var(--color-text-inverse)",
        },
        status: {
          success: "var(--color-status-success)",
          warning: "var(--color-status-warning)",
          error: "var(--color-status-error)",
          info: "var(--color-status-info)",
        },
      },
    },
  },
  plugins: [],
};

export default config;
