import type { Meta, StoryObj } from "@storybook/react-vite";
import type React from "react";
import { cn, section as sectionStyles, spacing } from "../../styles/theme";
import { WIFI_SURVEY_HELP } from "./WiFiSurveyHelp";

/**
 * WiFiSurveyHelp provides comprehensive help documentation for the WiFi Site Survey feature.
 *
 * This module exports help content as structured data that can be rendered in help modals
 * or documentation components. It covers:
 * - Survey overview and use cases
 * - Survey modes (Passive, Active, Throughput)
 * - Creating and conducting surveys
 * - Viewing results and heatmaps
 * - Troubleshooting common issues
 * - Best practices for WiFi surveying
 */
const meta: Meta = {
  title: "Help/WiFiSurveyHelp",
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Structured help content for WiFi Site Survey feature, providing user guidance on survey creation, modes, and interpretation.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj;

/**
 * All WiFi Survey help sections rendered as collapsible panels.
 */
export const AllSections: Story = {
  render: () => (
    <div class={cn("max-w-3xl mx-auto", spacing.pad.default, sectionStyles.spacing.comfortable)}>
      <h1 class={cn("heading-1 text-text-primary", spacing.margin.bottom.section)}>
        WiFi Survey Help
      </h1>
      {WIFI_SURVEY_HELP.map((section) => (
        <helpSection key={section.title} title={section.title}>
          {section.items.map((item) => (
            <helpItem key={item.question} question={item.question} answer={item.answer} />
          ))}
        </helpSection>
      ))}
    </div>
  ),
};

/**
 * Overview section explaining what WiFi Survey is and when to use it.
 */
export const Overview: Story = {
  render: () => {
    const overviewSection = WIFI_SURVEY_HELP.find((s) => s.title === "Overview");
    if (!overviewSection) {
      return <div>Section not found</div>;
    }

    return (
      <div class={cn("max-w-3xl mx-auto", spacing.pad.default)}>
        <helpSection title={overviewSection.title}>
          {overviewSection.items.map((item) => (
            <helpItem key={item.question} question={item.question} answer={item.answer} />
          ))}
        </helpSection>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Overview section explaining WiFi Survey purpose, use cases, and requirements.",
      },
    },
  },
};

/**
 * Survey Modes section explaining Passive, Active, and Throughput modes.
 */
export const SurveyModes: Story = {
  render: () => {
    const modesSection = WIFI_SURVEY_HELP.find((s) => s.title === "Survey Modes");
    if (!modesSection) {
      return <div>Section not found</div>;
    }

    return (
      <div class={cn("max-w-3xl mx-auto", spacing.pad.default)}>
        <helpSection title={modesSection.title}>
          {modesSection.items.map((item) => (
            <helpItem key={item.question} question={item.question} answer={item.answer} />
          ))}
        </helpSection>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          "Detailed explanation of survey modes: Passive Scan, Active Monitoring, and Throughput Testing.",
      },
    },
  },
};

/**
 * Creating a Survey section with step-by-step instructions.
 */
export const CreatingSurvey: Story = {
  render: () => {
    const createSection = WIFI_SURVEY_HELP.find((s) => s.title === "Creating a Survey");
    if (!createSection) {
      return <div>Section not found</div>;
    }

    return (
      <div class={cn("max-w-3xl mx-auto", spacing.pad.default)}>
        <helpSection title={createSection.title}>
          {createSection.items.map((item) => (
            <helpItem key={item.question} question={item.question} answer={item.answer} />
          ))}
        </helpSection>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          "Step-by-step guide for creating surveys, uploading floor plans, and getting started.",
      },
    },
  },
};

/**
 * Conducting a Survey section with measurement best practices.
 */
export const ConductingSurvey: Story = {
  render: () => {
    const conductSection = WIFI_SURVEY_HELP.find((s) => s.title === "Conducting a Survey");
    if (!conductSection) {
      return <div>Section not found</div>;
    }

    return (
      <div class={cn("max-w-3xl mx-auto", spacing.pad.default)}>
        <helpSection title={conductSection.title}>
          {conductSection.items.map((item) => (
            <helpItem key={item.question} question={item.question} answer={item.answer} />
          ))}
        </helpSection>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Instructions for taking measurements, sample point density, and pausing surveys.",
      },
    },
  },
};

/**
 * Viewing Results section with heatmap interpretation guide.
 */
