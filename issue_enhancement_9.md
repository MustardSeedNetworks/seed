## Summary
Enhance the Public IP Card to display additional context about the public IP address, including its estimated geolocation (City/Country) and owning network (ISP/ASN).

## Problem or Goal
Knowing the public IP address is useful, but knowing who owns it and where it's located provides much richer context for diagnostics. This information can help a technician quickly determine if they are on the correct ISP, if traffic is being routed unexpectedly, or if a VPN is active.

## Proposed Solution
- Integrate a third-party IP geolocation and ASN lookup service. Free services like `ip-api.com` or `ipinfo.io` provide simple JSON APIs for this.
- When the public IP address is determined, the backend should make a follow-up query to this service to fetch the associated data.
- The results should be cached to avoid excessive API calls.
- The Public IP Card should be updated to display the new fields:
    - **ISP / ASN:** (e.g., "AS15169 Google LLC")
    - **Location:** (e.g., "Mountain View, California, US")
- The card should also be enhanced to track and log any changes to the public IP address over time.

## Acceptance Criteria
- [ ] The backend queries a geolocation/ASN service after discovering the public IP.
- [ ] The Public IP Card displays the ISP/ASN and the City/Country for the current public IP.
- [ ] The lookup results are cached appropriately.
- [ ] The UI gracefully handles cases where the geolocation service is unavailable.
- [ ] The card logs and displays a history of public IP address changes.
