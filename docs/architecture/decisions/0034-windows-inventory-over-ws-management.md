# 0034 — Windows inventory over WS-Management on HTTPS, with a pinned certificate

Status: Accepted
Date: 2026-10-16

## Context

P-D11 (#3506) asks for a Windows host's operating system, hardware and
installed software. Seed already reads what a Windows host exposes through
HOST-RESOURCES-MIB (the `host_resources` collector), but the Windows SNMP
service is deprecated, off by default and says nothing about installed
software. The inventory lives in WMI, and WMI is reachable from another
machine three ways:

1. **DCOM** (MS-DCOM and MS-WMI over RPC): TCP 135 plus a dynamic high port
   per call. No maintained Go implementation exists, the dynamic port range
   is the firewall exception every hardening guide removes, and Microsoft's
   DCOM hardening (KB5004442) already broke clients that did not use packet
   integrity.
2. **WS-Management** (MS-WSMV, the WinRM service): SOAP over HTTP on 5985 or
   HTTPS on 5986. It is the transport PowerShell's CIM cmdlets use, the WinRM
   service runs by default on Windows Server, and a WQL query is one
   `Enumerate` request.
3. **SSH running PowerShell** (`Get-CimInstance … | ConvertTo-Json`). Seed has
   the SSH stack from P-D1, but OpenSSH Server is an optional feature that is
   off on every Windows edition, and the inventory then depends on a shell's
   text output rather than a protocol.

WS-Management over plain HTTP is not usable as shipped: Windows refuses
unencrypted traffic (`AllowUnencrypted=false`), so a client must seal each
message with the NTLM or Kerberos session key. That means implementing NTLM
sealing (RC4 and HMAC-MD5) in Seed, or asking operators to set
`AllowUnencrypted=true` and send credentials in the clear.

## Decision

### 1. Transport: WS-Management over HTTPS only

Seed talks WS-Management to TCP 5986, TLS 1.2 or later, and nothing else.
There is no HTTP mode, no DCOM and no SSH fallback. The operator enables an
HTTPS listener on the host (one `winrm` command or a Group Policy setting);
the setup document gives both.

### 2. Certificate trust: pinned, confirmed by an operator

WinRM listeners mostly carry self-signed or internal-CA certificates, so
verifying against the system roots would fail for most estates, and
`InsecureSkipVerify` would hand the password to whoever answers. The P-D1
host-key rule applies instead: a target with no pinned certificate is
contacted only for the TLS handshake, its leaf certificate's SHA-256 is stored
as pending, and no credential is sent until an operator confirms it. A pinned
target that presents another certificate is refused as a mismatch, and the
operator re-confirms after a planned renewal.

### 3. Authentication: NTLMv2 inside TLS

The WinRM service accepts Negotiate by default and refuses Basic, so Seed
authenticates with NTLMv2 through `github.com/Azure/go-ntlmssp` (MIT, the
library `masterzen/winrm` uses). It works for local and
domain accounts with no Kerberos configuration on the appliance. TLS carries
the confidentiality, so no message sealing is implemented. Kerberos is not in
this decision; it becomes necessary when Microsoft disables NTLM by default
(see Consequences).

The credential is a new vault kind, `winrm`: a user (`DOMAIN\user` or
`user@domain`) and a password stored as vault ciphertext, under the same schema
checks that make the SSH and SNMP shapes exclusive. A WMI target has its own
row with host, port and pin, like `device_config_targets`; it is not a polling
target, because nothing in the SNMP poller can use it.

### 4. What is read

Read-only WQL `SELECT` queries in `root\cimv2`. Seed never invokes a WMI
method.

| Inventory | Class | Properties |
| --- | --- | --- |
| Operating system | `Win32_OperatingSystem` | Caption, Version, BuildNumber, OSArchitecture, InstallDate, LastBootUpTime |
| System | `Win32_ComputerSystem` | Manufacturer, Model, Domain, TotalPhysicalMemory |
| Firmware | `Win32_BIOS` | SerialNumber, SMBIOSBIOSVersion |
| Processors | `Win32_Processor` | Name, NumberOfCores, NumberOfLogicalProcessors |
| Fixed disks | `Win32_LogicalDisk WHERE DriveType = 3` | DeviceID, Size, FreeSpace |
| Network adapters | `Win32_NetworkAdapterConfiguration WHERE IPEnabled = TRUE` | Description, MACAddress, IPAddress |
| Installed software | `Win32_InstalledWin32Program` | Name, Version, Vendor |

Installed software never comes from `Win32_Product`. Enumerating it makes
Windows Installer run a consistency check on every MSI package, which can
start repairs on the host being inventoried. `Win32_InstalledWin32Program` is
read-only; slice 2 confirms it on the lab host, and if a supported Windows
release lacks it, the row is amended rather than falling back to
`Win32_Product`.

### 5. Storage: the latest inventory, replaced per collection

Each target keeps its last successful inventory and the outcome of its last
attempt (time, status, error). A successful collection replaces the previous
inventory in one transaction. There is no inventory history and no change
alert: the acceptance asks what a host is, and history is a separate feature
that would need its own row.

### 6. Tier and authorization

The routes sit behind the existing `server_monitoring` Pro feature, which
already gates host monitoring through `host_resources`, so the licence
catalogue gains no string. Reads are any authenticated role; target and
credential writes are operator and above through `writeGated`, with CSRF, as
for configuration backup targets.

## Consequences

- An operator must enable an HTTPS listener and open TCP 5986 on each host.
  A Windows Server host with only the default HTTP listener reports that the
  port is closed, and the setup document says why HTTP is refused.
- Seed gains one dependency, `go-ntlmssp`. The WS-Management envelope is a
  few fixed SOAP documents written with `encoding/xml`, not a third-party WinRM
  client: those clients exist to run remote commands, which Seed never does.
- Microsoft has deprecated NTLM (removing NTLMv1 in Windows 11 24H2 and Server
  2025). When NTLMv2 is disabled by default, Kerberos authentication is a new
  row; the target, pin and storage model in this ADR do not change.
- A certificate renewal on the host stops collection until an operator
  re-confirms the new fingerprint. That is the cost of not trusting on first
  use, as with SSH host keys.
