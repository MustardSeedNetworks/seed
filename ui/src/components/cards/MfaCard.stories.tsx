import type { Meta, StoryObj } from '@storybook/react-vite';
import type { JSX } from 'react';
import { expect, within } from 'storybook/test';
import { MfaCard } from './MfaCard';

/**
 * MfaCard offers the two enrolment choices for a second factor.
 *
 * #2641: both buttons carried `btn btn-secondary`, a class that exists in no
 * stylesheet, and the passkey button's Tooltip wrapper is `display: contents`,
 * so the stack margin that should have separated them generated no box either.
 * The card rendered two unstyled inline buttons touching each other — a reader
 * saw the single word "Set up TOTPAdd passkey". No story covered the card, which
 * is how it shipped; this one asserts the buttons are two separated controls.
 */
const meta = {
  title: 'Cards/MfaCard',
  component: MfaCard,
  parameters: { layout: 'centered', a11y: { test: 'error' } },
  tags: ['autodocs'],
  decorators: [
    (StoryComponent: React.ComponentType): JSX.Element => (
      <div style={{ width: '420px' }}>
        <StoryComponent />
      </div>
    ),
  ],
} satisfies Meta<typeof MfaCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/** Nothing enrolled: both enrolment choices are offered. */
export const NoSecondFactor: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const totp = await canvas.findByTestId('mfa-setup-totp');
    const passkey = canvas.getByTestId('mfa-add-passkey');

    await expect(totp).toHaveAccessibleName('Set up TOTP');
    await expect(passkey).toHaveAccessibleName('Add passkey');

    // Two boxes with real space between them, not one run-together line. A
    // rendered gap is the property #2641 broke, so it is the property asserted.
    const first = totp.getBoundingClientRect();
    const second = passkey.getBoundingClientRect();
    await expect(first.width).toBeGreaterThan(0);
    // The row wraps on a narrow card, so the separation is whichever axis the
    // two buttons differ on.
    await expect(
      Math.max(second.left - first.right, second.top - first.bottom),
    ).toBeGreaterThanOrEqual(8);
  },
};
