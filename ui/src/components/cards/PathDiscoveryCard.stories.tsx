import type { Meta, StoryObj } from '@storybook/react-vite';
import { PathDiscoveryCard } from './PathDiscoveryCard';

const meta = {
  title: 'Cards/PathDiscoveryCard',
  component: PathDiscoveryCard,
} satisfies Meta<typeof PathDiscoveryCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    gateway: '192.168.1.1',
    dnsServer: '8.8.8.8',
  },
};
