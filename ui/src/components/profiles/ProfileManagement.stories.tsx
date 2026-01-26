import type { Meta, StoryObj } from '@storybook/react-vite';
import { ProfileManagement } from './ProfileManagement';

const meta = {
  title: 'Profiles/ProfileManagement',
  component: ProfileManagement,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof ProfileManagement>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    onClose: () => {},
  },
};
