/**
 * GatedButton: the shared Button for an action a role may not take. The reason
 * shows in the local Tooltip, and the button stays in the tab order as
 * `aria-disabled` (Button styles it like `disabled`) so the reason can be read
 * on focus as well as on hover. With no reason it is an ordinary Button, and a
 * `hint` is shown the same way.
 */

import { Button, type ButtonProps } from '../shared/Button';
import { Tooltip } from '../shared/Tooltip';

type GatedButtonProps = Extract<ButtonProps, { iconOnly?: false }> & {
  /** Why the action is unavailable; absent when it is available. */
  reason?: string | undefined;
  /** What the action does, shown the same way while it is available. */
  hint?: string | undefined;
};

export function GatedButton({ reason, hint, onClick, ...button }: GatedButtonProps) {
  return (
    <Tooltip text={reason ?? hint}>
      {(description) => (
        <Button
          {...button}
          aria-disabled={reason ? true : undefined}
          aria-describedby={description['aria-describedby'] || undefined}
          onClick={(event) => {
            if (reason) {
              description.onClick();
              return;
            }
            onClick?.(event);
          }}
        />
      )}
    </Tooltip>
  );
}
