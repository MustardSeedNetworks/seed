import type { Meta, StoryObj } from '@storybook/react-vite';
import { LogViewerCard } from './LogViewerCard';

const meta = {
  title: 'Cards/LogViewerCard',
  component: LogViewerCard,
} satisfies Meta<typeof LogViewerCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    className: 'w-[360px]',
  },
};
