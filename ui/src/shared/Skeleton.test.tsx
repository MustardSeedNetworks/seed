import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { Skeleton } from './Skeleton';

afterEach(cleanup);

describe('Skeleton', () => {
  it('is decorative and pulses only when motion is allowed', () => {
    const { container } = render(<Skeleton className="h-6 w-24" />);
    const bar = container.firstElementChild;
    expect(bar?.getAttribute('aria-hidden')).toBe('true');
    expect(bar?.className).toContain('motion-safe:animate-pulse');
    expect(bar?.className).toContain('w-24');
    expect(screen.queryByRole('progressbar')).toBeNull();
  });
});
