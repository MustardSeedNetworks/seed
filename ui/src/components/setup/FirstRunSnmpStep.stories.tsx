import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, userEvent, within } from 'storybook/test';
import { FirstRunSnmpStep } from './FirstRunSnmpStep';

/**
 * The second step of first-run setup: the optional SNMP read community that
 * lets discovery identify what its first sweep finds (#2722).
 */
const meta: Meta<typeof FirstRunSnmpStep> = {
  title: 'Setup/FirstRunSnmpStep',
  component: FirstRunSnmpStep,
  parameters: { layout: 'fullscreen' },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof FirstRunSnmpStep>;

/** As an operator first sees it: empty field, no community suggested. */
export const Offered: Story = {
  args: { onDone: () => undefined },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const field = canvas.getByLabelText<HTMLInputElement>('SNMP community string');
    await expect(field.value).toBe('');
    await expect(canvas.getByRole('button', { name: 'Skip for now' })).toBeEnabled();
  },
};

/** The field is a secret: typed characters are masked, like every password input. */
export const Typed: Story = {
  args: { onDone: () => undefined },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const field = canvas.getByLabelText<HTMLInputElement>('SNMP community string');
    await userEvent.type(field, 's3cret-ro');
    await expect(field.value).toBe('s3cret-ro');
    await expect(field.type).toBe('password');
  },
};
