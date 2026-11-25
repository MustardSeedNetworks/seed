import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import { SettingsTabs } from './SettingsTabs';

afterEach(cleanup);

const tabs = [
  { id: 'general', label: 'General' },
  { id: 'appearance', label: 'Appearance', icon: <svg data-testid="icon" /> },
  { id: 'about', label: 'About' },
];

function Harness() {
  const [value, setValue] = useState('general');
  return (
    <SettingsTabs tabs={tabs} value={value} onChange={setValue} label="Settings sections">
      <p>{`panel:${value}`}</p>
    </SettingsTabs>
  );
}

describe('SettingsTabs', () => {
  it('is a labelled tablist whose active tab owns the panel', () => {
    render(<Harness />);
    expect(screen.getByRole('tablist', { name: 'Settings sections' })).toBeTruthy();
    expect(screen.getAllByRole('tab')).toHaveLength(3);
    const general = screen.getByRole('tab', { name: 'General' });
    expect(general.getAttribute('aria-selected')).toBe('true');
    expect(screen.getByRole('tabpanel', { name: 'General' }).textContent).toBe('panel:general');
  });

  it('selects a tab on click and swaps the panel', () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('tab', { name: 'About' }));
    expect(screen.getByRole('tabpanel', { name: 'About' }).textContent).toBe('panel:about');
  });

  it('puts only the active tab in the Tab order', () => {
    render(<Harness />);
    expect(screen.getAllByRole('tab').map((tab) => tab.getAttribute('tabindex'))).toEqual([
      '0',
      '-1',
      '-1',
    ]);
  });

  it('moves and selects with the arrow keys, wrapping at both ends', () => {
    render(<Harness />);
    const list = screen.getByRole('tablist');
    fireEvent.keyDown(list, { key: 'ArrowRight' });
    expect(screen.getByRole('tab', { name: 'Appearance' }).getAttribute('aria-selected')).toBe(
      'true',
    );
    expect(document.activeElement).toBe(screen.getByRole('tab', { name: 'Appearance' }));

    fireEvent.keyDown(list, { key: 'ArrowLeft' });
    fireEvent.keyDown(list, { key: 'ArrowLeft' });
    expect(screen.getByRole('tab', { name: 'About' }).getAttribute('aria-selected')).toBe('true');

    fireEvent.keyDown(list, { key: 'ArrowRight' });
    expect(screen.getByRole('tab', { name: 'General' }).getAttribute('aria-selected')).toBe('true');
  });

  it('jumps with Home and End and ignores other keys', () => {
    render(<Harness />);
    const list = screen.getByRole('tablist');
    fireEvent.keyDown(list, { key: 'End' });
    expect(screen.getByRole('tab', { name: 'About' }).getAttribute('aria-selected')).toBe('true');
    fireEvent.keyDown(list, { key: 'Home' });
    expect(screen.getByRole('tab', { name: 'General' }).getAttribute('aria-selected')).toBe('true');
    fireEvent.keyDown(list, { key: 'a' });
    expect(screen.getByRole('tab', { name: 'General' }).getAttribute('aria-selected')).toBe('true');
  });

  it('scrolls the strip inside itself at 390px and declares it', () => {
    render(<Harness />);
    const scroller = screen.getByRole('tablist').parentElement;
    expect(scroller?.className).toContain('overflow-x-auto');
    expect(scroller?.getAttribute('data-phone-width-exempt')).toBe('tab-scroll');
    expect(screen.getByRole('tablist').className).toContain('min-w-max');
  });

  it('hides tab icons from assistive technology', () => {
    render(<Harness />);
    expect(screen.getByTestId('icon').parentElement?.getAttribute('aria-hidden')).toBe('true');
  });

  it('keeps the strip pinned on its own, with a background and no dependence on the container padding', () => {
    render(<Harness />);
    const strip = screen.getByRole('tablist').parentElement?.parentElement;
    expect(strip?.className).toContain('sticky');
    expect(strip?.className).toContain('top-0');
    expect(strip?.className).toContain('bg-surface-raised');
    expect(strip?.className).not.toMatch(/(^|\s)-(m|mx|mt|top)-/);
  });
});
