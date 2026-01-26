import type { Meta, StoryObj } from '@storybook/react-vite';
import { ConfigBackupsSection } from './ConfigBackupsSection';

const meta = {
  title: 'Settings/ConfigBackupsSection',
  component: ConfigBackupsSection,
} satisfies Meta<typeof ConfigBackupsSection>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
