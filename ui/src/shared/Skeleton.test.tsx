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

describe('Skeleton variants', () => {
  it('is a full-width 16px text line by default', () => {
    const { container } = render(<Skeleton />);
    const bar = container.firstElementChild as HTMLElement;
    expect(bar.className).toContain('h-4');
    expect(bar.className).toContain('w-full');
    expect(bar.className).toContain('rounded');
  });

  it('takes px numbers and CSS strings for width and height', () => {
    const { container } = render(<Skeleton variant="rectangular" width={120} height="3rem" />);
    const bar = container.firstElementChild as HTMLElement;
    expect(bar.style.width).toBe('120px');
    expect(bar.style.height).toBe('3rem');
    expect(bar.className).toContain('rounded-lg');
    expect(bar.className).not.toContain('h-4');
  });

  it('is round as a circle', () => {
    const { container } = render(<Skeleton variant="circular" width={40} height={40} />);
    expect((container.firstElementChild as HTMLElement).className).toContain('rounded-full');
  });

  it('stacks lines with the last at 75%, all decorative', () => {
    const { container } = render(<Skeleton lines={3} className="my-2" />);
    const wrapper = container.firstElementChild as HTMLElement;
    expect(wrapper.getAttribute('aria-hidden')).toBe('true');
    expect(wrapper.className).toContain('my-2');
    const bars = Array.from(wrapper.children) as HTMLElement[];
    expect(bars).toHaveLength(3);
    expect(bars.map((bar) => bar.style.width)).toEqual(['', '', '75%']);
    for (const bar of bars) {
      expect(bar.className).toContain('motion-safe:animate-pulse');
    }
  });
});
