/**
 * Application Entry Point
 *
 * Initializes the The Seed/The Seed React application with:
 * - React StrictMode for development warnings
 * - Global error boundary for crash protection
 * - Settings context provider for configuration management
 * - Root App component
 *
 * The application is mounted to the DOM element with id="root"
 * defined in index.html.
 */

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { SettingsProvider } from "./contexts/SettingsContext";
import "./index.css";

// Mount the React application to the root DOM element
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ErrorBoundary>
      <SettingsProvider>
        <App />
      </SettingsProvider>
    </ErrorBoundary>
  </StrictMode>
);
