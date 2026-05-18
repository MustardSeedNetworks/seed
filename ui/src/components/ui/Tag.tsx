/**
 * Tag primitive — ported from niac UI kit (Phase B).
 */
import type { FC, ReactNode } from 'react';

type TagColorScheme = 'gray' | 'red' | 'green' | 'blue' | 'yellow' | 'purple' | 'violet' | 'cyan';

interface TagProps {
  children: ReactNode;
  colorScheme?: TagColorScheme;
  className?: string;
}

const colorStyles: Record<TagColorScheme, string> = {
  gray: 'bg-bg-muted/20 text-text-secondary border-border-muted/30',
  red: 'bg-status-error/20 text-status-error border-status-error/30',
  green: 'bg-status-success/20 text-status-success border-status-success/30',
  blue: 'bg-status-info/20 text-status-info border-status-info/30',
  yellow: 'bg-status-warning/20 text-status-warning border-status-warning/30',
  purple: 'bg-brand-primary/20 text-brand-accent border-brand-primary/30',
  violet: 'bg-brand-primary/20 text-brand-accent border-brand-primary/30',
  cyan: 'bg-status-info/20 text-status-info border-status-info/30',
};

export const Tag: FC<TagProps> = ({ children, colorScheme = 'gray', className = '' }) => (
  <span
    class={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${colorStyles[colorScheme]} ${className}`}
  >
    {children}
  </span>
);
