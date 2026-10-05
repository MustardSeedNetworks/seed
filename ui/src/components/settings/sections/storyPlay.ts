import type { Meta } from '@storybook/react-vite';

/**
 * Settings sections render inside a CollapsibleSection that starts closed, so
 * a story left as rendered hands axe the header and nothing else. Opening
 * every closed section first puts the form controls in front of the gate.
 * It reads no args, so it fits the story of a component that takes no props.
 */
export const expandSections: NonNullable<Meta<unknown>['play']> = async ({ canvas, userEvent }) => {
  for (const toggle of canvas.queryAllByRole('button', { expanded: false })) {
    await userEvent.click(toggle);
  }
};
