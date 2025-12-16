import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { ErrorBoundary } from "./ErrorBoundary";
import { spacing, button, radius, section, layout, icon } from "../../styles/theme";

/**
 * ErrorBoundary catches JavaScript errors in child component trees,
 * logs them, and displays a fallback UI instead of crashing the entire application.
 *
 * This is a React class component implementing the Error Boundary pattern
 * with optional custom fallback UI and retry functionality.
 */
const meta: Meta<typeof ErrorBoundary> = {
  title: "UI/ErrorBoundary",
  component: ErrorBoundary,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Error Boundary component for catching and handling runtime errors in React component trees with graceful fallback UI.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    children: {
      description: "Child components to be protected by the error boundary",
    },
    fallback: {
      description: "Optional custom fallback UI to display when error occurs",
    },
  },
  decorators: [
    (Story) => (
      <div className={`${spacing.pad.xl} w-full max-w-2xl`}>
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

// ============================================================================
// Demo Components that can throw errors
// ============================================================================

/**
 * Component that immediately throws an error on render
 */
function BrokenComponent() {
  throw new Error("This component intentionally throws an error!");
  return <div>You should not see this</div>;
}

/**
 * Component that throws after a button click
 */
function ClickToBrokenComponent() {
  const [shouldThrow, setShouldThrow] = useState(false);

  if (shouldThrow) {
    throw new Error("Error triggered by user action!");
  }

  return (
    <div
      className={`${spacing.pad.default} bg-surface-raised border border-surface-border ${radius.lg}`}
    >
      <h3 className={`heading-4 ${spacing.margin.bottom.heading}`}>Interactive Error Demo</h3>
      <p className={`body-small text-text-secondary ${spacing.margin.bottom.content}`}>
        Click the button below to trigger an error. The Error Boundary will catch it and display the
        fallback UI.
      </p>
      <button
        onClick={() => setShouldThrow(true)}
        className={`${button.size.md} bg-status-error text-text-inverse ${radius.lg} hover:bg-status-error/90 transition-colors`}
      >
        Trigger Error
      </button>
    </div>
  );
}

/**
 * Component that throws an error with stack trace details
 */
function ComponentWithStackTrace() {
  function deepNestedFunction() {
    function evenDeeperFunction() {
      throw new Error("Detailed error message with stack trace from deep nested function call");
    }
    evenDeeperFunction();
  }
  deepNestedFunction();
  return null;
}

/**
 * Normal working component - no errors
 */
function WorkingComponent() {
  return (
    <div
      className={`${spacing.pad.default} bg-status-success/10 border border-status-success/20 ${radius.lg}`}
    >
      <div className={`${layout.inline.comfortable}`}>
        <svg
          className={
            `${icon.size.md} text-status-success shrink-0 ${spacing.micro.mt}` /* spacing.micro.mt for icon alignment */
          }
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clipRule="evenodd"
          />
        </svg>
        <div>
          <h3 className={`font-semibold text-status-success ${spacing.margin.bottom.tight}`}>
            Component Working Correctly
          </h3>
          <p className="body-small text-text-secondary">
            This component is functioning normally with no errors. The Error Boundary is monitoring
            it but not displaying any fallback UI.
          </p>
        </div>
      </div>
    </div>
  );
}

/**
 * Component simulating a network error
 */
function NetworkErrorComponent(): React.ReactElement {
  throw new Error("Network request failed: Unable to fetch data from API endpoint");
}

/**
 * Component simulating a type error
 */
function TypeErrorComponent() {
  const data = null as unknown as { property: { nested: { value: string } } };
  return <div>{data.property.nested.value}</div>;
}

// ============================================================================
// Stories
// ============================================================================

/**
 * Normal state - no errors, children render normally
 */
export const NoError: Story = {
  render: () => (
    <ErrorBoundary>
      <WorkingComponent />
    </ErrorBoundary>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Normal operation with no errors. Error Boundary is active but transparent - child component renders normally.",
      },
    },
  },
};

/**
 * Error caught - displays default error UI
 */
export const ErrorCaught: Story = {
  render: () => (
    <ErrorBoundary>
      <BrokenComponent />
    </ErrorBoundary>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Error Boundary catches the error and displays default fallback UI with error message and retry button.",
      },
    },
  },
};

/**
 * Error with detailed message
 */
export const ErrorWithDetailedMessage: Story = {
  render: () => (
    <ErrorBoundary>
      <NetworkErrorComponent />
    </ErrorBoundary>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Error with detailed message showing network failure. Demonstrates error message display in the fallback UI.",
      },
    },
  },
};

/**
 * Error with stack trace (check browser console)
 */
