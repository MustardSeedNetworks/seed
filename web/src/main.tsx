/**
 * Application Entry Point
 *
 * Initializes the The Seed React application with:
 * - React StrictMode for development warnings
 * - Global error boundary for crash protection
 * - Profile context provider for all configuration (profiles own all settings)
 * - Root App component
 *
 * The application is mounted to the DOM element with id="root"
 * defined in index.html.
 */

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { ProfileProvider } from "./contexts/ProfileContext";
import "./index.css";

// Mount the React application to the root DOM element
// ProfileProvider now manages both profiles AND all user settings
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ErrorBoundary>
      <ProfileProvider>
        <App />
      </ProfileProvider>
    </ErrorBoundary>
  </StrictMode>
);
