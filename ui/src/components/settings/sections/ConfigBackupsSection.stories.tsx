import type { Meta, StoryObj } from '@storybook/react-vite';
import { ConfigBackupsSection } from './ConfigBackupsSection';
import { expandSections } from './storyPlay';

const meta = {
  title: 'Settings/ConfigBackupsSection',
  component: ConfigBackupsSection,
  play: expandSections,
} satisfies Meta<typeof ConfigBackupsSection>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
