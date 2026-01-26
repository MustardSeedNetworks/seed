import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { ErrorBoundary } from './ErrorBoundary';

const meta = {
  title: 'App/ErrorBoundary',
  component: ErrorBoundary,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof ErrorBoundary>;

export default meta;

type Story = StoryObj<typeof meta>;

function Crash(): React.JSX.Element {
  throw new Error('Storybook induced crash');
}

export const Default: Story = {
  render: () => (
    <ErrorBoundary>
      <div class="p-6 text-text-base">Child content renders normally.</div>
    </ErrorBoundary>
  ),
};

export const WithError: Story = {
  render: () => (
    <ErrorBoundary>
      <Crash />
    </ErrorBoundary>
  ),
};

export const RetryFlow: Story = {
  render: () => {
    const [shouldThrow, setShouldThrow] = useState(true);
    return (
      <ErrorBoundary
        fallback={
          <div class="p-6 space-y-4">
            <p class="text-text-base">Custom fallback rendered.</p>
            <button
              class="px-3 py-2 rounded bg-brand-primary text-text-inverse"
              type="button"
              onClick={() => setShouldThrow(false)}
            >
              Render children
            </button>
          </div>
        }
      >
        {shouldThrow ? <Crash /> : <div class="p-6">Recovered content.</div>}
      </ErrorBoundary>
    );
  },
};
