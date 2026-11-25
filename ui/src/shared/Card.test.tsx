import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { Card } from './Card';

afterEach(cleanup);

describe('Card', () => {
  it('takes its radius and border from the shared panel class, and is flat', () => {
    render(<Card data-testid="card">body</Card>);
    const { className } = screen.getByTestId('card');
    expect(className).toContain('panel');
    expect(className).not.toMatch(/rounded|shadow|border|bg-/);
  });

  it('renders the title as an h2, never the page h1', () => {
    render(
      <Card title="Link" headerAction={<span>pill</span>}>
        body
      </Card>,
    );
    expect(screen.getByRole('heading', { level: 2, name: 'Link' })).toBeTruthy();
    expect(screen.queryByRole('heading', { level: 1 })).toBeNull();
    expect(screen.getByText('pill')).toBeTruthy();
  });

  it('has no header row without a title or action', () => {
    render(<Card data-testid="card">body</Card>);
    expect(screen.getByTestId('card').querySelector('h2')).toBeNull();
  });

  it('can shrink inside a flex or grid parent so it never widens the page', () => {
    render(<Card data-testid="card">body</Card>);
    expect(screen.getByTestId('card').className).toContain('min-w-0');
  });
});
