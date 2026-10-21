import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, userEvent, waitFor, within } from 'storybook/test';
import { FirstRunEoxStep } from './FirstRunEoxStep';

/**
 * The last step of first-run setup: the admin's answer to the opt-in EoX
 * feed (UI-SEED-47). Off is the default answer (owner 2026-10-14).
 */
const meta: Meta<typeof FirstRunEoxStep> = {
  title: 'Setup/FirstRunEoxStep',
  component: FirstRunEoxStep,
  parameters: { layout: 'fullscreen' },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof FirstRunEoxStep>;

/** As an admin first sees it: the feed unanswered, the answer on Off. */
export const Asked: Story = {
  args: { onDone: () => undefined },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    // The answer can be given once the stored feed has loaded.
    await waitFor(() => expect(canvas.getByTestId('setup-eox-save')).toBeEnabled());
    await expect(canvas.getByTestId('eox-feed-off')).toBeChecked();
  },
};

/** Turning it on asks for the Cisco API client the feed needs. */
export const TurnedOn: Story = {
  args: { onDone: () => undefined },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await waitFor(() => expect(canvas.getByTestId('setup-eox-save')).toBeEnabled());
    await userEvent.click(canvas.getByTestId('eox-feed-on'));
    await expect(canvas.getByTestId('eox-feed-client-id')).toBeVisible();
    await expect(canvas.getByTestId('eox-feed-client-secret')).toHaveAttribute('type', 'password');
  },
};
