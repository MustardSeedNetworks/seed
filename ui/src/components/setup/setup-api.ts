/**
 * Setup API utilities
 *
 * Provides API functions for the initial setup process.
 */

// API base URL for setup endpoints
const API_BASE: string = import.meta.env.VITE_API_BASE || '';

/**
 * Response from /api/setup/status endpoint
 */
export interface SetupStatusResponse {
  needsSetup: boolean; // True if initial setup is required
  username?: string; // Default admin username
  suggestedPassword?: string; // Pre-generated password (secure random)
  setupToken?: string; // Security fix #724, #758: One-time token for setup completion
}

/**
 * Checks if initial setup is required by querying the API status endpoint.
 */
export async function checkSetupStatus(): Promise<SetupStatusResponse> {
  try {
    const response = await fetch(`${API_BASE}/api/setup/status`);
    if (response.ok) {
      return response.json() as Promise<SetupStatusResponse>;
    }
  } catch {
    // If we can't reach the API, assume setup is complete
  }
  return { needsSetup: false };
}