export const ViewingResults: Story = {
  render: () => {
    const resultsSection = WIFI_SURVEY_HELP.find((s) => s.title === "Viewing Results");
    if (!resultsSection) {
      return <div>Section not found</div>;
    }

    return (
      <div class={cn("max-w-3xl mx-auto", spacing.pad.default)}>
        <helpSection title={resultsSection.title}>
          {resultsSection.items.map((item) => (
            <helpItem key={item.question} question={item.question} answer={item.answer} />
          ))}
        </helpSection>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Guide for viewing survey results, interpreting heatmap colors, and exporting data.",
      },
    },
  },
};

/**
 * Troubleshooting section for common issues.
 */
export const Troubleshooting: Story = {
  render: () => {
    const troubleSection = WIFI_SURVEY_HELP.find((s) => s.title === "Troubleshooting");
    if (!troubleSection) {
      return <div>Section not found</div>;
    }

    return (
      <div class={cn("max-w-3xl mx-auto", spacing.pad.default)}>
        <helpSection title={troubleSection.title}>
          {troubleSection.items.map((item) => (
            <helpItem key={item.question} question={item.question} answer={item.answer} />
          ))}
        </helpSection>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          "Solutions for common issues like missing WiFi card, iperf3 errors, and upload failures.",
      },
    },
  },
};

/**
 * Best Practices section with workflow recommendations.
 */
export const BestPractices: Story = {
  render: () => {
    const bestSection = WIFI_SURVEY_HELP.find((s) => s.title === "Best Practices");
    if (!bestSection) {
      return <div>Section not found</div>;
    }

    return (
      <div class={cn("max-w-3xl mx-auto", spacing.pad.default)}>
        <helpSection title={bestSection.title}>
          {bestSection.items.map((item) => (
            <helpItem key={item.question} question={item.question} answer={item.answer} />
          ))}
        </helpSection>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Recommended workflow and best practices for comprehensive WiFi surveys.",
      },
    },
  },
};

/**
 * Signal strength color legend for heatmap interpretation.
 */
export const SignalStrengthLegend: Story = {
  render: () => (
    <div class={cn("max-w-xl mx-auto", spacing.pad.default)}>
      <h2 class={cn("heading-2 text-text-primary", spacing.margin.bottom.content)}>
        Signal Strength Heatmap Legend
      </h2>
      <div class="stack-sm">
        <signalLevel
          color="bg-status-success"
          range="-30 to -50 dBm"
          label="Excellent"
          description="Maximum signal strength, ideal for all applications"
        />
        <signalLevel
          color="bg-lime-500"
          range="-50 to -60 dBm"
          label="Very Good"
          description="Strong signal, excellent for streaming and VoIP"
        />
        <signalLevel
          color="bg-status-warning"
          range="-60 to -70 dBm"
          label="Good"
          description="Reliable connection for most activities"
        />
        <signalLevel
          color="bg-orange-500"
          range="-70 to -80 dBm"
          label="Fair"
          description="Adequate for basic browsing, may have issues"
        />
        <signalLevel
          color="bg-status-error"
          range="Below -80 dBm"
          label="Weak/Dead Zone"
          description="Unstable connection, frequent dropouts"
        />
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Visual legend for interpreting signal strength heatmap colors.",
      },
    },
  },
};

/**
 * Throughput color legend for performance heatmap interpretation.
 */
export const ThroughputLegend: Story = {
  render: () => (
    <div class={cn("max-w-xl mx-auto", spacing.pad.default)}>
      <h2 class={cn("heading-2 text-text-primary", spacing.margin.bottom.content)}>
        Throughput Heatmap Legend
      </h2>
      <div class="stack-sm">
        <signalLevel
          color="bg-status-success"
          range="80-100%+ of expected"
          label="Excellent"
          description="Full speed, meeting or exceeding expectations"
        />
        <signalLevel
          color="bg-lime-500"
          range="60-80% of expected"
          label="Good"
          description="Strong performance for most applications"
        />
        <signalLevel
          color="bg-status-warning"
          range="40-60% of expected"
          label="Fair"
          description="Usable but may impact some activities"
        />
        <signalLevel
          color="bg-orange-500"
          range="20-40% of expected"
          label="Poor"
          description="Significantly degraded performance"
        />
        <signalLevel
          color="bg-status-error"
          range="Below 20% of expected"
          label="Critical"
          description="Severely limited, consider AP relocation"
        />
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Visual legend for interpreting throughput performance heatmap colors.",
      },
    },
  },
};

