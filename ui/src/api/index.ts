/**
 * API Module
 *
 * Exports the API client and related utilities for backend communication.
 */
export {
  api,
  beginSession,
  clearCSRFToken,
  SessionExpiredError,
  setSessionExpiredCallback,
} from './client';
