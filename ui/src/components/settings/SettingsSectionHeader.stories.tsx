import type { Meta, StoryObj } from '@storybook/react-vite';
import { Settings } from '../ui/icons';
import { SettingsSectionHeader } from './SettingsSectionHeader';

const meta = {
  title: 'Settings/SettingsSectionHeader',
  component: SettingsSectionHeader,
} satisfies Meta<typeof SettingsSectionHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    icon: Settings,
    titleKey: 'sections.appearance',
  },
};

export const Saving: Story = {
  args: {
    icon: Settings,
    titleKey: 'sections.appearance',
    status: 'saving',
  },
};

export const Saved: Story = {
  args: {
    icon: Settings,
    titleKey: 'sections.appearance',
    status: 'saved',
  },
};
