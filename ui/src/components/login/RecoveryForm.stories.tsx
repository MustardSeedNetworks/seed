import type { Meta, StoryObj } from '@storybook/react-vite';
import { RecoveryForm } from './RecoveryForm';

const meta = {
  title: 'Auth/RecoveryForm',
  component: RecoveryForm,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof RecoveryForm>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    onRecoveryComplete: () => {},
    onBackToLogin: () => {},
    remainingTime: 600,
    tokenFilePath: '/var/lib/seed/.recovery-token',
  },
};
