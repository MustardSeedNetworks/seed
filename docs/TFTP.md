# TFTP sessions

Seed can serve files to network devices over TFTP: an IOS image to upgrade a
switch, or a configuration to restore. It can also receive files from them,
such as a running configuration or a crash file (#3123).

TFTP has no authentication. Any host that can reach the port can read every
file served, so Seed keeps a session as small as the protocol allows:

- **Off until an admin starts it.** Nothing listens on UDP 69 by default. A
  session never survives a restart of Seed.
- **One interface.** The session binds UDP 69 on the chosen interface's IPv4
  address, not on every address.
- **Only nearby devices.** The session answers requests from the IPv4
  subnets configured on that interface and refuses every other source with a
  short TFTP error. A device one or more routed hops away needs its prefix
  named in `allowFrom` when the session starts, for example
  `"allowFrom":["10.20.0.0/16"]`; the list then replaces the interface's
  subnets, and up to 16 prefixes are accepted.
- **One directory.** Only `<data dir>/tftp` is served; on a packaged Linux
  install that is `/var/lib/seed/tftp`. A request cannot leave it, through
  `..` or through a symbolic link.
- **Read-only unless you allow uploads.** With uploads allowed, a device can
  write new files into `<data dir>/tftp-incoming`, a separate directory that
  is never served, so one device cannot plant a file another will fetch. A
  file appears there under its name only once it has arrived in full. An
  upload cannot replace an existing file, each one is capped at 256 MiB, and
  one session accepts at most 32 upload attempts and 1 GiB in all. Files stay until
  you remove them.
- **Eight transfers at a time.** Further requests are refused at once with a
  "server busy" error, so a flood of requests cannot hold sockets open.
- **Stops itself.** After ten minutes in which no transfer has completed, the
  session stops; refused and failed requests do not count as use. A transfer
  in progress keeps it open, but never past one hour: every session stops an
  hour after it started, busy or not.

Starting and stopping a session needs the admin role and Seed Pro. Every
start, stop and completed transfer is written to the audit log with resource
type `tftp`. For a transfer, the entry records the device's address, the file
name and the bytes moved. Failed and refused requests are written too, but at
most once per device per minute and 32 times per minute in all, so a host
hammering the port cannot flood the log; the rest are counted in an
`event=tftp.suppressed` log line. Each audited entry is also logged as
`event=tftp.start`, `tftp.stop`, `tftp.download` or `tftp.upload`. A stop
records its reason: `operator`, `idle`, `expired` or `shutdown`.

## Using it

1. Copy the files the devices need into the served directory.
2. Start a session on the interface that faces the devices:

   ```sh
   curl -sk -X POST https://seed.example:8443/api/v1/tftp/session \
     -H "Authorization: Bearer $SEED_TOKEN" -H 'Content-Type: application/json' \
     -d '{"interface":"eth0","allowUpload":false}'
   ```

   The token needs the admin scope.

   The response names the address the devices should use, the source
   prefixes the session answers (`allowFrom`) and when it will stop at the
   latest (`expiresAt`).
3. On the device, copy from that address, for example
   `copy tftp://192.0.2.10/c9300-universalk9.bin flash:`.
4. Stop the session with `DELETE /api/v1/tftp/session`, or let it stop itself
   when idle. `GET /api/v1/tftp/session` shows whether one is running.

## Permissions and firewalls

UDP 69 is a privileged port. The packaged service, and `seed install`, grant
Seed `CAP_NET_BIND_SERVICE` for it, because a device cannot be pointed at a
different port: IOS `copy tftp:` has no port option. When Seed cannot bind
the port, starting a session fails with `503` and says so.

The daemon holds `CAP_NET_BIND_SERVICE` for as long as it runs, not only
while a session is up. The systemd unit grants it as an ambient capability
and `seed install` sets it on the binary. Seed does not raise it per session
and drop it after: Linux capabilities belong to each thread, and changing
them across every thread of a running Go process is fragile. What the
capability allows is binding a port
below 1024, nothing more, and Seed binds 69 only when an admin starts a
session. The boundary is the session, not the capability. If you never use
TFTP and want the capability gone, remove `CAP_NET_BIND_SERVICE` from
`AmbientCapabilities` and `CapabilityBoundingSet` with a systemd drop-in;
starting a session then fails with `503`.

Seed's installer opens only its HTTPS port in the host firewall. To use TFTP
through an active firewall, allow UDP 69 and the transfer ports on the
interface you serve from, and remove the rule afterwards. TFTP moves each
transfer to a new port, so the rule needs connection tracking (the
`nf_conntrack_tftp` helper) or an allowance for high ports.

IPv6 is not served.
