# Windows inventory

Seed can read a Windows host's operating system, hardware and installed
software from WMI (P-D11, #3506). It signs in over WS-Management, the protocol
behind the WinRM service, and runs read-only queries. It never starts a
program or calls a WMI method on the host. The design is in
[ADR-0034](architecture/decisions/0034-windows-inventory-over-ws-management.md).

Each host reports:

- **Operating system:** name, version, build, architecture, install date and
  last boot.
- **System:** manufacturer, model, domain and installed memory.
- **Firmware:** serial number and BIOS version.
- **Processors:** name, cores and logical processors.
- **Fixed disks:** each drive letter's size and free space.
- **Network adapters:** each IP-enabled adapter's description, MAC address
  and IP addresses.
- **Installed software:** name, version and vendor, from
  `Win32_InstalledWin32Program`. Seed never reads `Win32_Product`, because
  enumerating it makes Windows Installer check every MSI package on the host.

Seed keeps the last successful inventory of each host. A failed collection
keeps the previous one and records why it failed.

## Preparing a host

Seed connects only to an **HTTPS** WinRM listener on TCP 5986. The HTTP
listener on 5985 is refused: Windows will not accept unencrypted WS-Management
traffic, and the only other way to protect it is message sealing, which Seed
does not do.

On the host, in an elevated PowerShell:

```powershell
Set-Service WinRM -StartupType Automatic
Start-Service WinRM
$cert = New-SelfSignedCertificate -DnsName $env:COMPUTERNAME -CertStoreLocation Cert:\LocalMachine\My
New-Item -Path WSMan:\localhost\Listener -Transport HTTPS -Address * `
  -CertificateThumbPrint $cert.Thumbprint -Force
New-NetFirewallRule -DisplayName 'WinRM HTTPS' -Direction Inbound -Protocol TCP `
  -LocalPort 5986 -Action Allow
$cert.GetCertHashString('SHA256')
```

A certificate from your own CA works the same way: name its thumbprint in
`New-Item`. Across a domain, Group Policy can create the listener instead
(Computer Configuration > Administrative Templates > Windows Components >
Windows Remote Management).

Keep the last line's output. It is the certificate's SHA-256 fingerprint, and
you compare it with the one Seed shows before Seed sends any password.

The account Seed signs in with must be allowed to read WMI remotely. A member
of the host's Administrators group can. For a local account other than the
built-in Administrator, Windows's remote UAC filtering removes administrator
rights over the network; a domain account avoids that.

## Using it

The routes are part of Seed Pro (`server_monitoring`). Creating, changing and
running targets needs the operator role.

1. Store the login in the credential vault. The password is encrypted at rest
   and never returned:

   ```sh
   curl -sk -X POST https://seed.example:8443/api/v1/device-credentials \
     -H "Authorization: Bearer $SEED_TOKEN" -H 'Content-Type: application/json' \
     -d '{"name":"inventory","winrmUser":"LAB\\inventory","winrmPassword":"…"}'
   ```

   `winrmUser` is `DOMAIN\user`, `user@domain`, or a local account name.

2. Add the host, naming the credential. The port defaults to 5986:

   ```sh
   curl -sk -X POST https://seed.example:8443/api/v1/wmi-targets \
     -H "Authorization: Bearer $SEED_TOKEN" -H 'Content-Type: application/json' \
     -d '{"name":"dc01","host":"10.0.0.5","credentialsId":"<credential id>"}'
   ```

3. Start a run with `POST /api/v1/wmi-runs` and an empty body, or with
   `{"targetIds":[…]}` to collect only some hosts. The run is a job, which
   `/api/v1/jobs/events` follows.

4. The first run against a new host stops after the TLS handshake. The host
   reports `cert_unconfirmed`, and the target's `pendingCertSha256` holds
   the certificate the host presented. Compare it with the fingerprint you kept
   from the host. If they match, confirm it:

   ```sh
   curl -sk -X PUT https://seed.example:8443/api/v1/wmi-targets/<id>/certificate \
     -H "Authorization: Bearer $SEED_TOKEN" -H 'Content-Type: application/json' \
     -d '{"fingerprint":"<pendingCertSha256>"}'
   ```

5. Run again. `GET /api/v1/wmi-targets/<id>/inventory` returns the inventory.

## When a collection fails

The target's `lastStatus` says why:

| Status | Meaning |
| --- | --- |
| `cert_unconfirmed` | No certificate is pinned yet. Confirm the pending one (step 4). |
| `cert_mismatch` | The host presented a certificate other than the pinned one. Seed sent nothing. After a planned renewal, `DELETE /api/v1/wmi-targets/<id>/certificate` and confirm the new one. Otherwise, find out what is answering on that address. |
| `auth_failed` | The host refused the account or its password. A host whose WinRM service requires channel binding (`CbtHardeningLevel` set to `Strict`) also refuses Seed, which does not send channel binding tokens; the pinned certificate is what protects the sign-in instead. |
| `unreachable` | Nothing answered on the port, or the host did not finish in time. A host with only the default HTTP listener reports this. |
| `query_failed` | The host answered with a WS-Management fault, for example a class it does not have. `lastError` carries the host's message. |
| `error` | Anything else, such as a credential that was changed to another kind. |

Moving a target to another host or port forgets its certificate, because the
pin belonged to the old address.
