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
  args: { children: null },
  render: () => (
    <ErrorBoundary>
      <div className="p-6 text-text-base">Child content renders normally.</div>
    </ErrorBoundary>
  ),
};

export const WithError: Story = {
  args: { children: null },
  render: () => (
    <ErrorBoundary>
      <Crash />
    </ErrorBoundary>
  ),
};

export const RetryFlow: Story = {
  args: { children: null },
  render: () => {
    const [shouldThrow, setShouldThrow] = useState(true);
    return (
      <ErrorBoundary
        fallback={
          <div className="p-6 space-y-4">
            <p className="text-text-base">Custom fallback rendered.</p>
            <button
              className="px-3 py-2 rounded bg-brand-primary text-text-inverse"
              type="button"
              onClick={() => setShouldThrow(false)}
            >
              Render children
            </button>
          </div>
        }
      >
        {shouldThrow ? <Crash /> : <div className="p-6">Recovered content.</div>}
      </ErrorBoundary>
    );
  },
};
