/**
 * Button.a11y.test.tsx — axe gate for the Button + IconButton primitives.
 *
 * These two components are the foundation of icon-button discipline: IconButton
 * makes aria-label a required prop, and Button always renders text children.
 * If anything in this file fails, the harness is broken or a regression has
 * landed in the primitives — investigate before tweaking the test.
 */
import { render } from '@testing-library/react';
import type React from 'react';
import { describe, it } from 'vitest';
import { expectNoAxeViolations } from '../../test/a11y';
import { Button, IconButton } from './Button';

// Minimal placeholder icon (axe needs no real svg here).
function Dot(): React.JSX.Element {
  return <span aria-hidden="true">·</span>;
}

describe('Button — a11y', () => {
  it('solid+violet has no axe violations', async () => {
    const { container } = render(<Button>Save</Button>);
    await expectNoAxeViolations(container);
  });

  it('all tones+variants render without a11y violations', async () => {
    const tones = ['violet', 'red', 'green', 'blue', 'gray'] as const;
    const variants = ['solid', 'outline', 'ghost', 'secondary'] as const;
    for (const variant of variants) {
      for (const tone of tones) {
        const { container } = render(
          <Button variant={variant} tone={tone}>
            Action
          </Button>,
        );
        await expectNoAxeViolations(container);
      }
    }
  });

  it('disabled button has no a11y violations', async () => {
    const { container } = render(<Button disabled>Save</Button>);
    await expectNoAxeViolations(container);
  });

  it('button with leading icon + label has no a11y violations', async () => {
    const { container } = render(<Button leftIcon={<Dot />}>Save</Button>);
    await expectNoAxeViolations(container);
  });
});

describe('IconButton — a11y', () => {
  it('icon-only button with aria-label is accessible', async () => {
    const { container } = render(<IconButton icon={<Dot />} aria-label="Refresh status" />);
    await expectNoAxeViolations(container);
  });

  it('all tones+variants render without a11y violations', async () => {
    const tones = ['violet', 'red', 'green', 'blue', 'gray'] as const;
    const variants = ['solid', 'outline', 'ghost'] as const;
    for (const variant of variants) {
      for (const tone of tones) {
        const { container } = render(
          <IconButton icon={<Dot />} aria-label="Action" variant={variant} tone={tone} />,
        );
        await expectNoAxeViolations(container);
      }
    }
  });
});
