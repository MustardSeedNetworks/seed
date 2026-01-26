import type { Meta, StoryObj } from '@storybook/react-vite';
import { UpdateSettings } from './UpdateSettings';

const meta = {
  title: 'Settings/UpdateSettings',
  component: UpdateSettings,
} satisfies Meta<typeof UpdateSettings>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    currentVersion: '1.2.3',
    onUpdateApplied: () => {},
  },
};
