import { type ReactNode, useMemo } from 'react';
import { Router } from 'wouter';
import { memoryLocation } from 'wouter/memory-location';

interface StoryRouterProps {
  children: ReactNode;
}

export function StoryRouter({ children }: StoryRouterProps) {
  const location = useMemo(() => memoryLocation(), []);
  return <Router hook={location.hook}>{children}</Router>;
}
