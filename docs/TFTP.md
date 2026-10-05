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
- **One directory.** Only `<data dir>/tftp` is served; on a packaged Linux
  install that is `/var/lib/seed/tftp`. A request cannot leave it, through
  `..` or through a symbolic link.
- **Read-only unless you allow uploads.** With uploads allowed, a device can
  write new files into the directory. It cannot replace an existing one, and
  each upload is capped at 256 MiB.
- **Stops itself when idle.** After ten minutes with no transfer, the session
  stops. A transfer in progress keeps it open.

Starting and stopping a session needs the admin role and Seed Pro. Every
start, stop and transfer is written to the audit log with resource type
`tftp`. For a transfer, the entry records the device's address, the file name,
the bytes moved and any error. Each one is also logged as `event=tftp.start`,
`tftp.stop`, `tftp.download` or `tftp.upload`.

## Using it

1. Copy the files the devices need into the served directory.
2. Start a session on the interface that faces the devices:

   ```sh
   curl -sk -X POST https://seed.example:8443/api/v1/tftp/session \
     -H "Authorization: Bearer $SEED_TOKEN" -H 'Content-Type: application/json' \
     -d '{"interface":"eth0","allowUpload":false}'
   ```

   The token needs the admin scope.

   The response names the address the devices should use.
3. On the device, copy from that address, for example
   `copy tftp://192.0.2.10/c9300-universalk9.bin flash:`.
4. Stop the session with `DELETE /api/v1/tftp/session`, or let it stop itself
   when idle. `GET /api/v1/tftp/session` shows whether one is running.

## Permissions and firewalls

UDP 69 is a privileged port. The packaged service, and `seed install`, grant
Seed `CAP_NET_BIND_SERVICE` for it, because a device cannot be pointed at a
different port: IOS `copy tftp:` has no port option. When Seed cannot bind
the port, starting a session fails with `503` and says so.

Seed's installer opens only its HTTPS port in the host firewall. To use TFTP
through an active firewall, allow UDP 69 and the transfer ports on the
interface you serve from, and remove the rule afterwards. TFTP moves each
transfer to a new port, so the rule needs connection tracking (the
`nf_conntrack_tftp` helper) or an allowance for high ports.

IPv6 is not served.
