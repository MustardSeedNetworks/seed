import type { Decorator, Meta, StoryObj } from '@storybook/react-vite';
import { useTestRunSignal, useTestRunStore } from '../../stores/testRunStore';
import { cn, spacing } from '../../styles/theme';
import { Fab } from './fab';

/**
 * The Floating Action Button (FAB) provides quick access to running all diagnostic tests.
 * It's positioned in the bottom-right corner and shows a loading spinner during execution.
 */
const meta: Meta<typeof Fab> = {
  title: 'UI/FAB',
  component: Fab,
  parameters: {
    layout: 'fullscreen',
  },
  tags: ['autodocs'],
  decorators: [
    (Story: Parameters<Decorator>[0]) => (
      <div className="relative h-96 bg-surface-base">
        <div className={cn(spacing.pad.default)}>
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
type Story = StoryObj<typeof Fab>;

export const Default: Story = {};

export const WithSimulatedTest: Story = {
  render: () => {
    // Simulate a card-managed run that completes after 3 seconds via the store.
    useTestRunSignal((): void => {
      useTestRunStore.getState().awaitTests(['speedtest']);
      setTimeout(() => {
        useTestRunStore.getState().reportComplete('speedtest');
      }, 3000);
    });

    return <Fab />;
  },
  parameters: {
    docs: {
      description: {
        story: 'Click the FAB to see it enter loading state. Tests complete after 3 seconds.',
      },
    },
  },
};

export const CustomPosition: Story = {
  args: {
    className: 'bottom-20 right-20',
  },
  parameters: {
    docs: {
      description: {
        story: 'The FAB position can be customized via className prop.',
      },
    },
  },
};
