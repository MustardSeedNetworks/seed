import type { StatusTone } from './StatusPill';

export const toneWash: Record<StatusTone, string> = {
  success: 'bg-status-success/10',
  warning: 'bg-status-warning/10',
  error: 'bg-status-error/10',
  info: 'bg-status-info/10',
  unknown: '',
};

export const cellPad = 'px-4 py-2';
export const focusRing =
  'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary';
export const pagerButton =
  'inline-flex h-8 w-8 items-center justify-center rounded-lg text-text-primary hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed disabled:text-text-disabled disabled:hover:bg-transparent';
