# Vendor API integrations

Seed can read device state from a vendor's management API (#3518). Each
integration is **off until an administrator turns it on**, and each is its own
question. A new install reports that it has not been asked yet
(`choice: "unanswered"`), so the web UI can ask at first run with "no" as the
default answer. Unanswered and off behave the same: Seed builds no request and
opens no connection because of the integration, which keeps an air-gapped
install silent.

## Cisco Meraki

The Meraki integration reads the Dashboard API: every organization the API key
can see, every device in it, and each device's availability. It reads once
when it is turned on, again at each start, and then every five minutes.

```text
GET /api/v1/integrations/meraki            (admin)
PUT /api/v1/integrations/meraki            (admin, CSRF)
GET /api/v1/integrations/meraki/devices
```

```json
{
  "choice": "on",
  "apiKey": "<Dashboard API key>"
}
```

- **API key.** Generate one under My Profile in the Meraki dashboard, with
  API access enabled for the organization. Read-only organization access is
  enough. The key is stored encrypted with the credential keyring and is never
  served back; `hasApiKey` says whether one is stored. Leave `apiKey` out to
  keep the stored one; `clearApiKey: true` removes it.
- **Endpoint.** `endpoint` defaults to `https://api.meraki.com/api/v1`. A
  regional dashboard has its own base URL (for example
  `https://api.meraki.ca/api/v1`). Plain http is refused because the request
  carries the key, redirects are not followed, and a pagination link to any
  other host stops the read.
- **Status.** The settings GET reports `lastAttempt`, `lastSuccess` and
  `lastError`. A read fails as a whole if any organization cannot be read, and
  the error names that organization's ID.

The devices GET lists each device's organization, network ID, serial, name,
model, product type, MAC, LAN IP, firmware and status (`online`, `alerting`,
`offline` or `dormant`). It is empty while the integration is off or before its
first read. Turning the integration off drops what an earlier read returned.

The same setting lives in the config file under `integrations.meraki`
(`choice` is `off` or `on`), so an installer can answer the question before
first start. Write the key through the API; the config file holds only its
keyring ciphertext.

## Palo Alto Networks PAN-OS

The Palo Alto integration reads the PAN-OS XML API of one firewall or one
Panorama. Against a firewall it reads the firewall itself; against Panorama it
reads Panorama and every firewall Panorama manages. It reads once when it is
turned on, again at each start, and then every five minutes.

```text
GET /api/v1/integrations/paloalto            (admin)
PUT /api/v1/integrations/paloalto            (admin, CSRF)
GET /api/v1/integrations/paloalto/devices
```

```json
{
  "choice": "on",
  "endpoint": "https://panorama.example.net",
  "apiKey": "<PAN-OS API key>"
}
```

- **Endpoint.** PAN-OS has no default, so `endpoint` is required to turn the
  integration on. Give the management interface's https URL; Seed calls
  `<endpoint>/api/`. The management certificate must chain to a CA the Seed
  host trusts: install the firewall's CA in the host's trust store if it signs
  its own. Plain http is refused because the request carries the key, and
  redirects are not followed.
- **API key.** Generate one for an admin role with XML API access to
  operational requests only (`type=keygen` or the web UI). Seed sends it in the
  `X-PAN-KEY` header, never in the URL. It is stored encrypted with the
  credential keyring and is never served back; `hasApiKey`, leaving `apiKey`
  out and `clearApiKey: true` work as for Meraki.
- **Status.** The settings GET reports `lastAttempt`, `lastSuccess` and
  `lastError`. An API error is reported by its PAN-OS error code; the response
  text is never quoted.

The devices GET lists the endpoint first, then the firewalls Panorama manages
in serial order. Each has its serial, hostname, model, family, PAN-OS version
(`swVersion`), management address and status: `connected` for the endpoint,
which answered, and for each managed firewall whether it is connected to
Panorama. `managedBy` is Panorama's serial on the firewalls it reported, and
`haState` is a firewall's high-availability state as Panorama last saw it. The
list is empty while the integration is off or before its first read, and
turning the integration off drops what an earlier read returned.

The setting lives in the config file under `integrations.paloAlto`.
