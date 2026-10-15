# Hardware end-of-life (EoX)

Seed reports where each inventoried part stands against Cisco's end-of-life
milestones: end of sale, end of software maintenance, end of security support
and last date of support (#3459). Parts come from the ENTITY-MIB inventory
that discovery walks; the milestones come from Cisco's EoX data.

Seed ships no EoX data. It gets it in one of two ways, and both are yours to
choose.

## Import a file

Download the records for your product IDs from the Cisco EoX API, then
import the response as it came:

```text
PUT /api/v1/lifecycle/eox      (operator, CSRF, 8 MB)
```

One response or an array of them is accepted. An import replaces the stored
records. This path never makes an outbound request, so it is the one for an
air-gapped site.

## Or turn on the feed

The feed asks the Cisco EoX API about every inventoried product ID once when
it is turned on, again at each start, and then daily. What it returns is
merged into the stored records: a product it covered is replaced, everything
else you imported stays.

The feed is **off until an administrator turns it on**. A new install reports
that it has not been asked yet (`choice: "unanswered"`), so the web UI can ask
at first run with "no" as the default answer. Unanswered and off behave the
same: Seed builds no request and opens no connection because of this feature.

```text
GET /api/v1/lifecycle/feed     (admin)
PUT /api/v1/lifecycle/feed     (admin, CSRF)
```

```json
{
  "choice": "on",
  "clientId": "<Cisco API client ID>",
  "clientSecret": "<Cisco API client secret>"
}
```

- **Credentials.** Cisco's API needs an API client registered to your Cisco
  account (client credentials grant). The secret is stored encrypted with the
  credential keyring and is never served back; `hasClientSecret` says whether
  one is stored. Leave `clientSecret` out to keep the stored one;
  `clearClientSecret: true` removes it.
- **Endpoint.** `endpoint` and `tokenUrl` default to Cisco's. Point them at
  your own https mirror of the API if you run one; a mirror that needs no
  credentials takes no client ID. Plain http is refused because the request
  carries a bearer token, and redirects are not followed.
- **Status.** The GET reports `lastAttempt`, `lastSuccess`, `lastError` and
  how many products the last refresh returned.

The same setting lives in the config file under `lifecycle.eox_feed`
(`choice` is `off` or `on`), so an installer can answer the question before
first start. Write the secret through the API; the config file holds only its
keyring ciphertext.

## Reading the result

```text
GET /api/v1/lifecycle/devices
```

lists every inventoried part with a model name, grouped by device, with its
status today: `not_listed`, `announced`, `end_of_sale`,
`end_of_sw_maintenance`, `end_of_security_support` or `end_of_support`.