/**
 * Quick reference card for survey modes comparison.
 */
export const ModesComparison: Story = {
  render: () => (
    <div class={cn("max-w-4xl mx-auto", spacing.pad.default)}>
      <h2 class={cn("heading-2 text-text-primary", spacing.margin.bottom.content)}>
        Survey Modes Comparison
      </h2>
      <div class={cn("grid md:grid-cols-3", spacing.gap.comfortable)}>
        <modeCard
          title="Passive Scan"
          icon="📡"
          description="Scans all visible networks"
          pros={["See all nearby APs", "Detect interference", "No connection required"]}
          cons={["Doesn't test actual speed", "No roaming detection"]}
          bestFor="Initial site assessment"
        />
        <modeCard
          title="Active Monitoring"
          icon="📶"
          description="Monitors current connection"
          pros={["Real-time signal tracking", "Roaming detection", "Data rate monitoring"]}
          cons={["Single network only", "Requires connection"]}
          bestFor="Coverage validation"
        />
        <modeCard
          title="Throughput Testing"
          icon="⚡"
          description="Measures actual speeds"
          pros={["True performance data", "Latency & jitter metrics", "Most comprehensive"]}
          cons={["Requires iperf3 server", "Slower to collect"]}
          bestFor="Performance validation"
        />
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Side-by-side comparison of survey modes with pros, cons, and use cases.",
      },
    },
  },
};

// Helper Components

function _helpSection({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}): React.JSX.Element {
  return (
    <div class="bg-surface-raised border border-surface-border rounded-lg overflow-hidden">
      <h2
        class={cn(
          "heading-3 text-text-primary bg-surface-base border-b border-surface-border",
          spacing.pad.default,
        )}
      >
        {title}
      </h2>
      <div class="divide-y divide-surface-border">{children}</div>
    </div>
  );
}

function _helpItem({ question, answer }: { question: string; answer: string }): React.JSX.Element {
  return (
    <div class={spacing.pad.default}>
      <h3 class={cn("body font-semibold text-text-primary", spacing.margin.bottom.inline)}>
        {question}
      </h3>
      <div class="body-small text-text-secondary whitespace-pre-line">{answer}</div>
    </div>
  );
}

function _signalLevel({
  color,
  range,
  label,
  description,
}: {
  color: string;
  range: string;
  label: string;
  description: string;
}): React.JSX.Element {
  return (
    <div
      class={cn(
        "flex items-center bg-surface-raised border border-surface-border rounded-lg",
        spacing.gap.default,
        spacing.pad.sm,
      )}
    >
      <div class={cn("w-8 h-8 rounded", color)} />
      <div class="flex-1">
        <div class={cn("flex items-baseline", spacing.gap.compact)}>
          <span class="body font-semibold text-text-primary">{label}</span>
          <span class="body-small text-text-muted">({range})</span>
        </div>
        <p class="caption text-text-secondary">{description}</p>
      </div>
    </div>
  );
}

function _modeCard({
  title,
  icon,
  description,
  pros,
  cons,
  bestFor,
}: {
  title: string;
  icon: string;
  description: string;
  pros: string[];
  cons: string[];
  bestFor: string;
}): React.JSX.Element {
  return (
    <div
      class={cn("bg-surface-raised border border-surface-border rounded-lg", spacing.pad.default)}
    >
      <div class={cn("text-3xl", spacing.margin.bottom.inline)}>{icon}</div>
      <h3 class={cn("heading-4 text-text-primary", spacing.margin.bottom.tight)}>{title}</h3>
      <p class={cn("body-small text-text-muted", spacing.margin.bottom.content)}>{description}</p>

      <div class="stack-sm">
        <div>
          <h4 class={cn("caption font-semibold text-status-success", spacing.margin.bottom.tight)}>
            Pros
          </h4>
          <ul class="list-disc list-inside caption text-text-secondary">
            {pros.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
        </div>
        <div>
          <h4 class={cn("caption font-semibold text-status-warning", spacing.margin.bottom.tight)}>
            Cons
          </h4>
          <ul class="list-disc list-inside caption text-text-secondary">
            {cons.map((c) => (
              <li key={c}>{c}</li>
            ))}
          </ul>
        </div>
        <div class={cn("border-t border-surface-border", spacing.padding.bottom.inline)}>
          <span class="caption text-text-muted">Best for: </span>
          <span class="caption text-text-primary font-medium">{bestFor}</span>
        </div>
      </div>
    </div>
  );
}
