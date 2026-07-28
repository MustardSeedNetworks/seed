import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, fn, userEvent, within } from 'storybook/test';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';

type Defect = 'accessibility' | 'interaction';

interface SharedControlsProps {
  defect?: Defect;
  onSave: () => void;
}

function SharedControls({ defect, onSave }: SharedControlsProps) {
  const buttonLabel = <span aria-hidden={defect === 'accessibility'}>Save target</span>;

  return (
    <form className="stack-md" onSubmit={(event) => event.preventDefault()}>
      <Input label="Target address" defaultValue="192.0.2.1" />
      <Button disabled={defect === 'interaction'} onClick={onSave}>
        {buttonLabel}
      </Button>
    </form>
  );
}

const injectedDefect = import.meta.env.VITE_STORYBOOK_INJECT_DEFECT as Defect | undefined;

const meta = {
  title: 'Test/Storybook gate',
  component: SharedControls,
  tags: ['test-ready'],
  parameters: {
    a11y: { test: 'error' },
    seedProfile: false,
  },
} satisfies Meta<typeof SharedControls>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SharedComponentInteraction: Story = {
  args: {
    defect: injectedDefect === 'interaction' ? injectedDefect : undefined,
    onSave: fn(),
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByRole('button', { name: 'Save target' }));
    await expect(args.onSave).toHaveBeenCalledOnce();
  },
};

export const SharedComponentAccessibility: Story = {
  args: {
    defect: injectedDefect === 'accessibility' ? injectedDefect : undefined,
    onSave: fn(),
  },
};
