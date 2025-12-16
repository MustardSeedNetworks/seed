import type { Meta, StoryObj } from "@storybook/react-vite";
import { SetupWizard } from "./SetupWizard";
import { useState } from "react";
import { spacing, radius } from "../../styles/theme";

// No-op function for story event handlers
const noop = () => {};

/**
 * SetupWizard guides users through initial system setup.
 *
 * Features:
 * - Admin password creation with validation
 * - Generated password suggestion option
 * - Custom password entry mode
 * - Password visibility toggle
 * - Password confirmation requirement
 * - Automatic login after setup
 * - Error handling and feedback
 *
 * This story demonstrates the complete setup flow and various states.
 */
const meta: Meta<typeof SetupWizard> = {
  title: "Setup/SetupWizard",
  component: SetupWizard,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof SetupWizard>;

/**
 * Initial setup wizard state with custom password mode.
 * User will create their own password.
 */
export const CustomPasswordMode: Story = {
  args: {
    onComplete: () => {
      // Handle setup completion
    },
    onLogin: async (_username: string, _password: string) => {
      return true;
    },
  },
};

/**
 * Setup wizard with suggested secure password option.
 * Shows generated password that user can accept or customize.
 */
export const WithSuggestedPassword: Story = {
  args: {
    onComplete: () => {
      // Handle setup completion
    },
    onLogin: async (_username: string, _password: string) => {
      return true;
    },
    suggestedPassword: "Xk9mP#2vL@q7Tn4w",
  },
};

/**
 * Generated password mode selected.
 * Shows the secure password with copy button and warning to save it.
 */
export const GeneratedPasswordSelected: Story = {
  render: () => {
    return (
      <div className="relative min-h-screen">
        <SetupWizard
          onComplete={() => {
            // Handle setup completion
          }}
          onLogin={async () => true}
          suggestedPassword="Xk9mP#2vL@q7Tn4w"
        />
      </div>
    );
  },
};

/**
 * Validation error: password too short.
 * Shows error message when password is less than 8 characters.
 */
export const ValidationErrorTooShort: Story = {
  render: () => {
    const WizardWithError = () => {
      return (
        <SetupWizard
          onComplete={() => {
            // Handle setup completion
          }}
          onLogin={async () => true}
        />
      );
    };

    return <WizardWithError />;
  },
};

/**
 * Validation error: passwords do not match.
 * Shows error when password confirmation doesn't match initial password.
 */
export const ValidationErrorMismatch: Story = {
  render: () => {
    const WizardWithError = () => {
      return (
        <SetupWizard
          onComplete={() => {
            // Handle setup completion
          }}
          onLogin={async () => true}
        />
      );
    };

    return <WizardWithError />;
  },
  parameters: {
    docs: {
      description: {
        story:
          "Enter different values in password and confirm password fields to see the mismatch error.",
      },
    },
  },
};

/**
 * Setup submission in progress.
 * Shows loading state while password is being set on server.
 */
export const SubmittingSetup: Story = {
  render: () => {
    const WizardSubmitting = () => {
      return (
        <SetupWizard
          onComplete={() => {
            // Handle setup completion
          }}
          onLogin={async () => {
            return new Promise((resolve) => {
              setTimeout(() => resolve(true), 3000);
            });
          }}
        />
      );
    };

    return <WizardSubmitting />;
  },
};

/**
 * Network error during setup.
 * Shows error message when API request fails.
 */
export const NetworkError: Story = {
  args: {
    onComplete: noop,
    onLogin: async () => {
      throw new Error("Network error. Please try again.");
    },
  },
};

/**
 * Setup complete but login failed.
 * Shows scenario where password was set but automatic login didn't work.
 */
export const SetupCompleteLoginFailed: Story = {
  args: {
    onComplete: noop,
    onLogin: async () => {
      return false;
    },
  },
};

/**
 * Password visibility toggled on.
 * Shows passwords in plain text when visibility toggle is enabled.
 */
export const PasswordVisible: Story = {
  render: () => {
    const WizardPasswordVisible = () => {
      return (
        <SetupWizard
          onComplete={() => {
            // Handle setup completion
          }}
          onLogin={async () => true}
        />
      );
    };

    return <WizardPasswordVisible />;
  },
  parameters: {
    docs: {
      description: {
        story: "Click the eye icon to toggle password visibility.",
      },
    },
  },
};

/**
 * Mobile viewport responsive layout.
 * Shows how the wizard adapts to smaller screens.
 */
export const MobileViewport: Story = {
  args: {
    onComplete: () => {
      // Handle setup completion
    },
    onLogin: async () => true,
    suggestedPassword: "Xk9mP#2vL@q7Tn4w",
  },
  parameters: {
    viewport: {
      defaultViewport: "mobile1",
    },
  },
};

/**
 * Tablet viewport responsive layout.
 * Shows how the wizard displays on tablet-sized screens.
 */
export const TabletViewport: Story = {
  args: {
    onComplete: () => {
      // Handle setup completion
    },
    onLogin: async () => true,
    suggestedPassword: "Xk9mP#2vL@q7Tn4w",
  },
  parameters: {
    viewport: {
      defaultViewport: "tablet",
    },
  },
};

/**
 * Interactive example: complete setup flow.
 * Demonstrates full user journey from password entry to completion.
 */
export const InteractiveSetupFlow: Story = {
  render: () => {
    const [setupComplete, setSetupComplete] = useState(false);

    if (setupComplete) {
      return (
        <div className="min-h-screen bg-surface-base flex items-center justify-center">
          <div className="text-center">
            <div
              className={`w-16 h-16 mx-auto ${spacing.margin.bottom.content} ${radius.full} bg-status-success/20 flex items-center justify-center`}
            >
              <svg
                className="w-8 h-8 text-status-success"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
            <h2 className={`heading-2 ${spacing.margin.bottom.inline}`}>Setup Complete!</h2>
            <p className="body-small text-text-muted">You are now logged in.</p>
          </div>
        </div>
      );
    }

    return (
      <SetupWizard
        onComplete={() => setSetupComplete(true)}
        onLogin={async (_username, _password) => {
          // Simulate API delay
          await new Promise((resolve) => setTimeout(resolve, 1000));
          return true;
        }}
        suggestedPassword="Xk9mP#2vL@q7Tn4w"
      />
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Fill out the form and submit to see the complete setup flow with success state.",
      },
    },
  },
};