export const ErrorWithStackTrace: Story = {
  render: () => (
    <ErrorBoundary>
      <ComponentWithStackTrace />
    </ErrorBoundary>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Error from deeply nested function calls. Open browser console to see full stack trace logged by componentDidCatch.",
      },
    },
  },
};

/**
 * Interactive error trigger with retry functionality
 */
export const InteractiveError: Story = {
  render: () => {
    return (
      <ErrorBoundary>
        <ClickToBrokenComponent />
      </ErrorBoundary>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          "Click 'Trigger Error' to throw an error. Then click 'Try again' in the error UI to reset the Error Boundary and retry rendering.",
      },
    },
  },
};

/**
 * Error with custom fallback UI
 */
export const CustomFallback: Story = {
  render: () => (
    <ErrorBoundary
      fallback={
        <div
          className={`${spacing.pad.lg} bg-linear-to-br from-status-error/20 to-status-error/5 border-2 border-status-error ${radius.xl}`}
        >
          <div className={`${layout.inline.spacious} items-start`}>
            <div
              className={`w-12 h-12 ${radius.full} bg-status-error/20 ${layout.flex.center} shrink-0`}
            >
              <svg
                className="w-6 h-6 text-status-error"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <div className="flex-1">
              <h3 className={`heading-3 text-status-error ${spacing.margin.bottom.inline}`}>
                Custom Error Fallback UI
              </h3>
              <p className={`body text-text-secondary ${spacing.margin.bottom.content}`}>
                This is a custom fallback component passed as a prop to the Error Boundary. You can
                design any UI you want to display when errors occur.
              </p>
              <div className={`${layout.inline.comfortable}`}>
                <button
                  className={`${button.size.md} bg-status-error text-text-inverse ${radius.lg} hover:bg-status-error/90 transition-colors`}
                >
                  Report Error
                </button>
                <button
                  className={`${button.size.md} bg-surface-raised border border-surface-border text-text-primary ${radius.lg} hover:bg-surface-hover transition-colors`}
                >
                  Go to Dashboard
                </button>
              </div>
            </div>
          </div>
        </div>
      }
    >
      <BrokenComponent />
    </ErrorBoundary>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Custom fallback UI passed as a prop. Demonstrates how to provide your own error display instead of the default alert-style UI.",
      },
    },
  },
};

/**
 * Multiple Error Boundaries protecting different sections
 */
export const MultipleErrorBoundaries: Story = {
  render: () => (
    <div className={`${section.spacing.default} w-full`}>
      <div>
        <h3 className={`heading-4 ${spacing.margin.bottom.inline} text-text-primary`}>
          Section 1 - Working
        </h3>
        <ErrorBoundary>
          <WorkingComponent />
        </ErrorBoundary>
      </div>

      <div>
        <h3 className={`heading-4 ${spacing.margin.bottom.inline} text-text-primary`}>
          Section 2 - Error
        </h3>
        <ErrorBoundary>
          <BrokenComponent />
        </ErrorBoundary>
      </div>

      <div>
        <h3 className={`heading-4 ${spacing.margin.bottom.inline} text-text-primary`}>
          Section 3 - Working
        </h3>
        <ErrorBoundary>
          <WorkingComponent />
        </ErrorBoundary>
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Multiple Error Boundaries isolate errors to specific sections. Only Section 2 shows error UI while others render normally.",
      },
    },
  },
};

/**
 * Error Boundary with retry functionality demonstration
 */
export const WithRetry: Story = {
  render: () => {
    const [attemptCount, setAttemptCount] = useState(0);

    function UnstableComponent() {
      if (attemptCount < 2) {
        throw new Error(
          `Simulated error (attempt ${attemptCount + 1}/3). Click "Try again" to retry.`
        );
      }
      return (
        <div
          className={`${spacing.pad.default} bg-status-success/10 border border-status-success/20 ${radius.lg}`}
        >
          <div className={`${layout.inline.comfortable} items-start`}>
            <svg
              className={
                `${icon.size.md} text-status-success shrink-0 ${spacing.micro.mt}` /* spacing.micro.mt for icon alignment */
              }
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                clipRule="evenodd"
              />
            </svg>
            <div>
              <h3 className={`font-semibold text-status-success ${spacing.margin.bottom.tight}`}>
                Success After Retries!
              </h3>
              <p className="body-small text-text-secondary">
                The component successfully rendered after {attemptCount + 1} attempts. Error
                Boundary retry functionality allowed recovery without page reload.
              </p>
            </div>
          </div>
        </div>
      );
    }

    return (
      <div className={`${section.spacing.default}`}>
        <div
          className={`${spacing.pad.default} bg-surface-hover border border-surface-border ${radius.lg}`}
        >
          <p className="body-small text-text-secondary">
            This demo simulates a component that fails twice then succeeds. Click "Try again" when
            you see an error to retry rendering.
          </p>
          <p className={`caption text-text-muted ${spacing.margin.top.inline}`}>
            Attempts: {attemptCount + 1}/3
          </p>
        </div>
        <ErrorBoundary key={attemptCount}>
          <UnstableComponent />
        </ErrorBoundary>
        <button
          onClick={() => setAttemptCount((c) => c + 1)}
          className={`${button.size.md} bg-surface-raised border border-surface-border text-text-primary ${radius.lg} hover:bg-surface-hover transition-colors`}
        >
          Manual Retry (increment counter)
        </button>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          "Demonstrates retry functionality. Component fails on first two attempts, then succeeds on third. Shows how Error Boundary allows recovery.",
      },
    },
  },
};

