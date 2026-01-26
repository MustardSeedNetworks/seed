import type { Meta, StoryObj } from '@storybook/react-vite';
import { AirMapperImport } from './AirMapperImport';

const meta = {
  title: 'Survey/AirMapperImport',
  component: AirMapperImport,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof AirMapperImport>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    onImport: () => {},
    onCancel: () => {},
  },
};
