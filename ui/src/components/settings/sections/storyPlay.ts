import type { Meta } from '@storybook/react-vite';

/**
 * Settings sections render inside a CollapsibleSection that starts closed, so
 * a story left as rendered hands axe the header and nothing else. Opening
 * every closed section first puts the form controls in front of the gate.
 */
export const expandSections: NonNullable<Meta['play']> = async ({ canvas, userEvent }) => {
  for (const toggle of canvas.queryAllByRole('button', { expanded: false })) {
    await userEvent.click(toggle);
  }
};
