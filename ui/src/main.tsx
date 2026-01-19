/**
 * Application Entry Point
 *
 * Initializes the The Seed React application with:
 * - React StrictMode for development warnings
 * - Global error boundary for crash protection
 * - React Query for API state management (#890)
 * - Profile context provider for all configuration (profiles own all settings)
 * - Root App component
 *
 * The application is mounted to the DOM element with id="root"
 * defined in index.html.
 */

import { QueryClientProvider } from '@tanstack/react-query';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ProfileProvider } from './contexts/profile-context';
import { getQueryClient } from './lib/query-client';
import './index.css';

// Mount the React application to the root DOM element
// QueryClientProvider enables React Query for API caching and deduplication
// ProfileProvider now manages both profiles AND all user settings
const rootElement: HTMLElement | null = document.getElementById('root');
if (!rootElement) {
  throw new Error('Root element not found');
}
createRoot(rootElement).render(
  <StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={getQueryClient()}>
        <ProfileProvider>
          <App />
        </ProfileProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  </StrictMode>,
);
