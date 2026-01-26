import type { Meta, StoryObj } from '@storybook/react-vite';
import { SLADashboardCard } from './SlaDashboardCard';

const meta = {
  title: 'Cards/SLADashboardCard',
  component: SLADashboardCard,
} satisfies Meta<typeof SLADashboardCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    className: 'w-[360px]',
  },
};
