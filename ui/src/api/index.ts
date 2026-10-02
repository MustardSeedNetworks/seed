/**
 * API Module
 *
 * Exports the API client and related utilities for backend communication.
 */
export {
  ApiError,
  api,
  beginSession,
  clearCSRFToken,
  SessionExpiredError,
  setSessionExpiredCallback,
} from './client';