/**
 * TypeError demonstration - null reference error
 */
export const TypeErrorDemo: Story = {
  render: () => (
    <ErrorBoundary>
      <TypeErrorComponent />
    </ErrorBoundary>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Catches TypeError from accessing properties on null/undefined. Common JavaScript error caught by Error Boundary.",
      },
    },
  },
};

/**
 * Nested Error Boundaries - granular error handling
 */
export const NestedErrorBoundaries: Story = {
  render: () => (
    <div className={`${section.spacing.default}`}>
      <ErrorBoundary
        fallback={
          <div
            className={`${spacing.pad.default} bg-status-error/10 border border-status-error ${radius.lg}`}
          >
            <p className="body-small text-status-error">
              Outer Error Boundary - This should not be visible
            </p>
          </div>
        }
      >
        <div
          className={`${spacing.pad.default} bg-surface-raised border border-surface-border ${radius.lg}`}
        >
          <h3 className={`heading-4 ${spacing.margin.bottom.heading}`}>Outer Boundary</h3>
          <p className={`body-small text-text-secondary ${spacing.margin.bottom.content}`}>
            This outer boundary protects the entire section. The inner boundary handles the specific
            error.
          </p>

          <ErrorBoundary
            fallback={
              <div
                className={`${spacing.pad.sm} bg-status-warning/10 border border-status-warning ${radius.lg}`}
              >
                <p className="body-small text-status-warning">
                  Inner Error Boundary - Caught error in nested component
                </p>
              </div>
            }
          >
            <BrokenComponent />
          </ErrorBoundary>

          <p className={`body-small text-text-secondary ${spacing.margin.top.content}`}>
            Content after inner error boundary still renders because error was caught by inner
            boundary.
          </p>
        </div>
      </ErrorBoundary>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Nested Error Boundaries provide granular error isolation. Inner boundary catches the error, outer boundary and sibling content continue rendering.",
      },
    },
  },
};

/**
 * Real-world example - Card with Error Boundary
 */
export const RealWorldCardExample: Story = {
  render: () => {
    function CardContent({ shouldError = false }: { shouldError?: boolean }) {
      if (shouldError) {
        throw new Error("Failed to load card data from API");
      }

      return (
        <div className="stack-sm">
          <div className="flex justify-between">
            <span className="caption text-text-muted">Speed</span>
            <span className="body-small font-medium text-text-primary">1000 Mbps</span>
          </div>
          <div className="flex justify-between">
            <span className="caption text-text-muted">Duplex</span>
            <span className="body-small font-medium text-text-primary">Full</span>
          </div>
          <div className="flex justify-between">
            <span className="caption text-text-muted">MTU</span>
            <span className="body-small font-medium text-text-primary">1500</span>
          </div>
        </div>
      );
    }

    return (
      <div className={`grid md:grid-cols-2 ${spacing.gap.comfortable} w-full`}>
        <div
          className={`${spacing.pad.default} bg-surface-raised border border-surface-border ${radius.lg}`}
        >
          <h3 className={`heading-4 ${spacing.margin.bottom.heading} text-text-primary`}>
            Link Status - Working
          </h3>
          <ErrorBoundary>
            <CardContent shouldError={false} />
          </ErrorBoundary>
        </div>

        <div
          className={`${spacing.pad.default} bg-surface-raised border border-surface-border ${radius.lg}`}
        >
          <h3 className={`heading-4 ${spacing.margin.bottom.heading} text-text-primary`}>
            Link Status - Error
          </h3>
          <ErrorBoundary>
            <CardContent shouldError={true} />
          </ErrorBoundary>
        </div>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          "Real-world usage showing Error Boundary protecting individual dashboard cards. Left card works, right card shows error - demonstrates isolation.",
      },
    },
  },
};
